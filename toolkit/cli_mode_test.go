package toolkit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIModeDefaultIsRuntime verifies T15: the default (absent genopt) CLI
// mode is "runtime" — the flip from "legacy" has landed. The generated main
// uses cli.New, not the legacy prepareXxxCmd pattern.
func TestCLIModeDefaultIsRuntime(t *testing.T) {
	outdir := t.TempDir()
	// Fixture 05 has no gen_options.cli, so the default is "runtime".
	if err := ExecuteGenerate(context.Background(), filepath.Join("testdata", "05_command_withfields.jsonc"), outdir, "go", false, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	mainFile := filepath.Join(outdir, "cmd", "test5Name", "test5Name_main_cmd.go")
	b, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("main cmd file not generated: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "cli.New") {
		t.Error("default (runtime) main should call cli.New")
	}
	if strings.Contains(s, "prepareOrderPizzaCmd") {
		t.Error("default (runtime) main should not use the legacy prepareXxxCmd pattern")
	}
}

// TestCLIModeLegacyExplicit verifies T15: gen_options.cli=legacy still
// generates the legacy hand-wired main.
func TestCLIModeLegacyExplicit(t *testing.T) {
	outdir := t.TempDir()
	specFile := filepath.Join(outdir, "legacy_spec.jsonc")
	spec := `{
		"$id": "legacymode",
		"$schema": "resources/mvepspec/0.2/schema/2026-01-15",
		"name": "legacymode",
		"namespace": "legns",
		"gen_options": { "cli": "legacy", "format": "plain" },
		"commands": {
			"EchoCmd": {
				"fields": {
					"msg": { "fnum": 1, "type": "string" }
				}
			}
		}
	}`
	if err := os.WriteFile(specFile, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if err := ExecuteGenerate(context.Background(), specFile, outdir, "go", false, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	mainFile := filepath.Join(outdir, "cmd", "legacymode", "legacymode_main_cmd.go")
	b, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("legacy main cmd file not generated: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "package main") {
		t.Error("legacy main should be package main")
	}
	if !strings.Contains(s, "github.com/mainvec/ugo/cli") {
		t.Error("legacy main should import ugo/cli")
	}
}

// TestCLIModeNone verifies T15: skipCmd=true forces "none" mode — no CLI main
// is generated, regardless of any genopt.
func TestCLIModeNone(t *testing.T) {
	outdir := t.TempDir()
	if err := ExecuteGenerate(context.Background(), filepath.Join("testdata", "05_command_withfields.jsonc"), outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	mainFile := filepath.Join(outdir, "cmd", "test5Name", "test5Name_main_cmd.go")
	if _, err := os.Stat(mainFile); !os.IsNotExist(err) {
		t.Errorf("skipCmd=true should not generate a CLI main; file exists: %v", err)
	}
}

// TestCLIModeRuntime verifies T15: gen_options.cli=runtime generates a main
// that builds the CLI via cli.New from the runtime descriptor, instead of the
// legacy hand-wired per-command pattern.
func TestCLIModeRuntime(t *testing.T) {
	outdir := t.TempDir()
	// Create a fixture with gen_options.cli: runtime.
	specFile := filepath.Join(outdir, "runtime_spec.jsonc")
	spec := `{
		"$id": "rtmode",
		"$schema": "resources/mvepspec/0.2/schema/2026-01-15",
		"name": "rtmode",
		"namespace": "rtns",
		"gen_options": { "cli": "runtime", "format": "plain" },
		"commands": {
			"EchoCmd": {
				"fields": {
					"msg": { "fnum": 1, "type": "string" }
				}
			}
		}
	}`
	if err := os.WriteFile(specFile, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	// Run from the toolkit directory so the $schema path resolves.
	wd, _ := os.Getwd()
	specPath := specFile
	if !filepath.IsAbs(specPath) {
		specPath = filepath.Join(wd, specPath)
	}
	// ExecuteGenerate resolves the spec path relative to wd; since we wrote
	// to a TempDir, pass the absolute path.
	if err := ExecuteGenerate(context.Background(), specPath, outdir, "go", false, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	mainFile := filepath.Join(outdir, "cmd", "rtmode", "rtmode_main_cmd.go")
	b, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("runtime main cmd file not generated: %v", err)
	}
	s := string(b)
	// Runtime main should use cli.New from the runtime descriptor.
	if !strings.Contains(s, "cli.New") {
		t.Error("runtime main should call cli.New")
	}
	// It should NOT use the legacy prepareXxxCmd pattern.
	if strings.Contains(s, "prepareEchoCmd") {
		t.Error("runtime main should not use the legacy prepareXxxCmd pattern")
	}
}

// TestCLIModeNoneGenopt verifies T15: gen_options.cli=none skips CLI main
// generation even when skipCmd is false.
func TestCLIModeNoneGenopt(t *testing.T) {
	outdir := t.TempDir()
	specFile := filepath.Join(outdir, "none_spec.jsonc")
	spec := `{
		"$id": "nonemode",
		"$schema": "resources/mvepspec/0.2/schema/2026-01-15",
		"name": "nonemode",
		"namespace": "nonens",
		"gen_options": { "cli": "none", "format": "plain" },
		"commands": {
			"EchoCmd": {
				"fields": {
					"msg": { "fnum": 1, "type": "string" }
				}
			}
		}
	}`
	if err := os.WriteFile(specFile, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if err := ExecuteGenerate(context.Background(), specFile, outdir, "go", false, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	mainFile := filepath.Join(outdir, "cmd", "nonemode", "nonemode_main_cmd.go")
	if _, err := os.Stat(mainFile); !os.IsNotExist(err) {
		t.Errorf("cli=none should not generate a CLI main; file exists")
	}
}
