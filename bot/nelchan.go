package nelchanbot

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

type NelchanConfig struct {
	Env            string
	CodeSandboxURL string
}

type Nelchan struct {
	Config           NelchanConfig
	Discord          *discordgo.Session
	CommandAPIClient *CommandAPIClient
	CommandParser    *CommandParser
	CommandRouter    *CommandRouter
}

func NewNelchan() (*Nelchan, error) {
	discordToken := os.Getenv("DISCORD_BOT_TOKEN")
	if discordToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is not set")
	}

	nelchanAPIKey := os.Getenv("NELCHAN_API_KEY")
	if nelchanAPIKey == "" {
		return nil, fmt.Errorf("NELCHAN_API_KEY is not set")
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

	commandAPIClient := NewCommandAPIClient(codeSandboxURL, nelchanAPIKey)
	commandParser := NewCommandParser()
	commandRouter := NewCommandRouter(commandParser, commandAPIClient)

	n := &Nelchan{
		Config:           config,
		Discord:          discord,
		CommandAPIClient: commandAPIClient,
		CommandParser:    commandParser,
		CommandRouter:    commandRouter,
	}

	// Register built-in commands
	commandRouter.
		AddCommand("register", n.handleRegisterCommand).
		AddCommand("reg", n.handleRegisterCommand).
		AddCommand("register_code", n.handleRegisterCodeCommand).
		AddCommand("regc", n.handleRegisterCodeCommand).
		AddCommand("smart_register", n.handleSmartRegisterCommand).
		AddCommand("sreg", n.handleSmartRegisterCommand).
		AddCommand("exec", n.handleExecCommand).
		AddCommand("show", n.handleShowCommand).
		AddCommand("set_mention", n.handleSetMentionCommand).
		SetCodeFallback(n.handleDynamicCodeCommand).
		SetTextFallback(n.handleTextCommand).
		SetMentionHandler(n.handleMention)

	return n, nil
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

	// Register command router handler (handles commands and mentions)
	n.Discord.AddHandler(n.CommandRouter.Handle)

	// Register message event handlers for mllm memory enhancement
	n.Discord.AddHandler(n.handleMessageCreate)
	n.Discord.AddHandler(n.handleMessageUpdate)
	n.Discord.AddHandler(n.handleMessageDelete)

	// Set intents for guild messages
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

// handleRegisterCodeCommand handles the !register_code command
// Usage: !register_code <command_name> <code>
// Code can be plain text or wrapped in backticks (```python ... ```)
func (n *Nelchan) handleRegisterCodeCommand(s *discordgo.Session, m *discordgo.MessageCreate, _ *SlashCommand) {
	// Re-parse with body support for code commands
	cmd := n.CommandParser.ParseSlashCommandWithBody(m.Content, 2)
	if cmd == nil || len(cmd.Args) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !register_code <コマンド名> <コード>")
		return
	}

	commandName := cmd.GetArg(0)
	code := n.CommandParser.ExtractCodeFromBackticks(cmd.GetArg(1))

	if commandName == "" || code == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !register_code <コマンド名> <コード>")
		return
	}

	fmt.Printf("register_code command: name=%s, code=%s\n", commandName, code)

	err := n.CommandAPIClient.RegisterCommand(RegisterCommandRequest{
		CommandName:    commandName,
		CommandContent: code,
		IsCode:         true,
		AuthorID:       m.Author.ID,
	})
	if err != nil {
		fmt.Println("error registering command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("コードコマンド「%s」を登録しました！", commandName))
}

// handleSmartRegisterCommand handles the !sreg command
// Usage: !sreg <command_name> <description>
// Generates Python code from natural language description and registers it
func (n *Nelchan) handleSmartRegisterCommand(s *discordgo.Session, m *discordgo.MessageCreate, _ *SlashCommand) {
	// Re-parse with body support for description
	cmd := n.CommandParser.ParseSlashCommandWithBody(m.Content, 2)
	if cmd == nil || len(cmd.Args) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !sreg <コマンド名> <説明>")
		return
	}

	commandName := cmd.GetArg(0)
	description := cmd.GetArg(1)

	if commandName == "" || description == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !sreg <コマンド名> <説明>")
		return
	}

	fmt.Printf("sreg command: name=%s, description=%s\n", commandName, description)

	// Show "typing" indicator while processing
	_ = s.ChannelTyping(m.ChannelID)

	result, err := n.CommandAPIClient.SmartRegisterCommand(SmartRegisterRequest{
		CommandName: commandName,
		Description: description,
		AuthorID:    m.Author.ID,
	})
	if err != nil {
		fmt.Println("error smart registering command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	// Format success message with generated code and usage
	message := fmt.Sprintf("コマンド「%s」を登録しました！\n\n**使い方:**\n%s\n\n**生成されたコード:**\n```python\n%s\n```",
		result.CommandName,
		result.Usage,
		result.GeneratedCode)

	err = n.sendMessage(s, m.ChannelID, message)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}
}

