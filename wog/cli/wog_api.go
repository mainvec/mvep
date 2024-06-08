package wogcli

import "context"

type InitializeiCmdHandler func(context.Context, *InitializeCmd) (*InitializeCmdResult, error)
type GenerateCmdHandler func(context.Context, *GenerateCmd) (*GenerateCmdResult, error)
