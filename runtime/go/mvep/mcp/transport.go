package mcp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// DefaultMaxMessageBytes bounds a single inbound JSON-RPC message read by the
// stream transport (4 MiB, matching mvep.DefaultMaxRequestBytes).
const DefaultMaxMessageBytes = 4 << 20

// Transport carries framed JSON-RPC 2.0 messages between an MCP client and a
// Server. Implementations must return io.EOF from Read when the client stream
// closes cleanly.
type Transport interface {
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, msg []byte) error
	Close() error
}

// Stdio returns a newline-delimited JSON-RPC 2.0 transport over os.Stdin and
// os.Stdout. Callers using Stdio must ensure nothing else writes to os.Stdout
// (direct application logs to os.Stderr).
func Stdio() Transport {
	return NewStreamTransport(os.Stdin, os.Stdout)
}

// StreamTransport frames one JSON-RPC message per newline over an io.Reader
// and io.Writer. It is safe for concurrent Write calls.
type StreamTransport struct {
	reader          *bufio.Reader
	writer          io.Writer
	mu              sync.Mutex
	maxMessageBytes int
}

// NewStreamTransport creates a newline-delimited JSON-RPC transport backed by
// r and w. Useful for in-process testing over io.Pipe or buffers.
func NewStreamTransport(r io.Reader, w io.Writer) *StreamTransport {
	return &StreamTransport{
		reader:          bufio.NewReaderSize(r, 64*1024),
		writer:          w,
		maxMessageBytes: DefaultMaxMessageBytes,
	}
}

// Read returns the next non-empty JSON-RPC message line from the underlying
// reader, or io.EOF when the stream ends.
func (t *StreamTransport) Read(ctx context.Context) ([]byte, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, err := t.readLine()
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				trimmed := bytes.TrimSpace(line)
				if len(trimmed) > 0 {
					return trimmed, nil
				}
			}
			return nil, err
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		return trimmed, nil
	}
}

func (t *StreamTransport) readLine() ([]byte, error) {
	limit := t.maxMessageBytes
	if limit <= 0 {
		limit = DefaultMaxMessageBytes
	}

	var buf []byte
	for {
		chunk, err := t.reader.ReadSlice('\n')
		if len(buf)+len(chunk) > limit {
			// Drain the rest of the oversized line so subsequent reads align.
			for err == bufio.ErrBufferFull {
				_, err = t.reader.ReadSlice('\n')
			}
			return nil, fmt.Errorf("mcp: message exceeds limit of %d bytes", limit)
		}
		buf = append(buf, chunk...)
		if err == bufio.ErrBufferFull {
			continue
		}
		return buf, err
	}
}

// Write serializes msg followed by a single newline and flushes if the writer
// supports flushing. Embedded newlines in msg are rejected to preserve stdio
// framing.
func (t *StreamTransport) Write(_ context.Context, msg []byte) error {
	if bytes.ContainsAny(msg, "\r\n") {
		return fmt.Errorf("mcp: outbound JSON-RPC message must not contain newlines")
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]byte, len(msg)+1)
	copy(out, msg)
	out[len(msg)] = '\n'
	if _, err := t.writer.Write(out); err != nil {
		return err
	}
	if f, ok := t.writer.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// Close is a no-op for StreamTransport; callers own closing the underlying
// streams when appropriate.
func (t *StreamTransport) Close() error {
	return nil
}
