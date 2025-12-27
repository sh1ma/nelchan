package nelchanbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CommandAPIClient struct {
	CodeSandboxURL string
}

func NewCommandAPIClient(codeSandboxURL string) *CommandAPIClient {
	return &CommandAPIClient{
		CodeSandboxURL: codeSandboxURL,
	}
}

type RegisterCommandRequest struct {
	CommandName    string `json:"command_name"`
	CommandContent string `json:"command_content"`
	IsCode         bool   `json:"isCode"`
	AuthorID       string `json:"author_id"`
}

func (c *CommandAPIClient) RegisterCommand(request RegisterCommandRequest) error {
	url := c.CodeSandboxURL + "/register_command"

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(respBody))
	}

	var registerResponse RegisterCommandResponse
	if err := json.Unmarshal(respBody, &registerResponse); err != nil {
		return fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if registerResponse.Error != nil {
		return fmt.Errorf("API error: %s", *registerResponse.Error)
	}

	return nil
}

type RunCommandRequest struct {
	CommandName string            `json:"command_name"`
	IsCode      bool              `json:"is_code"`
	Vars        map[string]string `json:"vars"`
	Args        []string          `json:"args"`
}

func (c *CommandAPIClient) RunCommand(request RunCommandRequest) (*CommandResult, error) {
	url := c.CodeSandboxURL + "/run_command"

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var runCommandResponse RunCommandResponse
	err = json.Unmarshal(respBody, &runCommandResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return runCommandResponse.Command, nil
}

type RunCommandResponse struct {
	Error   *string        `json:"error"`
	Command *CommandResult `json:"command"`
}

type RegisterCommandResponse struct {
	Error *string `json:"error"`
}

type CommandResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type GetCommandRequest struct {
	CommandName string `json:"command_name"`
}

type GetCommandResponse struct {
	Error   *string         `json:"error"`
	Command *GetCommandInfo `json:"command"`
}

type GetCommandInfo struct {
	Name    string `json:"name"`
	IsCode  bool   `json:"isCode"`
	Content string `json:"content"`
}

func (c *CommandAPIClient) GetCommand(request GetCommandRequest) (*GetCommandInfo, error) {
	url := c.CodeSandboxURL + "/get_command"

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode == 404 {
		return nil, nil // Command not found
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var getCommandResponse GetCommandResponse
	err = json.Unmarshal(respBody, &getCommandResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return getCommandResponse.Command, nil
}

// AutoMemoryRequest represents a request to auto-store memory
type AutoMemoryRequest struct {
	Text string `json:"text"`
}

// AutoMemoryResponse represents a response from auto-store memory
type AutoMemoryResponse struct {
	Error *string `json:"error"`
	Count int     `json:"count"`
}

// AutoStoreMemory sends text to the automemory API to extract and store memories
func (c *CommandAPIClient) AutoStoreMemory(text string) error {
	url := c.CodeSandboxURL + "/automemory"

	request := AutoMemoryRequest{Text: text}
	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(respBody))
	}

	var autoMemoryResponse AutoMemoryResponse
	if err := json.Unmarshal(respBody, &autoMemoryResponse); err != nil {
		return fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if autoMemoryResponse.Error != nil {
		return fmt.Errorf("API error: %s", *autoMemoryResponse.Error)
	}

	return nil
}
