package mcp

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
)

func TestStdioWrapper_EchoCommand(t *testing.T) {
	store := &recordingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pricer := NewPricer(nil)
	rec := ledger.NewRecorder(store, 100, 1, logger)
	defer rec.Close()

	interceptor := NewInterceptor("test", pricer, rec, "", "", "", logger)
	wrapper := NewStdioWrapper(interceptor, logger)

	// Use "echo" as a simple child process that exits immediately.
	code, err := wrapper.Run(context.Background(), "echo", []string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestStdioWrapper_FailingCommand(t *testing.T) {
	store := &recordingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pricer := NewPricer(nil)
	rec := ledger.NewRecorder(store, 100, 1, logger)
	defer rec.Close()

	interceptor := NewInterceptor("test", pricer, rec, "", "", "", logger)
	wrapper := NewStdioWrapper(interceptor, logger)

	code, err := wrapper.Run(context.Background(), "false", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestStdioWrapper_NonexistentCommand(t *testing.T) {
	store := &recordingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pricer := NewPricer(nil)
	rec := ledger.NewRecorder(store, 100, 1, logger)
	defer rec.Close()

	interceptor := NewInterceptor("test", pricer, rec, "", "", "", logger)
	wrapper := NewStdioWrapper(interceptor, logger)

	_, err := wrapper.Run(context.Background(), "nonexistent-command-xyz", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
}
