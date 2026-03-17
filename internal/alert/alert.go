package alert

import "context"

// Alert represents a notification to be sent.
type Alert struct {
	Type     string            // "budget_warning", "budget_exceeded", "loop_detected", "ghost_detected"
	Severity string            // "warning", "critical"
	Message  string            // human-readable message
	Details  map[string]string // additional structured data
}

// Notifier sends alerts to an external destination.
type Notifier interface {
	Notify(ctx context.Context, alert Alert) error
}

// MultiNotifier fans out alerts to multiple notifiers.
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMultiNotifier creates a notifier that sends to all provided notifiers.
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (m *MultiNotifier) Notify(ctx context.Context, alert Alert) error {
	var firstErr error
	for _, n := range m.notifiers {
		if err := n.Notify(ctx, alert); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
