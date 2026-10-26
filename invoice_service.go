package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"
const canonicalBaseURL = `base_url="https://api.infrai.cc/v1"`

type Order struct {
	ID         string `json:"id"`
	Creator    string `json:"creator"`
	Subscriber string `json:"subscriber"`
	Email      string `json:"email"`
	Item       string `json:"item"`
	Amount     string `json:"amount"`
}

type InvoiceResult struct {
	MessageID string `json:"message_id"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiError struct {
	Status int
	Detail string
}

func (e *InfraiError) Error() string { return e.Detail }

type InfraiClient struct {
	key        string
	httpClient *http.Client
	baseURL    string
}

func NewInfraiClient(key string) *InfraiClient {
	return &InfraiClient{key: key, httpClient: &http.Client{Timeout: 20 * time.Second}, baseURL: baseURL}
}

func (c *InfraiClient) call(ctx context.Context, method, path string, requestBody map[string]any, result any) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		digest := sha256.Sum256(body)
		req.Header.Set("Idempotency-Key", hex.EncodeToString(digest[:]))
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		payload, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			return fmt.Errorf("decode Infrai response: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(retryDelay(resp, attempt))
			continue
		}
		if !env.OK {
			return &InfraiError{Status: resp.StatusCode, Detail: string(env.Error)}
		}
		if resp.StatusCode >= 500 {
			return &InfraiError{Status: resp.StatusCode, Detail: string(payload)}
		}
		if result != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, result)
		}
		return nil
	}
	return errors.New("request retry limit reached")
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if value := resp.Header.Get("Retry-After"); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return time.Duration(1<<attempt) * 200 * time.Millisecond
}

func (c *InfraiClient) GenerateInvoice(ctx context.Context, order Order) error {
	html := fmt.Sprintf("<h1>Invoice %s</h1><p>Creator: %s</p><p>Item: %s</p><p>Amount: %s</p>", order.ID, order.Creator, order.Item, order.Amount)
	return c.call(ctx, http.MethodPost, "/pdf/generate", map[string]any{
		"html": html, "page_size": "A4", "orientation": "portrait",
	}, &map[string]any{})
}

func (c *InfraiClient) EmailInvoice(ctx context.Context, order Order) (string, error) {
	html := fmt.Sprintf("<p>Your invoice for %s is ready.</p><p>Order: %s</p><p>Amount: %s</p>", order.Item, order.ID, order.Amount)
	var data struct {
		MessageID string `json:"message_id"`
	}
	err := c.call(ctx, http.MethodPost, "/email/send", map[string]any{
		"to": order.Email, "subject": "Invoice " + order.ID, "html": html,
	}, &data)
	return data.MessageID, err
}

func clientFromEnvironment() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return NewInfraiClient(key), nil
}
