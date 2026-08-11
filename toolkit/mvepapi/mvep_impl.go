// NOMVEP
// Hand-written command implementations for the mvep CLI. This file is NOT
// regenerated (the NOMVEP marker guards it), so it can delegate to the real
// toolkit functions rather than emitting the stub template's
// "command not implemented" bodies.
package mvep

import (
	"context"

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
