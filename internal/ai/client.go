package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"a1s/internal/config"
)

type Client struct {
	BaseURL, APIKey, Model string
	HTTP                   *http.Client
}

func New(cfg config.Config) *Client {
	base := strings.TrimRight(cfg.AIBaseURL, "/")
	return &Client{BaseURL: base, APIKey: cfg.AIAPIKey, Model: cfg.AIModel, HTTP: &http.Client{Timeout: 45 * time.Second}}
}
func (c *Client) Enabled() bool { return c.BaseURL != "" && c.Model != "" }

func (c *Client) Ask(ctx context.Context, system, prompt string) (string, error) {
	if !c.Enabled() {
		return "", errors.New("AI is not configured; set A1S_AI_BASE_URL and A1S_AI_MODEL (and A1S_AI_API_KEY when required)")
	}
	body := map[string]any{"model": c.Model, "messages": []map[string]string{{"role": "system", "content": system}, {"role": "user", "content": prompt}}, "temperature": 0.2}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("AI endpoint returned %s", resp.Status)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("AI response had no choices")
	}
	return out.Choices[0].Message.Content, nil
}
