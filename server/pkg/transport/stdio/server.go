package stdio

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

// Server bridges stdio framing to the MCP protocol handler.
type Server struct {
	logger   *slog.Logger
	protocol *protocol.Server
	reader   *bufio.Reader
	writer   *bufio.Writer
	mode     FrameMode
}

// NewServer constructs a stdio MCP server.
func NewServer(
	handler *protocol.Server,
	input io.Reader,
	output io.Writer,
	logger *slog.Logger,
) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		logger:   logger,
		protocol: handler,
		reader:   bufio.NewReader(input),
		writer:   bufio.NewWriter(output),
		mode:     "",
	}
}

// Serve processes framed JSON-RPC messages until EOF or context cancellation.
func (s *Server) Serve(ctx context.Context) error {
	resultCh := make(chan readResult, 1)
	go func() {
		for {
			payload, mode, err := ReadFrameWithMode(s.reader)
			resultCh <- readResult{payload: payload, err: err, mode: mode}
			if err != nil {
				close(resultCh)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case result, ok := <-resultCh:
			if !ok {
				return nil
			}
			if s.mode == "" && result.mode != "" {
				s.mode = result.mode
				s.logger.Debug("stdio framing mode selected", "mode", s.mode)
			}
			if result.err != nil {
				if result.err == io.EOF {
					return nil
				}
				return fmt.Errorf("stdio read: %w", result.err)
			}

			responses, err := s.protocol.Handle(ctx, result.payload)
			if err != nil {
				s.logger.Error("protocol handler error", "error", err)
				continue
			}

			for _, response := range responses {
				if s.mode == ModeJSON {
					if err := WriteJSONFrame(s.writer, response); err != nil {
						return fmt.Errorf("stdio write: %w", err)
					}
					continue
				}
				if err := WriteFrame(s.writer, response); err != nil {
					return fmt.Errorf("stdio write: %w", err)
				}
			}
		}
	}
}

type readResult struct {
	payload []byte
	err     error
	mode    FrameMode
}
