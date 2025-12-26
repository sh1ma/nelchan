package nelchanbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type NelchanConfig struct {
	Env            string
	CodeSandboxURL string
}

type Nelchan struct {
	Config  NelchanConfig
	Discord *discordgo.Session
}

func NewNelchan() (*Nelchan, error) {
	discordToken := os.Getenv("DISCORD_BOT_TOKEN")
	if discordToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is not set")
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	var codeSandboxURL string
	if env == "production" {
		codeSandboxURL = "https://my-sandbox.sh1ma.workers.dev"
	} else {
		codeSandboxURL = "http://localhost:8787"
	}

	discord, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	config := NelchanConfig{
		Env:            env,
		CodeSandboxURL: codeSandboxURL,
	}

	return &Nelchan{
		Config:  config,
		Discord: discord,
	}, nil
}

func (n *Nelchan) PrintConfig() {
	fmt.Println("ねるちゃんの設定:")
	fmt.Println("Env:", n.Config.Env)
	fmt.Println("CodeSandboxURL:", n.Config.CodeSandboxURL)
}

func (n *Nelchan) SetIntents(intents discordgo.Intent) {
	n.Discord.Identify.Intents = intents
}

func (n *Nelchan) Start() error {
	n.PrintConfig()
	n.Discord.AddHandler(n.HandleMessageCreate)
	n.SetIntents(discordgo.IntentsGuildMessages)

	err := n.Discord.Open()
	if err != nil {
		return fmt.Errorf("ねるちゃんの起動に失敗しました: %w", err)
	}
	return nil
}

func (n *Nelchan) Close() error {
	fmt.Println("ねるちゃんを停止します...")
	err := n.Discord.Close()
	if err != nil {
		return fmt.Errorf("ねるちゃんの停止に失敗しました: %w", err)
	}
	fmt.Println("ねるちゃんを停止しました")
	fmt.Println("プログラムを終了します...")
	return nil
}

func (n *Nelchan) HandleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	fmt.Printf("message received: %s\n", m.Content)

	// Handle commands starting with "!"
	if strings.HasPrefix(m.Content, "!") {
		// Handle !register_code command
		if strings.HasPrefix(m.Content, "!register_code ") {
			handleRegisterCodeCommand(s, m)
			return
		}

		// Handle !register command
		if strings.HasPrefix(m.Content, "!register ") {
			handleRegisterCommand(s, m)
			return
		}

		if strings.HasPrefix(m.Content, "!exec") {
			handleExecCommand(s, m)
			return
		}

		// Handle code commands (with ! prefix)
		commandName := strings.TrimPrefix(m.Content, "!")
		// Split by space to get just the command name (ignore arguments for now)
		commandParts := strings.Split(commandName, " ")
		commandName = commandParts[0]

		if commandName == "" {
			return
		}

		fmt.Printf("code command received: %s\n", commandName)

		vars := map[string]string{
			"username":    m.Author.DisplayName(),
			"user_id":     m.Author.ID,
			"user_avatar": m.Author.Avatar,
		}
		// varsにargsを追加
		// arg1, arg2, arg3, ...
		for i := 1; i < len(commandParts); i++ {
			vars[fmt.Sprintf("arg%d", i)] = commandParts[i]
		}
		result, err := runCommandAPI(commandName, true, vars)

		if err != nil {
			fmt.Println("error running command,", err)
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
			return
		}

		if result == nil {
			// Code command not found, don't respond
			return
		}

		_, err = s.ChannelMessageSend(m.ChannelID, result.Content)
		if err != nil {
			fmt.Println("error sending message,", err)
			return
		}

		fmt.Printf("message sent: %s\n", result.Content)
		return
	}

	// Handle text commands (without ! prefix)
	commandName := strings.TrimSpace(m.Content)
	// Split by space to get just the command name (ignore arguments for now)
	commandParts := strings.SplitN(commandName, " ", 2)
	commandName = commandParts[0]

	if commandName == "" {
		return
	}

	fmt.Printf("text command received: %s\n", commandName)

	result, err := runCommandAPI(commandName, false, nil)
	if err != nil {
		fmt.Println("error running text command,", err)
		return
	}

	if result == nil {
		// Text command not found, don't respond (to avoid spamming)
		return
	}

	_, err = s.ChannelMessageSend(m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}

	fmt.Printf("message sent: %s\n", result.Content)
}

