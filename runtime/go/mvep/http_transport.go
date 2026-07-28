package mvep

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	//_ "github.com/mainvec/mvep/runtime/go/mvep/util/protojson"
	_ "github.com/mainvec/ugo/oencoding/json"
)

var _ Transporter = &HttpTransporter{}
var _ EnvelopeTransporter = &HttpTransporter{}

type HttpTransporter struct {
	url    string
	path   string
	client *http.Client
}

func NewHttpTransporter(url string, pkgPath string) (*HttpTransporter, error) {
	return NewHttpTransporterWithClient(url, pkgPath, &http.Client{Timeout: DefaultHTTPClientTimeout})
}

// DefaultHTTPClientTimeout is the timeout applied to transporters built without
// an explicit http.Client, so no code path can hang indefinitely.
const DefaultHTTPClientTimeout = 30 * time.Second

func NewHttpTransporterWithClient(url string, pkgPath string, httpClient *http.Client) (*HttpTransporter, error) {
	if len(url) == 0 {
		return nil, fmt.Errorf("missing url")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("invalid url")
	}
	if len(pkgPath) == 0 {
		return nil, fmt.Errorf("missing package path")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("missing http client")
	}

	return &HttpTransporter{
		url:    url,
		path:   url + pkgPath,
		client: httpClient,
	}, nil
}

func (t *HttpTransporter) TransportCmd(ctx context.Context, cmdName string, contentType string, cmdData io.ReadCloser) (io.ReadCloser, error) {
	if len(cmdName) == 0 {
		return nil, fmt.Errorf("missing command name")
	}
	if len(contentType) == 0 {
		return nil, fmt.Errorf("missing content type")
	}
	if cmdData == nil {
		return nil, fmt.Errorf("missing command data")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.path, cmdData)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-mainvec-cmd", cmdName)
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send http request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		if resp.Header.Get("x-mainvec-error") != "" {
			return nil, fmt.Errorf("error: %s", resp.Header.Get("x-mainvec-error"))
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// TransportCmdReq sends a command request with headers and returns a response with headers
func (t *HttpTransporter) TransportCmdReq(ctx context.Context, cmdReq *CmdReq, contentType string) (*CmdResp, error) {
	if cmdReq == nil {
		return nil, fmt.Errorf("missing command request")
	}
	if len(cmdReq.Cmd) == 0 {
		return nil, fmt.Errorf("missing command name")
	}
	if len(contentType) == 0 {
		return nil, fmt.Errorf("missing content type")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.path, bytes.NewReader(cmdReq.Payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// Set standard headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-mainvec-cmd", cmdReq.Cmd)

	// Set custom headers with prefix
	for k, v := range cmdReq.Headers {
		req.Header.Set(HeaderPrefix+k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send http request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body, bounded.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, DefaultMaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if int64(len(respBody)) > DefaultMaxResponseBytes {
		return nil, fmt.Errorf("response body exceeded limit of %d bytes", DefaultMaxResponseBytes)
	}

	// Build CmdResp
	cmdResp := &CmdResp{
		Headers: make(map[string]string),
		Payload: respBody,
	}

	// Extract response headers with prefix
	for k, v := range resp.Header {
		lowerKey := strings.ToLower(k)
		if strings.HasPrefix(lowerKey, HeaderPrefix) && len(v) > 0 {
			headerKey := strings.TrimPrefix(lowerKey, HeaderPrefix)
			cmdResp.Headers[headerKey] = v[0]
		}
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		errorMsg := resp.Header.Get("x-mainvec-error")
		if errorMsg == "" {
			errorMsg = string(respBody)
		}
		cmdResp.Error = &ErrorInfo{
			Code:    fmt.Sprintf("http_%d", resp.StatusCode),
			Message: errorMsg,
		}
	}

	return cmdResp, nil
}

func (h *PackageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("x-mainvec-error-code", "method_not_allowed")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cmdName := r.Header.Get("x-mainvec-cmd")
	slog.Debug("received cmd", "x-mainvec-cmd", cmdName)
	if cmdName == "" {
		http.Error(w, "missing x-mainvec-cmd header", http.StatusBadRequest)
		return
	}

	rawContentType := r.Header.Get("Content-Type")
	slog.Debug("received cmd", "Content-Type", rawContentType)
	if rawContentType == "" {
		http.Error(w, "missing Content-Type header", http.StatusBadRequest)
		return
	}
	// Parse as a media type so "application/json; charset=utf-8" resolves.
	mediaType, _, err := mime.ParseMediaType(rawContentType)
	if err != nil {
		w.Header().Set("x-mainvec-error-code", "unsupported_media_type")
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	// Extract request headers with prefix
	reqHeaders := make(map[string]string)
	for k, v := range r.Header {
		lowerKey := strings.ToLower(k)
		if strings.HasPrefix(lowerKey, HeaderPrefix) && len(v) > 0 {
			headerKey := strings.TrimPrefix(lowerKey, HeaderPrefix)
			reqHeaders[headerKey] = v[0]
		}
	}

	// Read request body, bounded.
	maxReq := h.MaxRequestBytes
	if maxReq <= 0 {
		maxReq = DefaultMaxRequestBytes
	}
	defer r.Body.Close()
	payload, readErr := io.ReadAll(http.MaxBytesReader(w, r.Body, maxReq))
	if readErr != nil {
		var maxErr *http.MaxBytesError
		if errors.As(readErr, &maxErr) {
			w.Header().Set("x-mainvec-error-code", "payload_too_large")
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		slog.Error("failed to read request body", "error", readErr.Error())
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Build CmdReq and call ServeCmdReq
	cmdReq := &CmdReq{
		Cmd:     cmdName,
		Headers: reqHeaders,
		Payload: payload,
	}

	cmdResp := h.ServeCmdReq(r.Context(), cmdReq, mediaType)

	// Set response headers with prefix
	for k, v := range cmdResp.Headers {
		w.Header().Set(HeaderPrefix+k, v)
	}

	// Handle error response: stable code always, raw detail only when VerboseErrors.
	if cmdResp.HasError() {
		code := cmdResp.Error.Code
		status := HTTPStatusForErrorCode(code)
		slog.Error("command failed", "code", code, "message", cmdResp.Error.Message, "status", status)
		w.Header().Set("x-mainvec-error-code", code)
		w.Header().Set("Content-Type", mediaType)
		msg := "command failed"
		if h.VerboseErrors {
			msg = cmdResp.Error.Message
			w.Header().Set("x-mainvec-error", msg)
		}
		http.Error(w, msg, status)
		return
	}

	// Write success response
	w.Header().Set("Content-Type", mediaType)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(cmdResp.Payload)
	if err != nil {
		slog.Error("failed to write response", "error", err.Error())
	}
}
