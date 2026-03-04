package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2/google"
)

// Client communicates with the Vertex AI Gemini API
type Client struct {
	project    string
	location   string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new Gemini client
func NewClient(project, location, model string, timeout time.Duration) *Client {
	return &Client{
		project:    project,
		location:   location,
		model:      model,
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
	}
}

// Request represents the Gemini API request
type Request struct {
	SystemInstruction *Content         `json:"systemInstruction,omitempty"`
	Contents          []ContentItem    `json:"contents"`
	GenerationConfig  GenerationConfig `json:"generationConfig"`
}

// Content holds text parts
type Content struct {
	Parts []Part `json:"parts"`
}

// ContentItem holds a role and parts
type ContentItem struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// Part is a text part
type Part struct {
	Text string `json:"text"`
}

// GenerationConfig configures the generation
type GenerationConfig struct {
	ResponseMIMEType   string      `json:"responseMimeType"`
	ResponseJSONSchema interface{} `json:"responseJsonSchema,omitempty"`
}

// Response represents the Gemini API response
type Response struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}

// Candidate represents a single response candidate
type Candidate struct {
	Content       CandidateContent `json:"content"`
	FinishReason  string           `json:"finishReason"`
	FinishMessage string           `json:"finishMessage"`
}

// CandidateContent is the content of a candidate
type CandidateContent struct {
	Parts []Part `json:"parts"`
}

// UsageMetadata contains token usage information
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GenerateResult holds the result of a generation call
type GenerateResult struct {
	Content       string
	UsageMetadata UsageMetadata
	Latency       time.Duration
}

// Generate calls the Gemini API
func (c *Client) Generate(ctx context.Context, systemPrompt, userPrompt string, responseSchema interface{}) (*GenerateResult, error) {
	start := time.Now()

	// Build the URL
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		c.location, c.project, c.location, c.model)

	// Build the request body
	req := Request{
		Contents: []ContentItem{
			{
				Role:  "user",
				Parts: []Part{{Text: userPrompt}},
			},
		},
		GenerationConfig: GenerationConfig{
			ResponseMIMEType:   "application/json",
			ResponseJSONSchema: responseSchema,
		},
	}

	if systemPrompt != "" {
		req.SystemInstruction = &Content{
			Parts: []Part{{Text: systemPrompt}},
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Get ADC token
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %w", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp Response
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in Gemini response")
	}

	// Concatenate all text parts
	var content string
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		content += part.Text
	}

	latency := time.Since(start)

	return &GenerateResult{
		Content:       content,
		UsageMetadata: geminiResp.UsageMetadata,
		Latency:       latency,
	}, nil
}

// GeminiClientInterface allows for mocking in tests
type GeminiClientInterface interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string, responseSchema interface{}) (*GenerateResult, error)
}

// Ensure Client implements the interface
var _ GeminiClientInterface = (*Client)(nil)
