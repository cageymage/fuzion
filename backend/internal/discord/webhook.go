package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Message struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

type Embed struct {
	Title       string       `json:"title,omitempty"`
	URL         string       `json:"url,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Image       *EmbedImage  `json:"image,omitempty"`
}

type EmbedImage struct {
	URL string `json:"url"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type Webhook struct {
	url    string
	client *http.Client
}

func NewWebhook(webhookURL string, client *http.Client) *Webhook {
	if client == nil {
		client = http.DefaultClient
	}
	return &Webhook{url: webhookURL, client: client}
}

func (w *Webhook) Send(ctx context.Context, msg Message) error {
	resp, err := w.do(ctx, http.MethodPost, w.url, msg)
	if err != nil {
		return fmt.Errorf("post webhook: %w", err)
	}
	return resp.Body.Close()
}

// Post sends the message and returns its id so it can be edited later.
func (w *Webhook) Post(ctx context.Context, msg Message) (string, error) {
	resp, err := w.do(ctx, http.MethodPost, w.url+"?wait=true", msg)
	if err != nil {
		return "", fmt.Errorf("post webhook: %w", err)
	}
	defer resp.Body.Close()

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decode webhook response: %w", err)
	}
	return created.ID, nil
}

// Edit replaces the message's content and embeds. Discord only lets a webhook edit its own messages.
func (w *Webhook) Edit(ctx context.Context, messageID string, msg Message) error {
	resp, err := w.do(ctx, http.MethodPatch, w.url+"/messages/"+url.PathEscape(messageID), msg)
	if err != nil {
		return fmt.Errorf("edit webhook message: %w", err)
	}
	return resp.Body.Close()
}

func (w *Webhook) do(ctx context.Context, method, target string, msg Message) (*http.Response, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("encode webhook message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, redactURL(err, w.url)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return resp, nil
}

// The webhook URL embeds its token, and net/http errors quote the full URL.
func redactURL(err error, webhookURL string) error {
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), webhookURL, "<webhook url>"))
}
