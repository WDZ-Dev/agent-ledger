package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
	"github.com/WDZ-Dev/agent-ledger/internal/meter"
	"github.com/WDZ-Dev/agent-ledger/internal/provider"
)

// streamInterceptor wraps a streaming response body. It passes bytes through
// to the caller unmodified while parsing SSE events to extract token usage.
type streamInterceptor struct {
	src      io.ReadCloser
	prov     provider.Provider
	reqMeta  *provider.RequestMeta
	meter    *meter.Meter
	recorder *ledger.Recorder
	logger   *slog.Logger

	parseBuf bytes.Buffer // accumulates raw bytes for SSE parsing

	model        string
	inputTokens  int
	outputTokens int

	start      time.Time
	apiKeyHash string
	path       string
	agentID    string
	sessionID  string
	userID     string

	once sync.Once // ensures finalize runs exactly once
}

func newStreamInterceptor(
	src io.ReadCloser,
	prov provider.Provider,
	reqMeta *provider.RequestMeta,
	m *meter.Meter,
	recorder *ledger.Recorder,
	logger *slog.Logger,
	start time.Time,
	apiKeyHash, path string,
	agentID, sessionID, userID string,
) *streamInterceptor {
	return &streamInterceptor{
		src:        src,
		prov:       prov,
		reqMeta:    reqMeta,
		meter:      m,
		recorder:   recorder,
		logger:     logger,
		start:      start,
		apiKeyHash: apiKeyHash,
		path:       path,
		agentID:    agentID,
		sessionID:  sessionID,
		userID:     userID,
	}
}

// Read passes bytes through to the proxy (and thus the client) while also
// feeding them into the SSE parser.
func (s *streamInterceptor) Read(p []byte) (int, error) {
	n, err := s.src.Read(p)
	if n > 0 {
		s.parseBuf.Write(p[:n])
		s.processEvents()
	}
	if err == io.EOF {
		s.finalize()
	}
	return n, err
}

func (s *streamInterceptor) Close() error {
	s.finalize()
	return s.src.Close()
}

// processEvents scans the buffer for complete SSE events (terminated by
// double newline) and parses each one.
func (s *streamInterceptor) processEvents() {
	for {
		data := s.parseBuf.Bytes()
		idx := bytes.Index(data, []byte("\n\n"))
		if idx == -1 {
			// Also check for \r\n\r\n
			idx = bytes.Index(data, []byte("\r\n\r\n"))
			if idx == -1 {
				break
			}
			event := make([]byte, idx)
			copy(event, data[:idx])
			s.parseBuf.Next(idx + 4)
			s.parseSSEEvent(event)
			continue
		}
		event := make([]byte, idx)
		copy(event, data[:idx])
		s.parseBuf.Next(idx + 2)
		s.parseSSEEvent(event)
	}
}

// parseSSEEvent extracts the event type and data payload from a raw SSE
// event block and delegates to the provider's stream parser.
func (s *streamInterceptor) parseSSEEvent(raw []byte) {
	var eventType string
	var dataPayload []byte

	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		switch {
		case bytes.HasPrefix(line, []byte("event:")):
			eventType = string(bytes.TrimSpace(bytes.TrimPrefix(line, []byte("event:"))))
		case bytes.HasPrefix(line, []byte("data:")):
			d := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
			if len(dataPayload) > 0 {
				dataPayload = append(dataPayload, '\n')
			}
			dataPayload = append(dataPayload, d...)
		}
	}

	if len(dataPayload) == 0 || bytes.Equal(dataPayload, []byte("[DONE]")) {
		return
	}

	chunk, err := s.prov.ParseStreamChunk(eventType, dataPayload)
	if err != nil || chunk == nil {
		return
	}

	if chunk.Model != "" {
		s.model = chunk.Model
	}
	if chunk.InputTokens > 0 {
		s.inputTokens = chunk.InputTokens
	}
	if chunk.OutputTokens > 0 {
		s.outputTokens = chunk.OutputTokens
	}
}

// finalize records the accumulated token usage after the stream ends.
func (s *streamInterceptor) finalize() {
	s.once.Do(func() {
		model := s.model
		if model == "" && s.reqMeta != nil {
			model = s.reqMeta.Model
		}

		cost := s.meter.Calculate(model, s.inputTokens, s.outputTokens)
		estimated := s.inputTokens == 0 && s.outputTokens == 0

		record := &ledger.UsageRecord{
			ID:           ulid.Make().String(),
			Timestamp:    s.start,
			Provider:     s.prov.Name(),
			Model:        model,
			APIKeyHash:   s.apiKeyHash,
			InputTokens:  s.inputTokens,
			OutputTokens: s.outputTokens,
			TotalTokens:  s.inputTokens + s.outputTokens,
			CostUSD:      cost,
			Estimated:    estimated,
			DurationMS:   time.Since(s.start).Milliseconds(),
			StatusCode:   200,
			Path:         s.path,
			AgentID:      s.agentID,
			SessionID:    s.sessionID,
			UserID:       s.userID,
		}
		s.recorder.Record(record)

		s.logger.Info("stream",
			"provider", s.prov.Name(),
			"model", model,
			"input_tokens", s.inputTokens,
			"output_tokens", s.outputTokens,
			"cost_usd", fmt.Sprintf("%.6f", cost),
			"duration_ms", record.DurationMS,
			"estimated", estimated,
		)
	})
}
