// NOMVEP
// Hand-written command implementations for the mvep CLI. This file is NOT
// regenerated (the NOMVEP marker guards it), so it can delegate to the real
// toolkit functions rather than emitting the stub template's
// "command not implemented" bodies.
package mvep

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	toolkit "github.com/mainvec/mvep/toolkit"
	api "github.com/mainvec/mvep/toolkit/mvepapi/api"
)

func runGenerateCmd(ctx context.Context, cmd *api.GenerateCmd) (*api.GenerateCmdResult, error) {
	err := toolkit.ExecuteGenerate(ctx, cmd.In, cmd.Outdir, cmd.Lang, false, cmd.Format)
	if err != nil {
		return nil, err
	}
	return &api.GenerateCmdResult{}, nil
}

func runInitializeCmd(ctx context.Context, cmd *api.InitializeCmd) (*api.InitializeCmdResult, error) {
	err := toolkit.ExeucuteInitializeCmd(ctx, cmd.Name, cmd.Namespace)
	if err != nil {
		return nil, err
	}
	// Scaffold a new spec file in the current directory with a couple of dummy
	// commands, so the user has a valid starting point to edit.
	if err := scaffoldSpec(cmd.Name, cmd.Namespace); err != nil {
		return nil, err
	}
	return &api.InitializeCmdResult{}, nil
}

func runValidateCmd(ctx context.Context, cmd *api.ValidateCmd) (*api.ValidateCmdResult, error) {
	res, err := toolkit.ExecuteValidateCmd(ctx, cmd.In)
	if err != nil {
		return nil, err
	}
	result := &api.ValidateCmdResult{
		Valid: res.Valid(),
	}
	if !res.Valid() {
		for _, e := range res.ValidationErrors() {
			result.Errors = append(result.Errors, e.String())
		}
	}
	return result, nil
}

// scaffoldSpec writes a <name>.jsonc MVEP spec into the current directory with
// a couple of dummy commands, so `mvep init` produces a valid, editable
// starting point. The scaffold is a valid spec (it passes mvep validate).
func scaffoldSpec(name, namespace string) error {
	spec := fmt.Sprintf(`{
	"$id": %q,
	"$schema": "https://spec.mainvec.com/mvepspec/0.2/schema/2026-01-15",
	"name": %q,
	"namespace": %q,
	"title": %q,
	"version": "v0.1",
	"gen_options": {
		"go_package": "github.com/example/%s/mvepapi;%s",
		"go_api_package": "github.com/example/%s/mvepapi/api;api",
		"format": "plain",
		"cli": "runtime"
	},
	"commands": {
		"PingCmd": {
			"title": "Ping the service",
			"alias": "ping",
			"desc": "Returns a pong to confirm the service is reachable",
			"fields": {
				"message": { "fnum": 1, "type": "string", "title": "Message to echo" }
			},
			"resultFields": {
				"pong": { "fnum": 1, "type": "string", "title": "Echoed message" }
			}
		},
		"StatusCmd": {
			"title": "Get service status",
			"alias": "status",
			"desc": "Reports the current service status",
			"resultFields": {
				"ok": { "fnum": 1, "type": "boolean", "title": "Whether the service is healthy" }
			}
		}
	}
}
`, name, name, namespace, name, name, name, name)

	path := filepath.Join(".", name+".jsonc")
	return os.WriteFile(path, []byte(spec), 0o644)
}
