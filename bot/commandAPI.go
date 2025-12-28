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
	APIKey         string
	httpClient     *http.Client
}

func NewCommandAPIClient(codeSandboxURL, apiKey string) *CommandAPIClient {
	return &CommandAPIClient{
		CodeSandboxURL: codeSandboxURL,
		APIKey:         apiKey,
		httpClient:     &http.Client{},
	}
}

// doRequest はAuthorizationヘッダー付きでHTTPリクエストを実行する
func (c *CommandAPIClient) doRequest(method, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	return c.httpClient.Do(req)
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

	response, err := c.doRequest("POST", url, requestBodyJSON)
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

	response, err := c.doRequest("POST", url, requestBodyJSON)
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

	response, err := c.doRequest("POST", url, requestBodyJSON)
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

// SmartRegisterRequest represents a request to smart register a command
type SmartRegisterRequest struct {
	CommandName string `json:"command_name"`
	Description string `json:"description"`
	AuthorID    string `json:"author_id"`
}

// SmartRegisterResponse represents a response from smart register
type SmartRegisterResponse struct {
	Error         *string `json:"error"`
	CommandName   string  `json:"command_name"`
	GeneratedCode string  `json:"generated_code"`
	Usage         string  `json:"usage"`
}

// SmartRegisterCommand generates code from description and registers it
func (c *CommandAPIClient) SmartRegisterCommand(request SmartRegisterRequest) (*SmartRegisterResponse, error) {
	url := c.CodeSandboxURL + "/smart_register"

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := c.doRequest("POST", url, requestBodyJSON)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var smartRegisterResponse SmartRegisterResponse
	if err := json.Unmarshal(respBody, &smartRegisterResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if smartRegisterResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", *smartRegisterResponse.Error)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(respBody))
	}

	return &smartRegisterResponse, nil
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

	response, err := c.doRequest("POST", url, requestBodyJSON)
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

// MentionCommandResponse represents a response from get/set mention command
type MentionCommandResponse struct {
	Error       *string `json:"error"`
	CommandName *string `json:"command_name"`
}

// GetMentionCommand gets the current mention command setting
func (c *CommandAPIClient) GetMentionCommand() (*string, error) {
	url := c.CodeSandboxURL + "/mention_command"

	response, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(respBody))
	}

	var mentionResponse MentionCommandResponse
	if err := json.Unmarshal(respBody, &mentionResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if mentionResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", *mentionResponse.Error)
	}

	return mentionResponse.CommandName, nil
}

// SetMentionCommandRequest represents a request to set mention command
type SetMentionCommandRequest struct {
	CommandName *string `json:"command_name"`
}

// SetMentionCommand sets the mention command
func (c *CommandAPIClient) SetMentionCommand(commandName *string) error {
	url := c.CodeSandboxURL + "/mention_command"

	request := SetMentionCommandRequest{CommandName: commandName}
	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := c.doRequest("POST", url, requestBodyJSON)
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

	var mentionResponse MentionCommandResponse
	if err := json.Unmarshal(respBody, &mentionResponse); err != nil {
		return fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if mentionResponse.Error != nil {
		return fmt.Errorf("API error: %s", *mentionResponse.Error)
	}

	return nil
}

// ==================
// Message API
// ==================

// StoreMessageAPIRequest represents a request to store a Discord message
type StoreMessageAPIRequest struct {
	ID                 string   `json:"id"`
	ChannelID          string   `json:"channel_id"`
	UserID             string   `json:"user_id"`
	Content            string   `json:"content"`
	Timestamp          string   `json:"timestamp"`
	EditedTimestamp    *string  `json:"edited_timestamp,omitempty"`
	ReferenceMessageID *string  `json:"reference_message_id,omitempty"`
	MentionUserIDs     []string `json:"mention_user_ids,omitempty"`
	MentionRoleIDs     []string `json:"mention_role_ids,omitempty"`
	HasAttachments     bool     `json:"has_attachments"`
	Username           string   `json:"username"`
	DisplayName        *string  `json:"display_name,omitempty"`
}

// StoreMessageResponse represents a response from store message
type StoreMessageResponse struct {
	Error      *string `json:"error"`
	Stored     bool    `json:"stored"`
	Vectorized bool    `json:"vectorized"`
}

// StoreMessage sends a message to be stored in the database
func (c *CommandAPIClient) StoreMessage(request StoreMessageAPIRequest) (*StoreMessageResponse, error) {
	url := c.CodeSandboxURL + "/message"

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := c.doRequest("POST", url, requestBodyJSON)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(respBody))
	}

	var storeResponse StoreMessageResponse
	if err := json.Unmarshal(respBody, &storeResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if storeResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", *storeResponse.Error)
	}

	return &storeResponse, nil
}

