// Package ollama is a minimal client for the Ollama HTTP API.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: http.DefaultClient}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Embed returns one embedding vector per input, in input order.
func (c *Client) Embed(ctx context.Context, model string, inputs []string) ([][]float32, error) {
	req := struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}{model, inputs}
	var resp struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := c.post(ctx, "/api/embed", req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("ollama embed: got %d embeddings for %d inputs", len(resp.Embeddings), len(inputs))
	}
	return resp.Embeddings, nil
}

// ChatJSON runs a deterministic, non-streaming chat completion constrained to
// the JSON schema and returns the raw message content.
func (c *Client) ChatJSON(ctx context.Context, model string, msgs []Message, schema any) ([]byte, error) {
	req := struct {
		Model    string         `json:"model"`
		Messages []Message      `json:"messages"`
		Stream   bool           `json:"stream"`
		Think    bool           `json:"think"`
		Format   any            `json:"format"`
		Options  map[string]any `json:"options"`
	}{model, msgs, false, false, schema, map[string]any{"temperature": 0}}
	var resp struct {
		Message Message `json:"message"`
	}
	if err := c.post(ctx, "/api/chat", req, &resp); err != nil {
		return nil, err
	}
	return []byte(resp.Message.Content), nil
}

func (c *Client) post(ctx context.Context, path string, reqBody, respBody any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("ollama %s: %s: %s", path, resp.Status, strings.TrimSpace(string(msg)))
	}
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return fmt.Errorf("ollama %s: decode response: %w", path, err)
	}
	return nil
}
