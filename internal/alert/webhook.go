package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// WebhookNotifier sends alerts as JSON to a configurable URL.
type WebhookNotifier struct {
	url     string
	headers map[string]string
	client  *http.Client
}

// NewWebhookNotifier creates a webhook notifier.
func NewWebhookNotifier(url string, headers map[string]string) *WebhookNotifier {
	return &WebhookNotifier{
		url:     url,
		headers: headers,
		client:  &http.Client{},
	}
}

func (w *WebhookNotifier) Notify(ctx context.Context, a Alert) error {
	payload := map[string]any{
		"type":     a.Type,
		"severity": a.Severity,
		"message":  a.Message,
		"details":  a.Details,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}
