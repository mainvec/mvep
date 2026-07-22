package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// DefaultEncoder is the default content type for encoding commands
const DefaultEncoder = "application/json"

// ClientConfig holds configuration for the MVP client
type ClientConfig struct {
	// BaseURL is the base URL of the MVP server (e.g., "http://127.0.0.1:8080")
	// For Unix sockets, use "unix:///path/to/socket"
	BaseURL string
	// BasePath is the base URL path for endpoints (e.g., "/api")
	BasePath string
	// Encoder is the content type for encoding commands (default: "application/json")
	Encoder string
	// Timeout is the HTTP client timeout (default: 30 seconds)
	Timeout time.Duration
	// HTTPClient allows using a custom HTTP client (optional)
	// If BaseURL is a Unix socket, this will be ignored
	HTTPClient *http.Client
	// Interceptor is the client interceptor chain for all outgoing requests
	Interceptor mvep.ClientInterceptor
}

// Client represents an MVP client for communicating with an MVP server
type Client struct {
	config      *ClientConfig
	httpClient  *http.Client
	packages    map[string]*PackageClient
	httpBaseURL string // The actual HTTP base URL (for unix sockets this is "http://unixsocket")
	interceptor mvep.ClientInterceptor
}

// PackageClient represents a client for a specific MVP package
type PackageClient struct {
	client  *Client
	pkg     mvep.Package
	handler *mvep.PackageHandler
	encoder string
}

// NewClient creates a new MVP client with the given configuration
func NewClient(config ClientConfig) (*Client, error) {
	// if config == nil {
	// 	return nil, errors.New("config is required")
	// }
	if len(config.BaseURL) == 0 {
		return nil, errors.New("base URL is required")
	}
	if len(config.Encoder) == 0 {
		config.Encoder = DefaultEncoder
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	var httpClient *http.Client
	var httpBaseURL string

	// Check if this is a Unix socket connection
	if strings.HasPrefix(config.BaseURL, "unix://") {
		socketPath := strings.TrimPrefix(config.BaseURL, "unix://")
		tr := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		}
		httpClient = &http.Client{
			Transport: tr,
			Timeout:   config.Timeout,
		}
		httpBaseURL = "http://unixsocket"
	} else {
		httpClient = config.HTTPClient
		if httpClient == nil {
			httpClient = &http.Client{
				Timeout: config.Timeout,
			}
		}
		httpBaseURL = config.BaseURL
	}

	return &Client{
		config:      &config,
		httpClient:  httpClient,
		packages:    make(map[string]*PackageClient),
		httpBaseURL: httpBaseURL,
		interceptor: config.Interceptor,
	}, nil
}

// RegisterPackage registers an MVP package with the client
func (c *Client) RegisterPackage(pkg mvep.Package) (*PackageClient, error) {
	if pkg == nil {
		return nil, errors.New("package is required")
	}

	pkgName := pkg.GetName()
	if _, exists := c.packages[pkgName]; exists {
		return nil, errors.New("package already registered: " + pkgName)
	}

	// Build the package path: basePath + "/" + packageName + "/cmd"
	pkgPath := c.config.BasePath + "/" + pkgName + "/cmd"

	transporter, err := mvep.NewHttpTransporterWithClient(c.httpBaseURL, pkgPath, c.httpClient)
	if err != nil {
		return nil, err
	}

	handler := &mvep.PackageHandler{
		Package:     pkg,
		Transporter: transporter,
	}

	pkgClient := &PackageClient{
		client:  c,
		pkg:     pkg,
		handler: handler,
		encoder: c.config.Encoder,
	}

	c.packages[pkgName] = pkgClient

	return pkgClient, nil
}

// GetPackage returns a registered package client by name
func (c *Client) GetPackage(name string) (*PackageClient, bool) {
	pkg, ok := c.packages[name]
	return pkg, ok
}

