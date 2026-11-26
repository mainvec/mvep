// Package mcp exposes any MVEP package (described by *mvep.PackageDesc and
// served by *mvep.PackageHandler) as a Model Context Protocol (MCP) server.
//
// Tool definitions (tools/list) are derived directly from PackageDesc, and
// invocations (tools/call) are dispatched through PackageHandler.ServeCmdReq
// so the full interceptor chain (panic recovery, auth, logging, request IDs)
// applies identically to HTTP and CLI execution.
package mcp

import (
	"strings"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// Tool is an MCP tool definition emitted by tools/list.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolNamer derives the MCP tool name for a CommandDesc. Returned names must
// be non-empty and unique across all commands in a package; collisions are
// rejected by SchemasFromPackage and NewServer.
type ToolNamer func(cmd *mvep.CommandDesc) string

// DefaultToolNamer produces a flat, LLM-friendly snake_case tool name from a
// CommandDesc. The leaf name is cmd.Alias when set, or snake_case(cmd.Name)
// otherwise. When cmd.Group is non-empty, its "/"-separated segments are
// prefixed with underscores (e.g. group "server/tls" + alias "rotate" becomes
// "server_tls_rotate").
func DefaultToolNamer(cmd *mvep.CommandDesc) string {
	leaf := cmd.Alias
	if leaf == "" {
		leaf = toSnake(cmd.Name)
	}
	group := strings.Trim(cmd.Group, "/")
	if group == "" {
		return leaf
	}
	prefix := strings.ReplaceAll(group, "/", "_")
	return prefix + "_" + leaf
}

// DescriptorNameToolNamer uses the descriptor's canonical CommandDesc.Name
// verbatim (e.g. "StartServerCmd"), matching the wire identity used by
// Package.InstanceOf and `mvep exec`/`send`.
func DescriptorNameToolNamer(cmd *mvep.CommandDesc) string {
	return cmd.Name
}

// Option configures SchemasFromPackage and NewServer.
type Option func(*config)

type config struct {
	namer         ToolNamer
	serverName    string
	serverVersion string
}

func defaultConfig(desc *mvep.PackageDesc) config {
	cfg := config{
		namer: DefaultToolNamer,
	}
	if desc != nil {
		cfg.serverName = desc.Name
		cfg.serverVersion = desc.SpecVersion
	}
	if cfg.serverName == "" {
		cfg.serverName = "mvep"
	}
	if cfg.serverVersion == "" {
		cfg.serverVersion = "0.0.0"
	}
	return cfg
}

// WithToolNamer overrides how CommandDesc entries are mapped to MCP tool names.
// Passing nil restores DefaultToolNamer.
func WithToolNamer(fn ToolNamer) Option {
	return func(c *config) {
		if fn == nil {
			c.namer = DefaultToolNamer
			return
		}
		c.namer = fn
	}
}

// WithServerInfo overrides the name and version reported in MCP initialize
// responses (defaults to PackageDesc.Name and PackageDesc.SpecVersion).
func WithServerInfo(name, version string) Option {
	return func(c *config) {
		if name != "" {
			c.serverName = name
		}
		if version != "" {
			c.serverVersion = version
		}
	}
}

// toSnake converts CamelCase to snake_case, matching mvep/cli's convention.
func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
