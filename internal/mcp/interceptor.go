package mcp

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

const methodToolsCall = "tools/call"

// AgentContext carries per-request agent identification (used in HTTP mode).
type AgentContext struct {
	AgentID   string
	SessionID string
	UserID    string
}

type pendingCall struct {
	toolName  string
	startTime time.Time
}

// Interceptor observes JSON-RPC messages between an MCP client and server,
// recording tool call usage into the ledger.
type Interceptor struct {
	serverName string
	pricer     *Pricer
	recorder   *ledger.Recorder
	logger     *slog.Logger

	// Default agent context (from env vars in stdio mode).
	agentID   string
	sessionID string
	userID    string

	mu       sync.Mutex
	inflight map[string]*pendingCall // JSON-RPC ID string → pending call
}

// NewInterceptor creates an Interceptor that records MCP tool call usage.
func NewInterceptor(
	serverName string,
	pricer *Pricer,
	recorder *ledger.Recorder,
	agentID, sessionID, userID string,
	logger *slog.Logger,
) *Interceptor {
	return &Interceptor{
		serverName: serverName,
		pricer:     pricer,
		recorder:   recorder,
		agentID:    agentID,
		sessionID:  sessionID,
		userID:     userID,
		logger:     logger,
		inflight:   make(map[string]*pendingCall),
	}
}

// HandleMessage processes a single JSON-RPC message. fromClient indicates
// direction (true = client→server, false = server→client). agentCtx overrides
// default agent context when non-nil (HTTP mode).
func (i *Interceptor) HandleMessage(data []byte, fromClient bool, agentCtx *AgentContext) {
	isReq, req, resp, err := ParseMessage(data)
	if err != nil {
		// Not all lines are JSON-RPC (e.g. debug output). Silently ignore.
		return
	}

	if fromClient && isReq {
		i.handleClientRequest(req)
		return
	}

	if !fromClient && !isReq && resp != nil {
		i.handleServerResponse(resp, agentCtx)
		return
	}

	// Server-side notifications or other messages: check for initialize result.
	if !fromClient && isReq {
		// Server sending a request/notification to client — nothing to meter.
		return
	}
}

func (i *Interceptor) handleClientRequest(req *Request) {
	if req.Method != methodToolsCall {
		return
	}
	if req.ID == nil {
		return
	}

	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		i.logger.Debug("failed to parse tools/call params", "error", err)
		return
	}

	idKey := string(req.ID)
	i.mu.Lock()
	i.inflight[idKey] = &pendingCall{
		toolName:  params.Name,
		startTime: time.Now(),
	}
	i.mu.Unlock()

	i.logger.Debug("mcp tool call started", "tool", params.Name, "id", idKey)
}

func (i *Interceptor) handleServerResponse(resp *Response, agentCtx *AgentContext) {
	if resp.ID == nil {
		// Check if this is an initialize result embedded differently.
		return
	}

	idKey := string(resp.ID)

	i.mu.Lock()
	pending, ok := i.inflight[idKey]
	if ok {
		delete(i.inflight, idKey)
	}
	i.mu.Unlock()

	if !ok {
		// Response to something other than tools/call, or an unknown ID.
		// Check for initialize result.
		i.tryExtractServerName(resp)
		return
	}

	duration := time.Since(pending.startTime)
	cost := i.pricer.CostForCall(i.serverName, pending.toolName)

	agentID, sessionID, userID := i.agentID, i.sessionID, i.userID
	if agentCtx != nil {
		agentID = agentCtx.AgentID
		sessionID = agentCtx.SessionID
		userID = agentCtx.UserID
	}

	statusCode := 200
	if resp.Error != nil {
		statusCode = 500
	}

	record := &ledger.UsageRecord{
		ID:         ulid.Make().String(),
		Timestamp:  time.Now(),
		Provider:   "mcp",
		Model:      i.serverName + ":" + pending.toolName,
		CostUSD:    cost,
		DurationMS: duration.Milliseconds(),
		StatusCode: statusCode,
		Path:       methodToolsCall,
		AgentID:    agentID,
		SessionID:  sessionID,
		UserID:     userID,
	}

	i.recorder.Record(record)
	i.logger.Debug("mcp tool call recorded",
		"tool", pending.toolName,
		"server", i.serverName,
		"cost", cost,
		"duration_ms", duration.Milliseconds(),
	)
}

func (i *Interceptor) tryExtractServerName(resp *Response) {
	if resp.Result == nil {
		return
	}
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return
	}
	if result.ServerInfo.Name != "" {
		i.mu.Lock()
		i.serverName = result.ServerInfo.Name
		i.mu.Unlock()
		i.logger.Info("mcp server identified", "name", result.ServerInfo.Name)
	}
}

// ServerName returns the current server name.
func (i *Interceptor) ServerName() string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.serverName
}
