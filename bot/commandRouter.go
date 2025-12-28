package nelchanbot

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"unicode"

	"github.com/bwmarrin/discordgo"
)

// CommandHandler is the function signature for command handlers
type CommandHandler func(s *discordgo.Session, m *discordgo.MessageCreate, cmd *SlashCommand)

// MentionHandler is the function signature for mention handlers
type MentionHandler func(s *discordgo.Session, m *discordgo.MessageCreate, args string)

// CommandRouter handles routing of commands to their respective handlers
type CommandRouter struct {
	parser              *CommandParser
	apiClient           *CommandAPIClient
	commands            map[string]CommandHandler
	codeFallbackHandler CommandHandler
	textFallbackHandler CommandHandler
	mentionHandler      MentionHandler
}

// NewCommandRouter creates a new CommandRouter instance
func NewCommandRouter(parser *CommandParser, apiClient *CommandAPIClient) *CommandRouter {
	return &CommandRouter{
		parser:    parser,
		apiClient: apiClient,
		commands:  make(map[string]CommandHandler),
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

// SetMentionHandler sets the handler for mention commands
func (r *CommandRouter) SetMentionHandler(handler MentionHandler) *CommandRouter {
	r.mentionHandler = handler
	return r
}

// Handle processes an incoming message and routes it to the appropriate handler
func (r *CommandRouter) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	fmt.Printf("message received: %s\n", m.Content)

	// Check if message mentions the bot
	if r.mentionHandler != nil {
		if args, ok := r.extractMentionArgs(s, m); ok {
			r.mentionHandler(s, m, args)
			return
		}
	}

	// Check if it's a command with ! prefix
	if strings.HasPrefix(m.Content, "!") {
		r.handleCodeCommand(s, m)
		return
	}

	// Handle as text command
	r.handleTextCommand(s, m)
}

// extractMentionArgs checks if the message mentions the bot and extracts the remaining text
// Returns the arguments string and true if the bot was mentioned, empty string and false otherwise
func (r *CommandRouter) extractMentionArgs(s *discordgo.Session, m *discordgo.MessageCreate) (string, bool) {
	botID := s.State.User.ID

	// Discord mention patterns: <@BOT_ID> or <@!BOT_ID>
	mentionPatterns := []string{
		fmt.Sprintf("<@%s>", botID),
		fmt.Sprintf("<@!%s>", botID),
	}

	content := m.Content
	for _, pattern := range mentionPatterns {
		if strings.Contains(content, pattern) {
			// Remove the mention and trim whitespace
			args := strings.TrimSpace(strings.Replace(content, pattern, "", 1))
			fmt.Printf("mention detected, args: %s\n", args)
			return args, true
		}
	}

	return "", false
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

	// Try to remember this message (30% chance)
	r.tryRememberMessage(m)

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

// Memory probability (30%)
const memoryProbability = 0.30

// URL pattern for detecting URL-only messages
var urlOnlyPattern = regexp.MustCompile(`^\s*https?://\S+\s*$`)

// tryRememberMessage attempts to store the message in memory with 30% probability
func (r *CommandRouter) tryRememberMessage(m *discordgo.MessageCreate) {
	content := strings.TrimSpace(m.Content)

	// Check if message should be remembered
	if !r.shouldRemember(content) {
		return
	}

	// 30% probability check
	if rand.Float64() > memoryProbability {
		return
	}

	// Get username (prefer display name, fallback to username)
	username := m.Author.GlobalName
	if username == "" {
		username = m.Author.Username
	}

	// Format: ユーザー「username」: message
	memoryText := fmt.Sprintf("ユーザー「%s」: %s", username, content)

	// Store memory asynchronously (don't block the message handler)
	go func() {
		if err := r.apiClient.AutoStoreMemory(memoryText); err != nil {
			fmt.Printf("error storing memory: %v\n", err)
		} else {
			fmt.Printf("memory stored: %s\n", memoryText)
		}
	}()
}

// shouldRemember checks if the message content should be remembered
func (r *CommandRouter) shouldRemember(content string) bool {
	// Empty message
	if content == "" {
		return false
	}

	// URL-only message
	if urlOnlyPattern.MatchString(content) {
		return false
	}

	// Emoji-only message
	if isEmojiOnly(content) {
		return false
	}

	return true
}

// isEmojiOnly checks if the string contains only emojis (including Discord custom emojis)
func isEmojiOnly(s string) bool {
	// Discord custom emoji pattern: <:name:id> or <a:name:id>
	discordEmojiPattern := regexp.MustCompile(`<a?:\w+:\d+>`)

	// Remove Discord custom emojis
	remaining := discordEmojiPattern.ReplaceAllString(s, "")
	remaining = strings.TrimSpace(remaining)

	// Check if remaining characters are all emoji or whitespace
	for _, r := range remaining {
		if unicode.IsSpace(r) {
			continue
		}
		// Check for Unicode emoji categories
		if !isEmoji(r) {
			return false
		}
	}

	return true
}

// isEmoji checks if a rune is an emoji
func isEmoji(r rune) bool {
	// Common emoji ranges
	return unicode.Is(unicode.So, r) || // Symbol, Other (includes many emojis)
		unicode.Is(unicode.Sk, r) || // Symbol, Modifier
		(r >= 0x1F300 && r <= 0x1F9FF) || // Miscellaneous Symbols and Pictographs, Emoticons, etc.
		(r >= 0x2600 && r <= 0x26FF) || // Miscellaneous Symbols
		(r >= 0x2700 && r <= 0x27BF) || // Dingbats
		(r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors
		(r >= 0x1F000 && r <= 0x1FFFF) // All emoji blocks
}
