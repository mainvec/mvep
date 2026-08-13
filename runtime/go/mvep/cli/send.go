package cli

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/ugo/cli"
	oenc "github.com/mainvec/ugo/oencoding"
)

// registerSend adds the `send` verb to the reserved namespace group:
// `svc mvep send`. It reads a stream of CmdReq envelopes (NDJSON or
// concatenated JSON), dispatches each, and emits a CmdResp envelope per record,
// flushing immediately so it works in a live pipeline.
func (a *App) registerSend(ns *cli.Command) {
	var failFast bool
	send := &cli.Command{
		Usage: "send",
		Short: "stream CmdReq envelopes and emit CmdResp envelopes",
		Args: func(cmd *cli.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("mvep send takes no command name; each record names its own command")
			}
			return nil
		},
	}
	send.Flags().BoolVar(&failFast, "fail-fast", false, "stop at the first error")
	send.RunE = func(ctx *cli.Context, args []string) error {
		return a.sendVerb(ctx, failFast)
	}
	ns.AddCommand(send)
}

// sendVerb reads CmdReq records from stdin and writes a CmdResp per record,
// flushing immediately so it works in a live pipeline. It reads line-by-line,
// decoding each line with a fresh decoder, which handles NDJSON (one object per
// line) and concatenated objects on a line with one code path and no format
// sniffing. A malformed line yields a single CmdResp.Error and processing moves
// to the next line — a json.Decoder cannot be reused after a syntax error, so a
// fresh decoder per line avoids the resync trap.
func (a *App) sendVerb(ctx *cli.Context, failFast bool) error {
	br := bufio.NewReader(a.stdin)
	var errored bool

	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			if anyErrored := a.sendLine(ctx, line); anyErrored {
				errored = true
				if failFast {
					break
				}
			}
		}
		if err != nil {
			break // EOF or read error
		}
	}

	if errored {
		return fmt.Errorf("mvep send: one or more records failed")
	}
	return nil
}

// sendLine decodes every CmdReq on one input line and writes its CmdResp for
// each, flushing immediately so a response is readable before the input closes
// (#54). A malformed line emits a single decode_error CmdResp. Returns true if
// any record on the line errored.
func (a *App) sendLine(ctx *cli.Context, line string) bool {
	dec := json.NewDecoder(strings.NewReader(line))
	anyErrored := false
	for {
		var req sendReq
		err := dec.Decode(&req)
		if err == io.EOF {
			return anyErrored
		}
		if err != nil {
			writeResp(ctx, mvep.NewCmdRespError("decode_error", err.Error()))
			return true
		}
		resp := a.sendOne(ctx, &req)
		writeResp(ctx, resp)
		if resp.HasError() {
			anyErrored = true
		}
	}
}

// sendReq is the CLI-local request envelope. Payload is json.RawMessage so a
// raw JSON payload is accepted directly (and base64 is decoded as a fallback),
// matching mvep exec's pleasant raw-JSON path (#62). The shared mvep.CmdReq
// keeps []byte for the transport/server wire format.
type sendReq struct {
	Cmd     string
	Headers map[string]string
	Payload json.RawMessage
}

// payloadBytes resolves the request payload to raw bytes. A JSON object/array
// is used verbatim; a JSON string is treated as base64 (backward compatible
// with the previous wire format).
func (r *sendReq) payloadBytes() ([]byte, error) {
	if len(r.Payload) == 0 {
		return nil, nil
	}
	// If the payload is a JSON string, it is base64-encoded bytes.
	if len(r.Payload) > 0 && r.Payload[0] == '"' {
		var s string
		if err := json.Unmarshal(r.Payload, &s); err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(s)
	}
	return r.Payload, nil
}

// sendOne dispatches a single request and builds its CmdResp. The request is
// placed on the context via ContextWithCmdReq so header-reading interceptors and
// hooks behave identically under the CLI and over HTTP; response headers set via
// SetResponseHeader are read back into the emitted envelope.
func (a *App) sendOne(ctx *cli.Context, req *sendReq) *mvep.CmdResp {
	payload, err := req.payloadBytes()
	if err != nil {
		return mvep.NewCmdRespError("decode_error", err.Error())
	}
	wireReq := mvep.NewCmdReq(req.Cmd, payload)
	wireReq.Headers = req.Headers

	// ContextWithCmdReq returns a plain context.Context; wrap it back into a
	// *cli.Context carrying the invocation's IO writers so dispatch can run the
	// shared execution core with the enriched context.
	base := mvep.ContextWithCmdReq(ctx, wireReq)
	base = mvep.ContextWithCmdResp(base, mvep.NewCmdResp(nil))
	ioctx := cli.ContextWithIO(base, a.stdin, a.stdout, a.stderr)

	result, err := a.dispatch(ioctx, req.Cmd, payload)
	if err != nil {
		return mvep.NewCmdRespError("command_error", err.Error())
	}

	payload, err = encodeResult(result)
	if err != nil {
		return mvep.NewCmdRespError("encode_error", err.Error())
	}

	resp := mvep.NewCmdResp(payload)
	if cr := mvep.CmdRespFromContext(base); cr != nil {
		resp.Headers = cr.Headers
	}
	return resp
}

// encodeResult marshals a command result to JSON for a CmdResp payload, using
// the same encoder registry as the server.
func encodeResult(result any) ([]byte, error) {
	if result == nil {
		return nil, nil
	}
	enc, ok := oenc.LookupEncoding("application/json")
	if !ok {
		return nil, fmt.Errorf("no application/json encoder registered")
	}
	return enc.Encode(result)
}

// writeResp encodes a CmdResp as JSON and writes it, with a newline so the
// stream stays NDJSON-parseable. The payload is emitted as raw JSON (not
// base64) so the response is directly consumable by scripts (#62).
func writeResp(w io.Writer, resp *mvep.CmdResp) {
	out := struct {
		Headers map[string]string `json:"headers,omitempty"`
		Payload json.RawMessage   `json:"payload,omitempty"`
		Error   *mvep.ErrorInfo   `json:"error,omitempty"`
	}{
		Headers: resp.Headers,
		Payload: resp.Payload,
		Error:   resp.Error,
	}
	b, _ := json.Marshal(out)
	w.Write(b)
	w.Write([]byte("\n"))
}