// handleRegisterCodeCommand handles the !register_code command
// Usage: !register_code <command_name> <code>
// Code can be plain text or wrapped in backticks (```python ... ```)
func handleRegisterCodeCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Remove the "!register_code " prefix
	content := strings.TrimPrefix(m.Content, "!register_code ")

	// Split by space or newline to get command name and code
	parts := strings.SplitN(content, " ", 2)
	if len(parts) < 2 {
		// Try splitting by newline
		parts = strings.SplitN(content, "\n", 2)
	}

	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !register_code <コマンド名> <コード>")
		return
	}

	commandName := strings.TrimSpace(parts[0])
	code := strings.TrimSpace(parts[1])

	// Extract code from backticks if present
	code = extractCodeFromBackticks(code)

	if commandName == "" || code == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !register_code <コマンド名> <コード>")
		return
	}

	fmt.Printf("register_code command: name=%s, code=%s\n", commandName, code)

	err := registerCommandAPI(commandName, code, true, m.Author.ID)
	if err != nil {
		fmt.Println("error registering command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("コードコマンド「%s」を登録しました！", commandName))
}

// handleRegisterCommand handles the !register command
// Usage: !register <command_name> <text>
func handleRegisterCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Remove the "!register " prefix
	content := strings.TrimPrefix(m.Content, "!register ")

	// Split by space or newline to get command name and text
	parts := strings.SplitN(content, " ", 2)
	if len(parts) < 2 {
		// Try splitting by newline
		parts = strings.SplitN(content, "\n", 2)
	}

	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !register <コマンド名> <テキスト>")
		return
	}

	commandName := strings.TrimSpace(parts[0])
	text := strings.TrimSpace(parts[1])

	if commandName == "" || text == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !register <コマンド名> <テキスト>")
		return
	}

	fmt.Printf("register command: name=%s, text=%s\n", commandName, text)

	err := registerCommandAPI(commandName, text, false, m.Author.ID)
	if err != nil {
		fmt.Println("error registering command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("コマンド「%s」を登録しました！", commandName))
}

func handleExecCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Remove the "!exec " prefix
	content := strings.TrimPrefix(m.Content, "!exec ")

	// Split by space to get command name and code
	parts := strings.SplitN(content, " ", 2)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !exec <コマンド名> <コード>")
		return
	}

	commandName := strings.TrimSpace(parts[0])
	code := strings.TrimSpace(parts[1])

	if commandName == "" || code == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !exec <コマンド名> <コード>")
		return
	}

	result, err := runCommandAPI(commandName, true, nil)
	if err != nil {
		fmt.Println("error running command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	if result == nil {
		// Code command not found, don't respond
		return
	}

	_, err = s.ChannelMessageSend(m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}

	fmt.Printf("message sent: %s\n", result.Content)
}

// extractCodeFromBackticks extracts code from markdown code blocks
// Supports the following formats:
// - ```code``` (inline triple backticks)
// - ```py\ncode\n``` (with language specifier)
// - ```\ncode\n``` (without language specifier)
// - `code` (single backticks)
// - code (plain text, no backticks)
func extractCodeFromBackticks(content string) string {
	content = strings.TrimSpace(content)

	// Case 1: Triple backticks with optional language specifier
	// Matches ```python\ncode\n``` or ```\ncode\n``` or ```code```
	tripleBacktickRe := regexp.MustCompile("(?s)^```(?:\\w*)?\\s*\\n?(.*?)\\n?```$")
	matches := tripleBacktickRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Case 2: Single backticks
	// Matches `code`
	singleBacktickRe := regexp.MustCompile("^`([^`]+)`$")
	matches = singleBacktickRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Case 3: Plain text (no backticks)
	return content
}

