// Package apiclient implements http client for opencode zen
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
)

const apiBase = "https://opencode.ai/zen/v1"
const baseURL = apiBase + "/chat/completions"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type Choices struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type ChatResponse struct {
	ID     string    `json:"id"`
	Choice []Choices `json:"choices"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message, model string, tools []Tool) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
		Tools:    tools,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error serializing request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned error: %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResponse ChatResponse
	err = json.Unmarshal(bodyBytes, &chatResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &chatResponse, nil
}

// Models lists the model ids available on the API.
func (c *Client) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned error: %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	models := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}

	if len(models) == 0 { // fallback: some APIs return a bare array
		var arr []struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(bodyBytes, &arr) == nil {
			for _, m := range arr {
				if m.ID != "" {
					models = append(models, m.ID)
				}
			}
		}
	}

	sort.Strings(models)
	return models, nil
}
