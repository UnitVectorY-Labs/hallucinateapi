package openaicompat

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/llm"
)

const defaultOpenAIURL = "https://api.openai.com/v1/chat/completions"

// Client communicates with an OpenAI-compatible Chat Completions API
type Client struct {
	model        string
	url          string
	apiKey       string
	strictSchema bool
	httpClient   *http.Client
}

// NewClient creates a new OpenAI-compatible client
func NewClient(model, url, apiKey string, strictSchema, insecure bool, timeout time.Duration) *Client {
	if url == "" {
		url = defaultOpenAIURL
	}
	client := &http.Client{Timeout: timeout}
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return &Client{
		model:        model,
		url:          url,
		apiKey:       apiKey,
		strictSchema: strictSchema,
		httpClient:   client,
	}
}

// request represents the OpenAI Chat Completions API request
type request struct {
	Model          string           `json:"model"`
	Messages       []message        `json:"messages"`
	ResponseFormat *responseFormat   `json:"response_format,omitempty"`
}

// message represents a chat message
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// responseFormat configures the response format
type responseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *jsonSchemaSpec `json:"json_schema,omitempty"`
}

// jsonSchemaSpec configures the JSON schema for structured output
type jsonSchemaSpec struct {
	Name   string      `json:"name"`
	Schema interface{} `json:"schema"`
	Strict bool        `json:"strict,omitempty"`
}

// response represents the OpenAI Chat Completions API response
type response struct {
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

// choice represents a single response choice
type choice struct {
	Message      choiceMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// choiceMessage represents the assistant's message
type choiceMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// usage contains token usage information
type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Generate calls the OpenAI-compatible Chat Completions API
func (c *Client) Generate(ctx context.Context, systemPrompt, userPrompt string, responseSchema interface{}) (*llm.GenerateResult, error) {
	start := time.Now()

	// Build the JSON schema config
	jsonSchemaConfig := &jsonSchemaSpec{
		Name:   "response",
		Schema: responseSchema,
	}
	if c.strictSchema {
		jsonSchemaConfig.Strict = true
	}

	// Build the request body
	req := request{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &responseFormat{
			Type:       "json_schema",
			JSONSchema: jsonSchemaConfig,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Set Authorization header if API key is provided
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var openaiResp response
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	choiceResult := openaiResp.Choices[0]
	if choiceResult.FinishReason != "stop" {
		return nil, fmt.Errorf("unexpected finish reason: %s", choiceResult.FinishReason)
	}

	content := choiceResult.Message.Content
	if content == "" {
		return nil, fmt.Errorf("empty response content from OpenAI")
	}

	latency := time.Since(start)

	return &llm.GenerateResult{
		Content:      content,
		PromptTokens: openaiResp.Usage.PromptTokens,
		OutputTokens: openaiResp.Usage.CompletionTokens,
		TotalTokens:  openaiResp.Usage.TotalTokens,
		Latency:      latency,
	}, nil
}

// Ensure Client implements the llm.Client interface
var _ llm.Client = (*Client)(nil)
