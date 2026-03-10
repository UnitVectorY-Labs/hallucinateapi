package gemini

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
	"golang.org/x/oauth2/google"
)

// Client communicates with the Vertex AI Gemini API
type Client struct {
	project    string
	location   string
	model      string
	url        string
	apiKey     string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new Gemini client
func NewClient(project, location, model, url, apiKey string, insecure bool, timeout time.Duration) *Client {
	client := &http.Client{Timeout: timeout}
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return &Client{
		project:    project,
		location:   location,
		model:      model,
		url:        url,
		apiKey:     apiKey,
		httpClient: client,
		timeout:    timeout,
	}
}

func buildGenerateContentURL(project, location, model string) string {
	if location == "global" {
		return fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
			project, location, model)
	}

	return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		location, project, location, model)
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

// Generate calls the Gemini API
func (c *Client) Generate(ctx context.Context, systemPrompt, userPrompt string, responseSchema interface{}) (*llm.GenerateResult, error) {
	start := time.Now()

	url := c.url
	if url == "" {
		url = buildGenerateContentURL(c.project, c.location, c.model)
	}

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

	// Determine authorization
	var authToken string
	if c.apiKey != "" {
		authToken = c.apiKey
	} else {
		// Get ADC token
		creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}

		token, err := creds.TokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}
		authToken = token.AccessToken
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+authToken)

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

	return &llm.GenerateResult{
		Content:      content,
		PromptTokens: geminiResp.UsageMetadata.PromptTokenCount,
		OutputTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
		TotalTokens:  geminiResp.UsageMetadata.TotalTokenCount,
		Latency:      latency,
	}, nil
}

// Ensure Client implements the llm.Client interface
var _ llm.Client = (*Client)(nil)