// handleRegisterCommand handles the !register command
// Usage: !register <command_name> <text>
func (n *Nelchan) handleRegisterCommand(s *discordgo.Session, m *discordgo.MessageCreate, _ *SlashCommand) {
	// Re-parse with body support for text commands
	cmd := n.CommandParser.ParseSlashCommandWithBody(m.Content, 2)
	if cmd == nil || len(cmd.Args) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !register <コマンド名> <テキスト>")
		return
	}

	commandName := cmd.GetArg(0)
	text := cmd.GetArg(1)

	if commandName == "" || text == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !register <コマンド名> <テキスト>")
		return
	}

	fmt.Printf("register command: name=%s, text=%s\n", commandName, text)

	err := n.CommandAPIClient.RegisterCommand(RegisterCommandRequest{
		CommandName:    commandName,
		CommandContent: text,
		IsCode:         false,
		AuthorID:       m.Author.ID,
	})
	if err != nil {
		fmt.Println("error registering command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("コマンド「%s」を登録しました！", commandName))
}

func (n *Nelchan) handleExecCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmd *SlashCommand) {
	if len(cmd.Args) < 1 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !exec <コマンド名>")
		return
	}

	commandName := cmd.GetArg(0)

	if commandName == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !exec <コマンド名>")
		return
	}

	vars := map[string]string{
		"username":    m.Author.GlobalName,
		"user_id":     m.Author.ID,
		"user_avatar": m.Author.Avatar,
	}

	result, err := n.CommandAPIClient.RunCommand(RunCommandRequest{
		CommandName: commandName,
		IsCode:      true,
		Vars:        vars,
		Args:        cmd.Args,
	})
	if err != nil {
		fmt.Println("error running command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	if result == nil {
		// Code command not found, don't respond
		return
	}

	err = n.sendMessage(s, m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}

	fmt.Printf("message sent: %s\n", result.Content)
}

// handleDynamicCodeCommand handles code commands that are not registered as built-in commands
func (n *Nelchan) handleDynamicCodeCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmd *SlashCommand) {
	vars := map[string]string{
		"username":    m.Author.DisplayName(),
		"user_id":     m.Author.ID,
		"user_avatar": m.Author.Avatar,
	}

	args := cmd.Args

	result, err := n.CommandAPIClient.RunCommand(RunCommandRequest{
		CommandName: cmd.Name,
		IsCode:      true,
		Vars:        vars,
		Args:        args,
	})

	if err != nil {
		fmt.Println("error running command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	if result == nil {
		// Code command not found, don't respond
		return
	}

	err = n.sendMessage(s, m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}

	fmt.Printf("message sent: %s\n", result.Content)
}

// handleShowCommand handles the !show command
// Usage: !show <command_name>
// Displays the content of a registered command (code as snippet, text as plain text)
func (n *Nelchan) handleShowCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmd *SlashCommand) {
	if len(cmd.Args) < 1 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !show <コマンド名>")
		return
	}

	commandName := cmd.GetArg(0)

	if commandName == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "使い方: !show <コマンド名>")
		return
	}

	result, err := n.CommandAPIClient.GetCommand(GetCommandRequest{
		CommandName: commandName,
	})
	if err != nil {
		fmt.Println("error getting command,", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	if result == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("コマンド「%s」は見つかりませんでした", commandName))
		return
	}

	var message string
	if result.IsCode {
		// Output as code snippet
		message = fmt.Sprintf("コマンド「%s」のコード:\n```python\n%s\n```", commandName, result.Content)
	} else {
		// Output as plain text
		message = fmt.Sprintf("コマンド「%s」のテキスト:\n%s", commandName, result.Content)
	}

	err = n.sendMessage(s, m.ChannelID, message)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}

	fmt.Printf("show command: name=%s, isCode=%v\n", commandName, result.IsCode)
}