// UpdateMessageAPIRequest represents a request to update a Discord message
type UpdateMessageAPIRequest struct {
	ID              string `json:"id"`
	Content         string `json:"content"`
	EditedTimestamp string `json:"edited_timestamp"`
}

// UpdateMessage sends a message update to the API
func (c *CommandAPIClient) UpdateMessage(request UpdateMessageAPIRequest) (*StoreMessageResponse, error) {
	url := c.CodeSandboxURL + "/message"

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := c.doRequest("PUT", url, requestBodyJSON)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// 404 is acceptable - message might not have been stored
	if response.StatusCode == 404 {
		return nil, nil
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(respBody))
	}

	var updateResponse StoreMessageResponse
	if err := json.Unmarshal(respBody, &updateResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if updateResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", *updateResponse.Error)
	}

	return &updateResponse, nil
}

// DeleteMessageAPIRequest represents a request to delete a Discord message
type DeleteMessageAPIRequest struct {
	ID string `json:"id"`
}

// DeleteMessageResponse represents a response from delete message
type DeleteMessageResponse struct {
	Error   *string `json:"error"`
	Success bool    `json:"success"`
}

// DeleteMessage sends a message deletion request to the API
func (c *CommandAPIClient) DeleteMessage(request DeleteMessageAPIRequest) (*DeleteMessageResponse, error) {
	url := c.CodeSandboxURL + "/message"

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := c.doRequest("DELETE", url, requestBodyJSON)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// 404 is acceptable - message might not have been stored
	if response.StatusCode == 404 {
		return nil, nil
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(respBody))
	}

	var deleteResponse DeleteMessageResponse
	if err := json.Unmarshal(respBody, &deleteResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if deleteResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", *deleteResponse.Error)
	}

	return &deleteResponse, nil
}

// ==================
// Enhanced mllm API (v2)
// ==================

// EnhancedMllmRequest represents a request to the enhanced mllm endpoint
type EnhancedMllmRequest struct {
	Prompt       string `json:"prompt"`
	ChannelID    string `json:"channel_id"`
	UserID       string `json:"user_id"`
	RecentCount  *int   `json:"recent_count,omitempty"`
	SimilarCount *int   `json:"similar_count,omitempty"`
}

// EnhancedMllmContextInfo represents context information in the response
type EnhancedMllmContextInfo struct {
	RecentCount  int  `json:"recent_count"`
	SimilarCount int  `json:"similar_count"`
	UserFound    bool `json:"user_found"`
}

// EnhancedMllmResponse represents a response from the enhanced mllm endpoint
type EnhancedMllmResponse struct {
	Error   *string                  `json:"error"`
	Output  *string                  `json:"output"`
	Context *EnhancedMllmContextInfo `json:"context"`
}

// EnhancedMllm calls the enhanced mllm endpoint with 3-layer context
func (c *CommandAPIClient) EnhancedMllm(request EnhancedMllmRequest) (*EnhancedMllmResponse, error) {
	url := c.CodeSandboxURL + "/mllm/v2"

	requestBodyJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %w", err)
	}

	response, err := c.doRequest("POST", url, requestBodyJSON)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(respBody))
	}

	var mllmResponse EnhancedMllmResponse
	if err := json.Unmarshal(respBody, &mllmResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if mllmResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", *mllmResponse.Error)
	}

	return &mllmResponse, nil
}
