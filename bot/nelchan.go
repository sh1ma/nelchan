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
	BotOwnerUserID string
}

type Nelchan struct {
	Config           NelchanConfig
	Discord          *discordgo.Session
	CommandAPIClient *CommandAPIClient
	CommandParser    *CommandParser
	CommandRouter    *CommandRouter
}

// builtinSlashCommands defines the built-in slash commands to register on startup
var builtinSlashCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "register",
		Description: "テキストコマンドを登録します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "command_name",
				Description: "コマンド名",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "text",
				Description: "登録するテキスト",
				Required:    true,
			},
		},
	},
	{
		Name:        "set_mention",
		Description: "メンション時に実行するコマンドを設定します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "command_name",
				Description: "コマンド名（省略で現在の設定を表示、clearでクリア）",
				Required:    false,
			},
		},
	},
	{
		Name:        "reset-slash-commands",
		Description: "【管理者専用】全てのスラッシュコマンドを削除します",
	},
	{
		Name:        "register-builtin-commands",
		Description: "【管理者専用】ビルトインコマンドを再登録します",
	},
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

	botOwnerUserID := os.Getenv("BOT_OWNER_USER_ID")

	config := NelchanConfig{
		Env:            env,
		CodeSandboxURL: codeSandboxURL,
		BotOwnerUserID: botOwnerUserID,
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

	// Register interaction handler for slash commands
	n.Discord.AddHandler(n.handleInteraction)

	// Register ready handler to register slash commands on startup
	n.Discord.AddHandler(n.handleReady)

	// Set intents for guild messages
	n.SetIntents(discordgo.IntentsGuildMessages)

	err := n.Discord.Open()
	if err != nil {
		return fmt.Errorf("ねるちゃんの起動に失敗しました: %w", err)
	}
	return nil
}

// handleReady is called when the bot is ready and registers built-in slash commands
func (n *Nelchan) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Printf("ねるちゃんが起動しました！ユーザー: %s#%s\n", r.User.Username, r.User.Discriminator)
	fmt.Printf("参加ギルド数: %d\n", len(r.Guilds))

	// Register built-in slash commands globally
	n.registerBuiltinSlashCommands(s)
}

// registerBuiltinSlashCommands registers built-in slash commands globally
// It will update existing commands if they have different options
func (n *Nelchan) registerBuiltinSlashCommands(s *discordgo.Session) {
	// Get existing global commands
	existingCmds, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		fmt.Printf("error getting existing global commands: %v\n", err)
		return
	}

	// Create a map of existing commands by name
	existingByName := make(map[string]*discordgo.ApplicationCommand)
	for _, cmd := range existingCmds {
		existingByName[cmd.Name] = cmd
	}

	// Register or update each built-in command
	for _, cmd := range builtinSlashCommands {
		existing, exists := existingByName[cmd.Name]
		if exists {
			// Check if options count differs (simple check for update)
			if len(existing.Options) != len(cmd.Options) {
				// Update the command
				_, err := s.ApplicationCommandEdit(s.State.User.ID, "", existing.ID, cmd)
				if err != nil {
					fmt.Printf("error updating global slash command /%s: %v\n", cmd.Name, err)
					continue
				}
				fmt.Printf("updated global slash command /%s (options: %d -> %d)\n", cmd.Name, len(existing.Options), len(cmd.Options))
			} else {
				fmt.Printf("global slash command /%s already exists with same options, skipping\n", cmd.Name)
			}
			continue
		}

		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			fmt.Printf("error registering global slash command /%s: %v\n", cmd.Name, err)
			continue
		}
		fmt.Printf("registered global slash command /%s\n", cmd.Name)
	}
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
// If code contains "# args = [...]" comment, registers as Discord slash command
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

	// Check for args comment and register as slash command if present
	args := n.CommandParser.ExtractArgsFromComment(code)
	if args != nil {
		err := n.registerSlashCommand(s, m.GuildID, commandName, args)
		if err != nil {
			fmt.Printf("error registering slash command: %v\n", err)
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("コードコマンド「%s」を登録しました！\n⚠️ スラッシュコマンドの登録に失敗: %s", commandName, err.Error()))
			return
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("コードコマンド「%s」を登録しました！（スラッシュコマンド対応）", commandName))
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("コードコマンド「%s」を登録しました！", commandName))
}

// registerSlashCommand registers a Discord application command for the given code command
func (n *Nelchan) registerSlashCommand(s *discordgo.Session, guildID, commandName string, args []ArgOption) error {
	options := make([]*discordgo.ApplicationCommandOption, len(args))
	for i, arg := range args {
		optionType := argTypeToDiscordType(arg.Type)
		description := arg.Description
		if description == "" {
			description = arg.Name // Use name as fallback description
		}
		options[i] = &discordgo.ApplicationCommandOption{
			Type:        optionType,
			Name:        arg.Name,
			Description: description,
			Required:    arg.Required,
		}
	}

	appCmd := &discordgo.ApplicationCommand{
		Name:        commandName,
		Description: fmt.Sprintf("ねるちゃんコマンド: %s", commandName),
		Options:     options,
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, appCmd)
	if err != nil {
		return fmt.Errorf("failed to create application command: %w", err)
	}

	fmt.Printf("registered slash command: /%s with %d options\n", commandName, len(options))
	return nil
}