type CodeSandboxResponse struct {
	Result ExecutionResult `json:"result"`
}

type RunCommandRequest struct {
	CommandName string            `json:"command_name"`
	IsCode      bool              `json:"isCode"`
	Vars        map[string]string `json:"vars"`
}

type RegisterCommandRequest struct {
	CommandName    string `json:"command_name"`
	CommandContent string `json:"command_content"`
	IsCode         bool   `json:"isCode"`
	AuthorID       string `json:"author_id"`
}

type RegisterCommandResponse struct {
	Error *string `json:"error"`
}

func registerCommandAPI(commandName, commandContent string, isCode bool, authorID string) error {
	requestBody := RegisterCommandRequest{
		CommandName:    commandName,
		CommandContent: commandContent,
		IsCode:         isCode,
		AuthorID:       authorID,
	}
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshalling request body: %w", err)
	}

	env := os.Getenv("ENV")
	var CodeSandboxURL string
	if env == "production" {
		CodeSandboxURL = "https://my-sandbox.sh1ma.workers.dev"
	} else {
		CodeSandboxURL = "http://localhost:8787"
	}

	request, err := http.NewRequest("POST", CodeSandboxURL+"/register_command", bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	fmt.Printf("register_command response body: %s\n", string(responseBody))

	if response.StatusCode == 409 {
		return fmt.Errorf("コマンド「%s」は既に存在します", commandName)
	}

	if response.StatusCode != 200 {
		var registerResponse RegisterCommandResponse
		err = json.Unmarshal(responseBody, &registerResponse)
		if err == nil && registerResponse.Error != nil {
			return fmt.Errorf("%s", *registerResponse.Error)
		}
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	return nil
}

type RunCommandResponse struct {
	Error   *string        `json:"error"`
	Command *CommandResult `json:"command"`
}

type CommandResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func runCommandAPI(commandName string, isCode bool, vars map[string]string) (*CommandResult, error) {
	requestBody := RunCommandRequest{
		CommandName: commandName,
		IsCode:      isCode,
		Vars:        vars,
	}
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	env := os.Getenv("ENV")
	var CodeSandboxURL string
	if env == "production" {
		CodeSandboxURL = "https://my-sandbox.sh1ma.workers.dev"
	} else {
		CodeSandboxURL = "http://localhost:8787"
	}

	request, err := http.NewRequest("POST", CodeSandboxURL+"/run_command", bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	fmt.Printf("run_command response body: %s\n", string(responseBody))

	if response.StatusCode == 404 {
		return nil, nil
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var runCommandResponse RunCommandResponse
	err = json.Unmarshal(responseBody, &runCommandResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return runCommandResponse.Command, nil
}

type ExecutionResult struct {
	Code string `json:"code"`
	Logs struct {
		Stdout []string `json:"stdout"`
		Stderr []string `json:"stderr"`
	} `json:"logs"`
	Error          *ExecutionError `json:"error"`
	ExecutionCount int             `json:"executionCount"`
	Results        []Result        `json:"results"`
}

type ExecutionError struct {
	Message    string   `json:"message"`
	Traceback  []string `json:"traceback"`
	LineNumber int      `json:"lineNumber"`
}

type Result struct {
	Text       string `json:"text"`
	Html       string `json:"html"`
	Png        string `json:"png"`
	Jpeg       string `json:"jpeg"`
	Svg        string `json:"svg"`
	Latex      string `json:"latex"`
	Markdown   string `json:"markdown"`
	Javascript string `json:"javascript"`
	Json       string `json:"json"`
	Chart      string `json:"chart"`
	Data       string `json:"data"`
}
