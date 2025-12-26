package nelchanbot

import (
	"fmt"
	"os"

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

	commandAPIClient := NewCommandAPIClient(codeSandboxURL)
	commandParser := NewCommandParser()
	commandRouter := NewCommandRouter(commandParser)

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
		AddCommand("register_code", n.handleRegisterCodeCommand).
		AddCommand("exec", n.handleExecCommand).
		SetCodeFallback(n.handleDynamicCodeCommand).
		SetTextFallback(n.handleTextCommand)

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
	n.Discord.AddHandler(n.CommandRouter.Handle)
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

	result, err := n.CommandAPIClient.RunCommand(RunCommandRequest{
		CommandName: commandName,
		IsCode:      true,
		Vars:        nil,
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

	_, err = s.ChannelMessageSend(m.ChannelID, result.Content)
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
	// varsにargsを追加
	// arg1, arg2, arg3, ...
	for i, arg := range cmd.Args {
		vars[fmt.Sprintf("arg%d", i+1)] = arg
	}

	result, err := n.CommandAPIClient.RunCommand(RunCommandRequest{
		CommandName: cmd.Name,
		IsCode:      true,
		Vars:        vars,
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

	_, err = s.ChannelMessageSend(m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}

	fmt.Printf("message sent: %s\n", result.Content)
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

	_, err = s.ChannelMessageSend(m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}

	fmt.Printf("message sent: %s\n", result.Content)
}