// argTypeToDiscordType converts ArgOption type string to Discord ApplicationCommandOptionType
func argTypeToDiscordType(t string) discordgo.ApplicationCommandOptionType {
	switch t {
	case "string":
		return discordgo.ApplicationCommandOptionString
	case "number", "integer":
		return discordgo.ApplicationCommandOptionInteger
	case "boolean", "bool":
		return discordgo.ApplicationCommandOptionBoolean
	default:
		return discordgo.ApplicationCommandOptionString
	}
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
		"channel_id":  m.ChannelID,
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
		"username":    m.Author.GlobalName,
		"user_id":     m.Author.ID,
		"user_avatar": m.Author.Avatar,
		"channel_id":  m.ChannelID,
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
	// Silently ignore errors and not found (404) for text commands
	if err != nil || result == nil {
		return
	}

	fmt.Printf("text command fired: %s\n", cmd.Name)

	err = n.sendMessage(s, m.ChannelID, result.Content)
	if err != nil {
		fmt.Println("error sending message,", err)
		return
	}
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
		"username":    m.Author.GlobalName,
		"user_id":     m.Author.ID,
		"user_avatar": m.Author.Avatar,
		"channel_id":  m.ChannelID,
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

// handleInteraction handles Discord slash command interactions
func (n *Nelchan) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	commandName := data.Name

	// Handle built-in commands
	switch commandName {
	case "register":
		n.handleRegisterSlashCommand(s, i)
		return
	case "set_mention":
		n.handleSetMentionSlashCommand(s, i)
		return
	case "reset-slash-commands":
		n.handleResetSlashCommandsCommand(s, i)
		return
	case "register-builtin-commands":
		n.handleRegisterBuiltinCommandsCommand(s, i)
		return
	}

	// Handle dynamic code commands
	n.handleDynamicSlashCommand(s, i)
}

// handleRegisterSlashCommand handles the /register slash command
func (n *Nelchan) handleRegisterSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	// Debug: log all options
	fmt.Printf("DEBUG /register: options count=%d\n", len(data.Options))
	for idx, opt := range data.Options {
		fmt.Printf("DEBUG /register: option[%d] name=%s type=%d value=%v\n", idx, opt.Name, opt.Type, opt.Value)
	}

	// Get user info for author_id
	var user *discordgo.User
	if i.Member != nil {
		user = i.Member.User
	} else {
		user = i.User
	}

	// Extract options
	var commandNameOpt, textOpt string
	for _, opt := range data.Options {
		switch opt.Name {
		case "command_name":
			commandNameOpt = opt.StringValue()
		case "text":
			textOpt = opt.StringValue()
		}
	}
	fmt.Printf("DEBUG /register: commandNameOpt=%s, textOpt=%s\n", commandNameOpt, textOpt)

	// Register the command
	err := n.CommandAPIClient.RegisterCommand(RegisterCommandRequest{
		CommandName:    commandNameOpt,
		CommandContent: textOpt,
		IsCode:         false,
		AuthorID:       user.ID,
	})

	if err != nil {
		fmt.Printf("error registering command via slash: %v\n", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("エラー: %s", err.Error()),
			},
		})
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("コマンド「%s」を登録しました！", commandNameOpt),
		},
	})
	fmt.Printf("slash command /register executed: name=%s\n", commandNameOpt)
}

// handleSetMentionSlashCommand handles the /set_mention slash command
func (n *Nelchan) handleSetMentionSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	// Extract command_name option (optional)
	var commandNameOpt string
	for _, opt := range data.Options {
		if opt.Name == "command_name" {
			commandNameOpt = opt.StringValue()
		}
	}

	// No args - show current mention command
	if commandNameOpt == "" {
		currentCmd, err := n.CommandAPIClient.GetMentionCommand()
		if err != nil {
			fmt.Println("error getting mention command:", err)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("エラー: %s", err.Error()),
				},
			})
			return
		}

		var content string
		if currentCmd == nil || *currentCmd == "" {
			content = "メンションコマンドは設定されていません"
		} else {
			content = fmt.Sprintf("現在のメンションコマンド: `%s`", *currentCmd)
		}

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
			},
		})
		return
	}

	// Clear command
	if commandNameOpt == "clear" {
		err := n.CommandAPIClient.SetMentionCommand(nil)
		if err != nil {
			fmt.Println("error clearing mention command:", err)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("エラー: %s", err.Error()),
				},
			})
			return
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "メンションコマンドをクリアしました",
			},
		})
		return
	}

	// Set command
	err := n.CommandAPIClient.SetMentionCommand(&commandNameOpt)
	if err != nil {
		fmt.Println("error setting mention command:", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("エラー: %s", err.Error()),
			},
		})
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("メンションコマンドを `%s` に設定しました", commandNameOpt),
		},
	})
	fmt.Printf("slash command /set_mention executed: name=%s\n", commandNameOpt)
}

