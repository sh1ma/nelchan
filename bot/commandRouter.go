package nelchanbot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// CommandHandler is the function signature for command handlers
type CommandHandler func(s *discordgo.Session, m *discordgo.MessageCreate, cmd *SlashCommand)

// CommandRouter handles routing of commands to their respective handlers
type CommandRouter struct {
	parser              *CommandParser
	commands            map[string]CommandHandler
	codeFallbackHandler CommandHandler
	textFallbackHandler CommandHandler
}

// NewCommandRouter creates a new CommandRouter instance
func NewCommandRouter(parser *CommandParser) *CommandRouter {
	return &CommandRouter{
		parser:   parser,
		commands: make(map[string]CommandHandler),
	}
}

// AddCommand registers a command handler for the given command name
func (r *CommandRouter) AddCommand(name string, handler CommandHandler) *CommandRouter {
	r.commands[name] = handler
	return r
}

// SetCodeFallback sets the fallback handler for code commands (! prefix) that are not registered
func (r *CommandRouter) SetCodeFallback(handler CommandHandler) *CommandRouter {
	r.codeFallbackHandler = handler
	return r
}

// SetTextFallback sets the fallback handler for text commands (no ! prefix)
func (r *CommandRouter) SetTextFallback(handler CommandHandler) *CommandRouter {
	r.textFallbackHandler = handler
	return r
}

// Handle processes an incoming message and routes it to the appropriate handler
func (r *CommandRouter) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	fmt.Printf("message received: %s\n", m.Content)

	// Check if it's a command with ! prefix
	if strings.HasPrefix(m.Content, "!") {
		r.handleCodeCommand(s, m)
		return
	}

	// Handle as text command
	r.handleTextCommand(s, m)
}

// handleCodeCommand handles commands with ! prefix
func (r *CommandRouter) handleCodeCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Parse the command
	cmd := r.parser.ParseSlashCommand(m.Content)
	if cmd == nil || !cmd.IsValid() {
		return
	}

	fmt.Printf("code command received: %s\n", cmd.Name)

	// Check if there's a registered handler for this command
	if handler, exists := r.commands[cmd.Name]; exists {
		handler(s, m, cmd)
		return
	}

	// Use fallback handler for unregistered code commands
	if r.codeFallbackHandler != nil {
		r.codeFallbackHandler(s, m, cmd)
	}
}

// handleTextCommand handles commands without ! prefix
func (r *CommandRouter) handleTextCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	content := strings.TrimSpace(m.Content)
	if content == "" {
		return
	}

	// Create a SlashCommand-like structure for text commands
	parts := strings.SplitN(content, " ", 2)
	cmd := &SlashCommand{
		Name: parts[0],
		Args: []string{},
	}
	if len(parts) > 1 {
		cmd.Args = strings.Fields(parts[1])
	}

	fmt.Printf("text command received: %s\n", cmd.Name)

	// Use fallback handler for text commands
	if r.textFallbackHandler != nil {
		r.textFallbackHandler(s, m, cmd)
	}
}
