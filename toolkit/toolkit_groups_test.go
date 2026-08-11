package toolkit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mainvec/mvep/toolkit"
)

// getTestFilePath resolves a testdata fixture path (mirrors toolkit_test.go).
func groupsTestFilePath(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, "testdata", name), nil
}

// TestCommandGroupsRoundTrip is the T2 guard against the "tags" class of
// defect (#23): a spec property validated by the JSON schema but silently
// dropped on unmarshal. The fixture declares groups at depth 1 and 2, an
// alias, and a hidden group; every one must survive BuildSrvDefFromJSON.
func TestCommandGroupsRoundTrip(t *testing.T) {
	path, err := groupsTestFilePath("13_command_groups.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	srvDef, err := toolkit.BuildSrvDefFromJSON(f)
	if err != nil {
		t.Fatalf("BuildSrvDefFromJSON: %v", err)
	}

	// Top-level commandGroups metadata survives.
	g, ok := srvDef.CommandGroups.Get("server")
	if !ok {
		t.Fatalf("commandGroups missing group %q", "server")
	}
	if g.Title != "LLM Servers" {
		t.Errorf("server.Title = %q, want %q", g.Title, "LLM Servers")
	}
	if g.Desc != "Start, stop and inspect servers" {
		t.Errorf("server.Desc = %q, want %q", g.Desc, "Start, stop and inspect servers")
	}

	keys, ok := srvDef.CommandGroups.Get("server/keys")
	if !ok {
		t.Fatalf("commandGroups missing group %q", "server/keys")
	}
	if keys.Title != "API Keys" {
		t.Errorf("server/keys.Title = %q, want %q", keys.Title, "API Keys")
	}
	if len(keys.Aliases) != 1 || keys.Aliases[0] != "key" {
		t.Errorf("server/keys.Aliases = %v, want [key]", keys.Aliases)
	}

	hidden, ok := srvDef.CommandGroups.Get("hidden")
	if !ok {
		t.Fatalf("commandGroups missing group %q", "hidden")
	}
	if !hidden.Hidden {
		t.Errorf("hidden.Hidden = false, want true")
	}

	// Per-command group paths survive.
	cmd, ok := srvDef.Commands.Get("StartServerCmd")
	if !ok {
		t.Fatalf("commands missing StartServerCmd")
	}
	if cmd.Group != "server" {
		t.Errorf("StartServerCmd.Group = %q, want %q", cmd.Group, "server")
	}

	cmd2, ok := srvDef.Commands.Get("CreateKeyCmd")
	if !ok {
		t.Fatalf("commands missing CreateKeyCmd")
	}
	if cmd2.Group != "server/keys" {
		t.Errorf("CreateKeyCmd.Group = %q, want %q", cmd2.Group, "server/keys")
	}

	// A command with no group keeps Group empty.
	ungrouped, ok := srvDef.Commands.Get("UngroupedCmd")
	if !ok {
		t.Fatalf("commands missing UngroupedCmd")
	}
	if ungrouped.Group != "" {
		t.Errorf("UngroupedCmd.Group = %q, want empty", ungrouped.Group)
	}
}