// handleDynamicSlashCommand handles dynamically registered code commands via slash
func (n *Nelchan) handleDynamicSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	commandName := data.Name

	// Extract args from interaction options
	args := make([]string, len(data.Options))
	for idx, opt := range data.Options {
		args[idx] = fmt.Sprintf("%v", opt.Value)
	}

	// Get user info
	var user *discordgo.User
	if i.Member != nil {
		user = i.Member.User
	} else {
		user = i.User
	}

	vars := map[string]string{
		"username":    user.GlobalName,
		"user_id":     user.ID,
		"user_avatar": user.Avatar,
	}

	// Defer response to allow longer processing time
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		fmt.Printf("error deferring interaction response: %v\n", err)
		return
	}

	// Run the command
	result, err := n.CommandAPIClient.RunCommand(RunCommandRequest{
		CommandName: commandName,
		IsCode:      true,
		Vars:        vars,
		Args:        args,
	})

	if err != nil {
		fmt.Printf("error running slash command: %v\n", err)
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(fmt.Sprintf("エラー: %s", err.Error())),
		})
		return
	}

	if result == nil {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(fmt.Sprintf("コマンド「%s」は見つかりませんでした", commandName)),
		})
		return
	}

	// Edit the deferred response with the result
	content := result.Content
	if utf8.RuneCountInString(content) > maxMessageLength {
		content = content[:maxMessageLength-3] + "..."
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		fmt.Printf("error editing interaction response: %v\n", err)
	}

	fmt.Printf("slash command executed: /%s\n", commandName)
}

func stringPtr(s string) *string {
	return &s
}

// handleResetSlashCommandsCommand handles the /reset-slash-commands slash command (owner only)
func (n *Nelchan) handleResetSlashCommandsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get user info
	var user *discordgo.User
	if i.Member != nil {
		user = i.Member.User
	} else {
		user = i.User
	}

	// Check if user is the bot owner
	if n.Config.BotOwnerUserID == "" || user.ID != n.Config.BotOwnerUserID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "このコマンドはBot管理者のみ実行できます",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Defer response as this may take a while
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("error deferring interaction response: %v\n", err)
		return
	}

	guildID := i.GuildID
	deletedGuild := 0
	deletedGlobal := 0

	// Delete guild commands
	guildCommands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(fmt.Sprintf("ギルドコマンド取得エラー: %s", err.Error())),
		})
		return
	}

	for _, cmd := range guildCommands {
		if cmd.Name == "reset-slash-commands" {
			fmt.Printf("skipping deletion of /%s (guild)\n", cmd.Name)
			continue
		}
		err := s.ApplicationCommandDelete(s.State.User.ID, guildID, cmd.ID)
		if err != nil {
			fmt.Printf("error deleting guild command /%s: %v\n", cmd.Name, err)
			continue
		}
		fmt.Printf("deleted guild slash command /%s from guild %s\n", cmd.Name, guildID)
		deletedGuild++
	}

	// Delete global commands
	globalCommands, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(fmt.Sprintf("ギルドコマンド%d個削除。グローバルコマンド取得エラー: %s", deletedGuild, err.Error())),
		})
		return
	}

	for _, cmd := range globalCommands {
		if cmd.Name == "reset-slash-commands" {
			fmt.Printf("skipping deletion of /%s (global)\n", cmd.Name)
			continue
		}
		err := s.ApplicationCommandDelete(s.State.User.ID, "", cmd.ID)
		if err != nil {
			fmt.Printf("error deleting global command /%s: %v\n", cmd.Name, err)
			continue
		}
		fmt.Printf("deleted global slash command /%s\n", cmd.Name)
		deletedGlobal++
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr(fmt.Sprintf("ギルドコマンド%d個、グローバルコマンド%d個を削除しました。Botを再起動するとビルトインコマンドが再登録されます。", deletedGuild, deletedGlobal)),
	})
	fmt.Printf("reset-slash-cmd executed by %s: deleted %d guild + %d global commands\n", user.Username, deletedGuild, deletedGlobal)
}

// handleRegisterBuiltinCommandsCommand handles the /register-builtin-commands slash command (owner only)
func (n *Nelchan) handleRegisterBuiltinCommandsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get user info
	var user *discordgo.User
	if i.Member != nil {
		user = i.Member.User
	} else {
		user = i.User
	}

	// Check if user is the bot owner
	if n.Config.BotOwnerUserID == "" || user.ID != n.Config.BotOwnerUserID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "このコマンドはBot管理者のみ実行できます",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Register builtin commands globally
	n.registerBuiltinSlashCommands(s)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "ビルトインコマンドをグローバルに再登録しました",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	fmt.Printf("register-builtin-commands executed by %s\n", user.Username)
}
