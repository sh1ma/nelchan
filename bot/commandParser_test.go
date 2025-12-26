package nelchanbot

import (
	"reflect"
	"testing"
)

func TestExtractCodeFromBackticks(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text without backticks",
			input:    "print('Hello')",
			expected: "print('Hello')",
		},
		{
			name:     "plain text with leading/trailing whitespace",
			input:    "  print('Hello')  ",
			expected: "print('Hello')",
		},
		{
			name:     "single backticks",
			input:    "`print('Hello')`",
			expected: "print('Hello')",
		},
		{
			name:     "triple backticks without language",
			input:    "```print('Hello')```",
			expected: "print('Hello')",
		},
		{
			name:     "triple backticks with language specifier",
			input:    "```python\nprint('Hello')\n```",
			expected: "print('Hello')",
		},
		{
			name:     "triple backticks with py language specifier",
			input:    "```py\nprint('Hello')\n```",
			expected: "print('Hello')",
		},
		{
			name:     "triple backticks without language but with newlines",
			input:    "```\nprint('Hello')\n```",
			expected: "print('Hello')",
		},
		{
			name:     "triple backticks with multiline code",
			input:    "```python\nx = 1\ny = 2\nprint(x + y)\n```",
			expected: "x = 1\ny = 2\nprint(x + y)",
		},
		{
			name:     "triple backticks with leading whitespace",
			input:    "  ```python\nprint('Hello')\n```  ",
			expected: "print('Hello')",
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ExtractCodeFromBackticks(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractCodeFromBackticks(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSlashCommand(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name     string
		input    string
		expected *SlashCommand
	}{
		{
			name:  "simple command without args",
			input: "!hello",
			expected: &SlashCommand{
				Name: "hello",
				Args: []string{},
			},
		},
		{
			name:  "command with single arg",
			input: "!register name",
			expected: &SlashCommand{
				Name: "register",
				Args: []string{"name"},
			},
		},
		{
			name:  "command with multiple args",
			input: "!register name value extra",
			expected: &SlashCommand{
				Name: "register",
				Args: []string{"name", "value", "extra"},
			},
		},
		{
			name:  "command with leading whitespace",
			input: "  !hello world  ",
			expected: &SlashCommand{
				Name: "hello",
				Args: []string{"world"},
			},
		},
		{
			name:     "message without prefix",
			input:    "hello world",
			expected: nil,
		},
		{
			name:     "empty message",
			input:    "",
			expected: nil,
		},
		{
			name:     "only prefix",
			input:    "!",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseSlashCommand(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ParseSlashCommand(%q) = %+v, want nil", tt.input, result)
				}
				return
			}
			if result == nil {
				t.Errorf("ParseSlashCommand(%q) = nil, want %+v", tt.input, tt.expected)
				return
			}
			if result.Name != tt.expected.Name {
				t.Errorf("ParseSlashCommand(%q).Name = %q, want %q", tt.input, result.Name, tt.expected.Name)
			}
			if !reflect.DeepEqual(result.Args, tt.expected.Args) {
				t.Errorf("ParseSlashCommand(%q).Args = %v, want %v", tt.input, result.Args, tt.expected.Args)
			}
		})
	}
}

func TestParseSlashCommandWithBody(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name     string
		input    string
		argCount int
		expected *SlashCommand
	}{
		{
			name:     "register_code with code body",
			input:    "!register_code test print('hello world')",
			argCount: 2,
			expected: &SlashCommand{
				Name: "register_code",
				Args: []string{"test", "print('hello world')"},
			},
		},
		{
			name:     "register_code with newline in body",
			input:    "!register_code test\nprint('hello')",
			argCount: 2,
			expected: &SlashCommand{
				Name: "register_code",
				Args: []string{"test", "print('hello')"},
			},
		},
		{
			name:     "command with multiple args",
			input:    "!cmd arg1 arg2 arg3 arg4",
			argCount: 4,
			expected: &SlashCommand{
				Name: "cmd",
				Args: []string{"arg1", "arg2", "arg3", "arg4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseSlashCommandWithBody(tt.input, tt.argCount)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ParseSlashCommandWithBody(%q, %d) = %+v, want nil", tt.input, tt.argCount, result)
				}
				return
			}
			if result == nil {
				t.Errorf("ParseSlashCommandWithBody(%q, %d) = nil, want %+v", tt.input, tt.argCount, tt.expected)
				return
			}
			if result.Name != tt.expected.Name {
				t.Errorf("ParseSlashCommandWithBody(%q, %d).Name = %q, want %q", tt.input, tt.argCount, result.Name, tt.expected.Name)
			}
			if !reflect.DeepEqual(result.Args, tt.expected.Args) {
				t.Errorf("ParseSlashCommandWithBody(%q, %d).Args = %v, want %v", tt.input, tt.argCount, result.Args, tt.expected.Args)
			}
		})
	}
}

func TestSlashCommandHelpers(t *testing.T) {
	cmd := &SlashCommand{
		Name: "test",
		Args: []string{"arg0", "arg1", "arg2"},
	}

	t.Run("IsValid", func(t *testing.T) {
		if !cmd.IsValid() {
			t.Error("IsValid() = false, want true")
		}
		emptyCmd := &SlashCommand{Name: "", Args: []string{}}
		if emptyCmd.IsValid() {
			t.Error("IsValid() for empty name = true, want false")
		}
	})

	t.Run("GetArg", func(t *testing.T) {
		if got := cmd.GetArg(0); got != "arg0" {
			t.Errorf("GetArg(0) = %q, want %q", got, "arg0")
		}
		if got := cmd.GetArg(2); got != "arg2" {
			t.Errorf("GetArg(2) = %q, want %q", got, "arg2")
		}
		if got := cmd.GetArg(3); got != "" {
			t.Errorf("GetArg(3) = %q, want empty string", got)
		}
		if got := cmd.GetArg(-1); got != "" {
			t.Errorf("GetArg(-1) = %q, want empty string", got)
		}
	})

	t.Run("GetArgsFrom", func(t *testing.T) {
		if got := cmd.GetArgsFrom(1); got != "arg1 arg2" {
			t.Errorf("GetArgsFrom(1) = %q, want %q", got, "arg1 arg2")
		}
		if got := cmd.GetArgsFrom(0); got != "arg0 arg1 arg2" {
			t.Errorf("GetArgsFrom(0) = %q, want %q", got, "arg0 arg1 arg2")
		}
		if got := cmd.GetArgsFrom(3); got != "" {
			t.Errorf("GetArgsFrom(3) = %q, want empty string", got)
		}
	})
}
