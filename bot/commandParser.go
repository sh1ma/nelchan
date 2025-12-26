package nelchanbot

import (
	"regexp"
	"strings"
)

type CommandParser struct{}

// SlashCommand represents a parsed slash command
type SlashCommand struct {
	Name string   // Command name without the prefix (e.g., "register" from "!register")
	Args []string // Arguments passed to the command
}

// IsValid returns true if the command has a valid name
func (c *SlashCommand) IsValid() bool {
	return c.Name != ""
}

// GetArg returns the argument at the specified index, or empty string if not found
func (c *SlashCommand) GetArg(index int) string {
	if index < 0 || index >= len(c.Args) {
		return ""
	}
	return c.Args[index]
}

// GetArgsFrom returns all arguments from the specified index onwards joined by space
func (c *SlashCommand) GetArgsFrom(index int) string {
	if index < 0 || index >= len(c.Args) {
		return ""
	}
	return strings.Join(c.Args[index:], " ")
}

func NewCommandParser() *CommandParser {
	return &CommandParser{}
}

// ParseSlashCommand parses a message starting with "!" into a SlashCommand
// Example: "!register name value" -> SlashCommand{Name: "register", Args: ["name", "value"]}
// Returns nil if the message doesn't start with "!"
func (p *CommandParser) ParseSlashCommand(message string) *SlashCommand {
	message = strings.TrimSpace(message)

	// Check if it starts with "!"
	if !strings.HasPrefix(message, "!") {
		return nil
	}

	// Remove the "!" prefix
	content := strings.TrimPrefix(message, "!")

	// Split by whitespace
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return nil
	}

	return &SlashCommand{
		Name: parts[0],
		Args: parts[1:],
	}
}

// ParseSlashCommandWithBody parses a slash command where the body may contain spaces or newlines
// Example: "!register_code name print('hello world')" -> SlashCommand{Name: "register_code", Args: ["name", "print('hello world')"]}
// The second argument onwards is treated as a single body (useful for code commands)
func (p *CommandParser) ParseSlashCommandWithBody(message string, argCount int) *SlashCommand {
	message = strings.TrimSpace(message)

	// Check if it starts with "!"
	if !strings.HasPrefix(message, "!") {
		return nil
	}

	// Remove the "!" prefix
	content := strings.TrimPrefix(message, "!")

	// Replace newlines with spaces to treat them as separators
	// This allows "!cmd arg1\narg2" to be parsed as "!cmd arg1 arg2"
	normalizedContent := strings.ReplaceAll(content, "\n", " ")

	// Split by space with limit
	parts := strings.SplitN(normalizedContent, " ", argCount+1)
	if len(parts) == 0 {
		return nil
	}

	args := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		trimmed := strings.TrimSpace(parts[i])
		if trimmed != "" {
			args = append(args, trimmed)
		}
	}

	return &SlashCommand{
		Name: strings.TrimSpace(parts[0]),
		Args: args,
	}
}

// ExtractCodeFromBackticks extracts code from markdown code blocks
// Supports the following formats:
// - ```code``` (inline triple backticks)
// - ```py\ncode\n``` (with language specifier)
// - ```\ncode\n``` (without language specifier)
// - `code` (single backticks)
// - code (plain text, no backticks)
func (p *CommandParser) ExtractCodeFromBackticks(content string) string {
	content = strings.TrimSpace(content)

	// Case 1: Triple backticks with language specifier and newline
	// Matches ```python\ncode\n``` or ```\ncode\n```
	tripleBacktickWithNewlineRe := regexp.MustCompile("(?s)^```\\w*\\n(.*?)\\n?```$")
	matches := tripleBacktickWithNewlineRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Case 2: Triple backticks without newline (inline)
	// Matches ```code```
	tripleBacktickInlineRe := regexp.MustCompile("^```([^`]+)```$")
	matches = tripleBacktickInlineRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Case 3: Single backticks
	// Matches `code`
	singleBacktickRe := regexp.MustCompile("^`([^`]+)`$")
	matches = singleBacktickRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Case 4: Plain text (no backticks)
	return content
}
