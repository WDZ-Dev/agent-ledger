package mcp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
	"github.com/WDZ-Dev/agent-ledger/internal/provider"
)

const maxRequestBody = 10 << 20 // 10 MB

// HTTPProxy proxies JSON-RPC requests to an upstream MCP server over HTTP,
// intercepting tool calls for metering.
type HTTPProxy struct {
	upstream string
	pricer   *Pricer
	recorder *ledger.Recorder
	logger   *slog.Logger
	client   *http.Client
}

// NewHTTPProxy creates an HTTP proxy for MCP servers.
func NewHTTPProxy(upstream string, pricer *Pricer, recorder *ledger.Recorder, logger *slog.Logger) *HTTPProxy {
	return &HTTPProxy{
		upstream: strings.TrimRight(upstream, "/"),
		pricer:   pricer,
		recorder: recorder,
		logger:   logger,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ServeHTTP handles an incoming MCP JSON-RPC request.
func (p *HTTPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Path sanitization: reject path traversal attempts.
	if strings.Contains(r.URL.Path, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	cleanPath := path.Clean(r.URL.Path)

	// Read request body with size limit.
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	body, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Extract agent context from headers.
	agentID, sessionID, userID, _ := provider.ExtractAgentHeaders(r)
	agentCtx := &AgentContext{
		AgentID:   agentID,
		SessionID: sessionID,
		UserID:    userID,
	}

	// Create a per-request interceptor so concurrent requests don't share
	// inflight state.
	interceptor := NewInterceptor("unknown", p.pricer, p.recorder,
		agentID, sessionID, userID, p.logger)

	// Intercept outbound request.
	interceptor.HandleMessage(body, true, agentCtx)

	// Build upstream URL safely using url.Parse + path.Join.
	upstreamBase, err := url.Parse(p.upstream)
	if err != nil {
		p.logger.Error("failed to parse upstream URL", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	upstreamBase.Path = path.Join(upstreamBase.Path, cleanPath)
	if r.URL.RawQuery != "" {
		upstreamBase.RawQuery = r.URL.RawQuery
	}
	upstreamURL := upstreamBase.String()

	upReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		p.logger.Error("failed to create upstream request", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Copy relevant headers.
	upReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	if accept := r.Header.Get("Accept"); accept != "" {
		upReq.Header.Set("Accept", accept)
	}

	// Forward to upstream.
	resp, err := p.client.Do(upReq) //nolint:gosec // upstream URL is from trusted server config
	if err != nil {
		p.logger.Error("upstream request failed", "error", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	contentType := resp.Header.Get("Content-Type")

	if strings.Contains(contentType, "text/event-stream") {
		p.handleSSE(w, resp, interceptor, agentCtx)
		return
	}

	// Non-streaming: read full response.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Error("failed to read upstream response", "error", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	// Intercept response.
	interceptor.HandleMessage(respBody, false, agentCtx)

	// Write response to client.
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

func (p *HTTPProxy) handleSSE(w http.ResponseWriter, resp *http.Response, interceptor *Interceptor, agentCtx *AgentContext) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Copy headers.
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Use a line scanner to correctly handle SSE data lines that may span
	// TCP read boundaries.
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, maxScanBuffer), maxScanBuffer)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
			if data != "" && data != "[DONE]" {
				interceptor.HandleMessage([]byte(data), false, agentCtx)
			}
		}

		// Write the line through to the client, preserving SSE format.
		_, _ = fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()
	}
}
