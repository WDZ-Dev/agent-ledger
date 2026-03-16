package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SlackNotifier sends alerts to a Slack webhook URL.
type SlackNotifier struct {
	webhookURL string
	client     *http.Client
}

// NewSlackNotifier creates a Slack notifier.
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

type slackMessage struct {
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type string     `json:"type"`
	Text *slackText `json:"text,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *SlackNotifier) Notify(ctx context.Context, a Alert) error {
	icon := ":warning:"
	if a.Severity == "critical" {
		icon = ":rotating_light:"
	}

	header := fmt.Sprintf("%s *AgentLedger Alert: %s*", icon, a.Type)

	var detailLines string
	for k, v := range a.Details {
		detailLines += fmt.Sprintf("• *%s:* %s\n", k, v)
	}

	msg := slackMessage{
		Blocks: []slackBlock{
			{Type: "header", Text: &slackText{Type: "plain_text", Text: header}},
			{Type: "section", Text: &slackText{Type: "mrkdwn", Text: a.Message}},
		},
	}
	if detailLines != "" {
		msg.Blocks = append(msg.Blocks, slackBlock{
			Type: "section",
			Text: &slackText{Type: "mrkdwn", Text: detailLines},
		})
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshalling slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending slack notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
