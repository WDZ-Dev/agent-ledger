package mcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

const maxScanBuffer = 1 << 20 // 1 MB — MCP messages can be large

// StdioWrapper wraps an MCP server process, intercepting stdin/stdout
// JSON-RPC messages for metering while passing data through transparently.
type StdioWrapper struct {
	interceptor *Interceptor
	logger      *slog.Logger
}

// NewStdioWrapper creates a StdioWrapper.
func NewStdioWrapper(interceptor *Interceptor, logger *slog.Logger) *StdioWrapper {
	return &StdioWrapper{
		interceptor: interceptor,
		logger:      logger,
	}
}

// Run starts the child MCP server process and pipes stdin/stdout through the
// interceptor. Stderr is passed through directly. Returns the child exit code.
func (w *StdioWrapper) Run(ctx context.Context, command string, args []string) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...) //nolint:gosec // command is from CLI args, not untrusted input
	cmd.Stderr = os.Stderr

	childIn, err := cmd.StdinPipe()
	if err != nil {
		return 1, fmt.Errorf("stdin pipe: %w", err)
	}
	childOut, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("starting command: %w", err)
	}

	// Forward signals to child process. Closing sigCh after signal.Stop
	// ensures the goroutine exits (range terminates on close).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()
	defer func() {
		signal.Stop(sigCh)
		close(sigCh)
	}()

	outDone := make(chan struct{})

	// stdin → child: read from parent stdin, intercept, write to child stdin.
	// This goroutine may outlive Run() if stdin remains open after the child
	// exits — that is expected for a CLI tool and cleaned up on process exit.
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 0, maxScanBuffer), maxScanBuffer)
		for scanner.Scan() {
			line := scanner.Bytes()
			w.interceptor.HandleMessage(line, true, nil)
			if _, err := childIn.Write(line); err != nil {
				break
			}
			if _, err := childIn.Write([]byte("\n")); err != nil {
				break
			}
		}
		_ = childIn.Close()
	}()

	// child stdout → parent: read from child stdout, intercept, write to
	// parent stdout. Signals outDone when the child closes stdout.
	go func() {
		defer close(outDone)
		scanner := bufio.NewScanner(childOut)
		scanner.Buffer(make([]byte, 0, maxScanBuffer), maxScanBuffer)
		for scanner.Scan() {
			line := scanner.Bytes()
			w.interceptor.HandleMessage(line, false, nil)
			if _, err := io.WriteString(os.Stdout, scanner.Text()+"\n"); err != nil {
				break
			}
		}
		if err := scanner.Err(); err != nil {
			w.logger.Debug("stdout pipe error", "error", err)
		}
	}()

	// Wait for child stdout to drain fully before calling cmd.Wait
	// (required by exec.Cmd.StdoutPipe contract).
	<-outDone

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}

	return 0, nil
}