// Close closes all idle connections
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// Ping sends a health check request to the server
// It uses the configured BasePath + "/health" endpoint
func (c *Client) Ping(ctx context.Context) (string, error) {
	healthPath := c.config.BasePath + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.httpBaseURL+healthPath, nil)
	if err != nil {
		return "", err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// SendCmd sends a command to the appropriate package based on command type
// The command must be registered with one of the registered packages
func (c *Client) SendCmd(ctx context.Context, cmd any) (any, error) {
	if cmd == nil {
		return nil, errors.New("command is required")
	}

	// Find the package that knows about this command
	for _, pkgClient := range c.packages {
		cmdName := pkgClient.pkg.NameOf(cmd)
		if cmdName != "" {
			return pkgClient.SendCmd(ctx, cmd)
		}
	}

	return nil, errors.New("no registered package found for command")
}

// SendCmd sends a command through this package client.
// Routes through the interceptor chain (auth, logging, etc.) when available.
func (p *PackageClient) SendCmd(ctx context.Context, cmd any) (any, error) {
	result, _, err := p.sendCmdReqInternal(ctx, cmd, nil, p.encoder)
	return result, err
}

// SendCmdWithEncoder sends a command with a specific encoder
func (p *PackageClient) SendCmdWithEncoder(ctx context.Context, cmd any, encoder string) (any, error) {
	return p.handler.SendCmd(ctx, cmd, encoder)
}

// SendRawCmd sends a raw command with bytes directly without encoding/decoding
// This is useful when you want to handle serialization yourself
func (p *PackageClient) SendRawCmd(ctx context.Context, cmdName string, cmdData []byte) ([]byte, error) {
	cmdDataBuf := bytes.NewBuffer(cmdData)
	respData, err := p.handler.Transporter.TransportCmd(ctx, cmdName, p.encoder, io.NopCloser(cmdDataBuf))
	if err != nil {
		return nil, err
	}
	defer respData.Close()
	respBytes, err := io.ReadAll(respData)
	if err != nil {
		return nil, err
	}
	return respBytes, nil
}

// SetEncoder sets the default encoder for this package client
func (p *PackageClient) SetEncoder(encoder string) {
	p.encoder = encoder
}

// GetEncoder returns the current encoder for this package client
func (p *PackageClient) GetEncoder() string {
	return p.encoder
}

// Package returns the underlying MVP package
func (p *PackageClient) Package() mvep.Package {
	return p.pkg
}

// SendCmdReq sends a command with headers and returns the result along with response headers
func (p *PackageClient) SendCmdReq(ctx context.Context, cmd any, headers map[string]string) (any, *mvep.CmdResp, error) {
	return p.sendCmdReqInternal(ctx, cmd, headers, p.encoder)
}

// SendCmdReqWithEncoder sends a command with headers using a specific encoder
func (p *PackageClient) SendCmdReqWithEncoder(ctx context.Context, cmd any, headers map[string]string, encoder string) (any, *mvep.CmdResp, error) {
	return p.sendCmdReqInternal(ctx, cmd, headers, encoder)
}

// sendCmdReqInternal handles the actual sending with optional interceptor
func (p *PackageClient) sendCmdReqInternal(ctx context.Context, cmd any, headers map[string]string, encoder string) (any, *mvep.CmdResp, error) {
	// If no interceptor, call handler directly
	if p.client.interceptor == nil {
		return p.handler.SendCmdReq(ctx, cmd, headers, encoder)
	}

	// Build a CmdReq with the command name and headers for the interceptor
	cmdName := p.pkg.NameOf(cmd)
	if cmdName == "" {
		return nil, nil, errors.New("invalid command")
	}

	req := mvep.NewCmdReq(cmdName, nil)
	if headers != nil {
		for k, v := range headers {
			req.Headers[k] = v
		}
	}

	// Variable to capture the result from the invoker
	var result any

	// Define the invoker that calls the actual handler
	invoker := func(ctx context.Context, req *mvep.CmdReq) (*mvep.CmdResp, error) {
		var resp *mvep.CmdResp
		var err error
		result, resp, err = p.handler.SendCmdReq(ctx, cmd, req.Headers, encoder)
		return resp, err
	}

	// Run through interceptor chain
	resp, err := p.client.interceptor(ctx, req, invoker)
	return result, resp, err
}
