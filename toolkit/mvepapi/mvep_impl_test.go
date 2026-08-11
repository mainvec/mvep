package mvep

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	api "github.com/mainvec/mvep/toolkit/mvepapi/api"
)

// chdirOnce moves the test process's working directory to the toolkit module
// root (the parent of the mvepapi package dir). go test ./mvepapi/ runs with
// cwd=mvepapi, but fixtures reference the embedded schema via relative $schema
// paths that resolve from the module root, exactly as in the root package's
// tests. sync.Once ensures the chdir happens once; the cwd is process-global
// across test functions, so a second chdir would stack.
var chdirOnce = sync.OnceFunc(func() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	_ = os.Chdir(filepath.Dir(wd))
})

func chdirToolkitRoot(t *testing.T) {
	t.Helper()
	chdirOnce()
}

// TestRunValidateCmdValid verifies the validate command returns a Valid result
// for a well-formed spec and populates no errors.
func TestRunValidateCmdValid(t *testing.T) {
	chdirToolkitRoot(t)
	spec := filepath.Join("testdata", "05_command_withfields.jsonc")
	res, err := runValidateCmd(context.Background(), &api.ValidateCmd{In: spec})
	if err != nil {
		t.Fatalf("runValidateCmd: %v", err)
	}
	if !res.Valid {
		t.Errorf("Valid = false, want true (spec is well-formed)")
	}
	if len(res.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", res.Errors)
	}
}

// TestRunValidateCmdInvalid verifies the validate command reports invalid and
// populates Errors for a malformed spec.
func TestRunValidateCmdInvalid(t *testing.T) {
	chdirToolkitRoot(t)
	spec := filepath.Join("testdata", "03_basic_wo_invalid.jsonc")
	res, err := runValidateCmd(context.Background(), &api.ValidateCmd{In: spec})
	if err != nil {
		t.Fatalf("runValidateCmd: %v", err)
	}
	if res.Valid {
		t.Errorf("Valid = true, want false (spec is malformed)")
	}
	if len(res.Errors) == 0 {
		t.Errorf("Errors empty, want at least one validation error")
	}
}

// TestRunValidateCmdMissingIn verifies a missing --in is a validation error
// (not the generic "command not implemented" stub error).
func TestRunValidateCmdMissingIn(t *testing.T) {
	_, err := runValidateCmd(context.Background(), &api.ValidateCmd{})
	if err == nil {
		t.Fatal("expected an error for missing --in, got nil")
	}
	if err.Error() == "command not implemented" {
		t.Fatalf("got generic stub error, want validation error: %v", err)
	}
}

// TestRunGenerateCmd verifies the generate command delegates to the generator
// and writes output.
func TestRunGenerateCmd(t *testing.T) {
	chdirToolkitRoot(t)
	outdir := t.TempDir()
	spec := filepath.Join("testdata", "05_command_withfields.jsonc")
	res, err := runGenerateCmd(context.Background(), &api.GenerateCmd{
		In:     spec,
		Lang:   "go",
		Outdir: outdir,
		Format: "plain",
	})
	if err != nil {
		t.Fatalf("runGenerateCmd: %v", err)
	}
	if res == nil {
		t.Fatal("runGenerateCmd returned nil result")
	}
	// The generated package file must exist.
	if _, err := os.Stat(filepath.Join(outdir, "api", "test5Name_package.go")); err != nil {
		t.Errorf("expected generated package file: %v", err)
	}
}

// TestRunInitializeCmd verifies the init command validates name/namespace and
// scaffolds a spec. It runs in a temp dir so the scaffolded file does not
// pollute the repo working tree.
func TestRunInitializeCmd(t *testing.T) {
	chdirToolkitRoot(t)
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	if _, err := runInitializeCmd(context.Background(), &api.InitializeCmd{Name: "svc", Namespace: "ns"}); err != nil {
		t.Fatalf("runInitializeCmd with valid args: %v", err)
	}
}

// TestRunInitializeCmdMissingName verifies init rejects a missing name (a
// validation error, not the generic stub error).
func TestRunInitializeCmdMissingName(t *testing.T) {
	_, err := runInitializeCmd(context.Background(), &api.InitializeCmd{Namespace: "ns"})
	if err == nil {
		t.Fatal("expected an error for missing name, got nil")
	}
	if err.Error() == "command not implemented" {
		t.Fatalf("got generic stub error, want validation error: %v", err)
	}
}

// TestRunInitializeCmdScaffoldsSpec verifies init writes a <name>.jsonc spec
// file into the current directory with a couple of dummy commands, and that
// the scaffolded spec is itself valid.
func TestRunInitializeCmdScaffoldsSpec(t *testing.T) {
	chdirToolkitRoot(t)
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}

	if _, err := runInitializeCmd(context.Background(), &api.InitializeCmd{Name: "myservice", Namespace: "myservicens"}); err != nil {
		t.Fatalf("runInitializeCmd: %v", err)
	}

	specPath := filepath.Join(dir, "myservice.jsonc")
	b, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("expected scaffolded spec file %s: %v", specPath, err)
	}
	src := string(b)

	// The scaffold must carry the name and namespace.
	if !strings.Contains(src, `"name": "myservice"`) {
		t.Errorf("scaffold missing name; got:\n%s", src)
	}
	if !strings.Contains(src, `"namespace": "myservicens"`) {
		t.Errorf("scaffold missing namespace; got:\n%s", src)
	}
	// It must include a couple of dummy commands.
	if !strings.Contains(src, `"commands"`) {
		t.Errorf("scaffold missing commands; got:\n%s", src)
	}

	// The scaffolded spec must itself validate.
	res, err := runValidateCmd(context.Background(), &api.ValidateCmd{In: specPath})
	if err != nil {
		t.Fatalf("validate scaffolded spec: %v", err)
	}
	if !res.Valid {
		t.Errorf("scaffolded spec is invalid: %v", res.Errors)
	}
}
