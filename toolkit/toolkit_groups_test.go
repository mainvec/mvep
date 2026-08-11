package toolkit_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// TestCommandGroupsDescriptorEmission is the T4 check: generating the grouped
// fixture emits a pkgDesc whose Groups slice and per-command Group fields carry
// the group metadata. It reads the generated *_package.go rather than calling
// unexported toolkit helpers, exercising the real codegen path.
func TestCommandGroupsDescriptorEmission(t *testing.T) {
	outdir := t.TempDir()
	if err := toolkit.ExecuteGenerate(t.Context(), filepath.Join("testdata", "13_command_groups.jsonc"), outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(outdir, "api", "test13_package.go"))
	if err != nil {
		t.Fatalf("read generated package: %v", err)
	}
	src := string(got)

	// The generated package must contain the groups surface: auto-created
	// intermediates (server for server/keys), declared metadata, and hidden.
	// gofmt aligns struct fields, so match on the value with flexible spacing.
	for _, want := range []string{
		`Path: "server"`,
		`Path: "server/keys"`,
		`Name: "keys"`,
		`Title: "LLM Servers"`,
		`Aliases: []string{"key"}`,
		`Hidden: true`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated package missing %q\n---\n%s", want, src)
		}
	}
	// Group is gofmt-aligned, so match with a regexp.
	groupRe := regexp.MustCompile(`Group:\s+"server/keys"`)
	if !groupRe.MatchString(src) {
		t.Errorf("generated package missing Group: \"server/keys\"\n---\n%s", src)
	}

	// A root command must not carry a group: the round-trip test covers the
	// empty-Group case; here we only assert the grouped surface is emitted.
}

// TestCommandGroupsNoGroupBackwardCompat is the T4/T7 backward-compat guard: a
// spec with no groups must emit a pkgDesc with no Groups field at all, so
// existing consumers generate byte-identical output.
func TestCommandGroupsNoGroupBackwardCompat(t *testing.T) {
	outdir := t.TempDir()
	if err := toolkit.ExecuteGenerate(t.Context(), filepath.Join("testdata", "05_command_withfields.jsonc"), outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(outdir, "api", "test5Name_package.go"))
	if err != nil {
		t.Fatalf("read generated package: %v", err)
	}
	src := string(got)

	if strings.Contains(src, "Groups:") {
		t.Errorf("no-group spec emitted a Groups field; output should be unchanged:\n%s", src)
	}
	if strings.Contains(src, "Group:") {
		t.Errorf("no-group spec emitted a CommandDesc.Group field; output should be unchanged:\n%s", src)
	}
}
