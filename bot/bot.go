package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token string
)

func main() {
	Token = os.Getenv("DISCORD_BOT_TOKEN")
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

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

		// Handle code commands (with ! prefix)
		commandName := strings.TrimPrefix(m.Content, "!")
		// Split by space to get just the command name (ignore arguments for now)
		commandParts := strings.SplitN(commandName, " ", 2)
		commandName = commandParts[0]

		if commandName == "" {
			return
		}

		fmt.Printf("code command received: %s\n", commandName)

		result, err := runCommandAPI(commandName, true)
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

	result, err := runCommandAPI(commandName, false)
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
	CommandName string `json:"command_name"`
	IsCode      bool   `json:"isCode"`
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

func runCommandAPI(commandName string, isCode bool) (*CommandResult, error) {
	requestBody := RunCommandRequest{
		CommandName: commandName,
		IsCode:      isCode,
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

// type Result struct {
// interface ExecutionResult {
//   code: string;
//   logs: {
//     stdout: string[];
//     stderr: string[];
//   };
//   error?: ExecutionError;
//   executionCount?: number;
//   results: Array<{
//     text?: string;
//     html?: string;
//     png?: string;
//     jpeg?: string;
//     svg?: string;
//     latex?: string;
//     markdown?: string;
//     javascript?: string;
//     json?: any;
//     chart?: ChartData;
//     data?: any;
//   }>;
// }