// handleTextCommand handles text commands (without ! prefix)
func (n *Nelchan) handleTextCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmd *SlashCommand) {
	result, err := n.CommandAPIClient.RunCommand(RunCommandRequest{
		CommandName: cmd.Name,
		IsCode:      false,
		Vars:        nil,
	})
	if err != nil {
		fmt.Println("error running text command,", err)
		return
	}

	if result == nil {
		// Text command not found, don't respond (to avoid spamming)
		return
	}

	err = n.sendMessage(s, m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}

	fmt.Printf("message sent: %s\n", result.Content)
}

// handleSetMentionCommand handles the !set_mention command
// Usage: !set_mention <command_name> - Set the mention command
// Usage: !set_mention - Show current mention command
// Usage: !set_mention clear - Clear the mention command
func (n *Nelchan) handleSetMentionCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmd *SlashCommand) {
	// No args - show current mention command
	if len(cmd.Args) == 0 {
		currentCmd, err := n.CommandAPIClient.GetMentionCommand()
		if err != nil {
			fmt.Println("error getting mention command:", err)
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
			return
		}

		if currentCmd == nil || *currentCmd == "" {
			_, _ = s.ChannelMessageSend(m.ChannelID, "メンションコマンドは設定されていません")
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("現在のメンションコマンド: `%s`", *currentCmd))
		}
		return
	}

	commandName := cmd.GetArg(0)

	// Clear command
	if commandName == "clear" {
		err := n.CommandAPIClient.SetMentionCommand(nil)
		if err != nil {
			fmt.Println("error clearing mention command:", err)
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
			return
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, "メンションコマンドをクリアしました")
		return
	}

	// Set command
	err := n.CommandAPIClient.SetMentionCommand(&commandName)
	if err != nil {
		fmt.Println("error setting mention command:", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("メンションコマンドを `%s` に設定しました", commandName))
}

// handleMention handles when the bot is mentioned
func (n *Nelchan) handleMention(s *discordgo.Session, m *discordgo.MessageCreate, args string) {
	// Get the current mention command
	mentionCmd, err := n.CommandAPIClient.GetMentionCommand()
	if err != nil {
		fmt.Println("error getting mention command:", err)
		return
	}

	// No mention command set
	if mentionCmd == nil || *mentionCmd == "" {
		fmt.Println("no mention command set, ignoring mention")
		return
	}

	fmt.Printf("executing mention command: %s with args: %s\n", *mentionCmd, args)

	// Show "typing" indicator while processing
	_ = s.ChannelTyping(m.ChannelID)

	vars := map[string]string{
		"username":    m.Author.DisplayName(),
		"user_id":     m.Author.ID,
		"user_avatar": m.Author.Avatar,
	}

	// Split args into slice
	argSlice := strings.Fields(args)

	result, err := n.CommandAPIClient.RunCommand(RunCommandRequest{
		CommandName: *mentionCmd,
		IsCode:      true,
		Vars:        vars,
		Args:        argSlice,
	})

	if err != nil {
		fmt.Println("error running mention command:", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("エラー: %s", err.Error()))
		return
	}

	if result == nil {
		fmt.Printf("mention command %s not found\n", *mentionCmd)
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("メンションコマンド `%s` が見つかりません", *mentionCmd))
		return
	}

	err = n.sendMessage(s, m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message:", err)
		return
	}

	fmt.Printf("mention command executed: %s\n", result.Content)
}

const maxMessageLength = 2000

// sendMessage sends a message to the specified channel.
// If the content exceeds Discord's 2000 character limit, it sends the content as a text file attachment.
func (n *Nelchan) sendMessage(s *discordgo.Session, channelID, content string) error {
	if utf8.RuneCountInString(content) <= maxMessageLength {
		_, err := s.ChannelMessageSend(channelID, content)
		return err
	}

	// Content is too long, send as a file attachment
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "結果が長すぎるためファイルとして送信します",
		Files: []*discordgo.File{
			{
				Name:        "result.txt",
				ContentType: "text/plain",
				Reader:      strings.NewReader(content),
			},
		},
	})
	return err
}
