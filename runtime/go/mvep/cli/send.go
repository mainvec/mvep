package cli

import (
	"bufio"
	"bytes"
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
	var respOut bytes.Buffer

	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			if anyErrored := a.sendLine(ctx, &respOut, line); anyErrored {
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

	ctx.Write(respOut.Bytes())
	if errored {
		return fmt.Errorf("mvep send: one or more records failed")
	}
	return nil
}

// sendLine decodes every CmdReq on one input line and writes its CmdResp for
// each. A malformed line emits a single decode_error CmdResp. Returns true if
// any record on the line errored.
func (a *App) sendLine(ctx *cli.Context, out *bytes.Buffer, line string) bool {
	dec := json.NewDecoder(strings.NewReader(line))
	anyErrored := false
	for {
		var req mvep.CmdReq
		err := dec.Decode(&req)
		if err == io.EOF {
			return anyErrored
		}
		if err != nil {
			writeResp(out, mvep.NewCmdRespError("decode_error", err.Error()))
			return true
		}
		resp := a.sendOne(ctx, &req)
		writeResp(out, resp)
		if resp.HasError() {
			anyErrored = true
		}
	}
}

// sendOne dispatches a single CmdReq and builds its CmdResp. The request is
// placed on the context via ContextWithCmdReq so header-reading interceptors and
// hooks behave identically under the CLI and over HTTP; response headers set via
// SetResponseHeader are read back into the emitted envelope.
func (a *App) sendOne(ctx *cli.Context, req *mvep.CmdReq) *mvep.CmdResp {
	// ContextWithCmdReq returns a plain context.Context; wrap it back into a
	// *cli.Context carrying the invocation's IO writers so dispatch can run the
	// shared execution core with the enriched context.
	base := mvep.ContextWithCmdReq(ctx, req)
	base = mvep.ContextWithCmdResp(base, mvep.NewCmdResp(nil))
	ioctx := cli.ContextWithIO(base, a.stdin, a.stdout, a.stderr)

	result, err := a.dispatch(ioctx, req.Cmd, req.Payload)
	if err != nil {
		return mvep.NewCmdRespError("command_error", err.Error())
	}

	payload, err := encodeResult(result)
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
// stream stays NDJSON-parseable.
func writeResp(w io.Writer, resp *mvep.CmdResp) {
	b, _ := json.Marshal(resp)
	w.Write(b)
	w.Write([]byte("\n"))
}
