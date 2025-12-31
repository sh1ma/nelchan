package nelchanbot

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ArgOption represents a single argument option for slash commands
type ArgOption struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "string", "number", "boolean"
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// argsCommentRe matches lines like: # args = [...]
var argsCommentRe = regexp.MustCompile(`^\s*#\s*args\s*=\s*(.+?)\s*$`)

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
// Example: "!register_code name print('hello\nworld')" -> SlashCommand{Name: "register_code", Args: ["name", "print('hello\nworld')"]}
// The last argument (body) preserves newlines, useful for code commands
func (p *CommandParser) ParseSlashCommandWithBody(message string, argCount int) *SlashCommand {
	message = strings.TrimSpace(message)

	// Check if it starts with "!"
	if !strings.HasPrefix(message, "!") {
		return nil
	}

	// Remove the "!" prefix
	content := strings.TrimPrefix(message, "!")

	// Find the first line to extract command name and initial args
	firstLineEnd := strings.Index(content, "\n")
	var firstLine, rest string
	if firstLineEnd == -1 {
		firstLine = content
		rest = ""
	} else {
		firstLine = content[:firstLineEnd]
		rest = content[firstLineEnd+1:] // Preserve remaining content with newlines
	}

	// Split the first line by space with limit
	parts := strings.SplitN(firstLine, " ", argCount+1)
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

	// If we have remaining content (after newline), append it to the last argument
	if rest != "" {
		if len(args) > 0 {
			// Append rest to the last argument
			args[len(args)-1] = args[len(args)-1] + "\n" + rest
		} else {
			// No args yet, rest becomes the first arg
			args = append(args, rest)
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

// ExtractArgsFromComment extracts args definition from a comment line in code
// Expected format: # args = [{"name": "arg1", "type": "string"}, ...]
// Returns nil if no args comment is found or if parsing fails
func (p *CommandParser) ExtractArgsFromComment(code string) []ArgOption {
	for _, line := range strings.Split(code, "\n") {
		matches := argsCommentRe.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		raw := matches[1]
		var args []ArgOption
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			return nil
		}
		return args
	}
	return nil
}
