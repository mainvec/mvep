package toolkit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestExecuteGenerateInvalidSpecReturnsError is the T13 regression check:
// generation against an invalid spec must return an error, never os.Exit.
// Before the log.Fatal removal this test process would simply die here.
func TestExecuteGenerateInvalidSpecReturnsError(t *testing.T) {
	outdir := t.TempDir()
	err := ExecuteGenerate(context.Background(), filepath.Join("testdata", "03_basic_wo_invalid.jsonc"), outdir, "go", false, "plain")
	if err == nil {
		t.Fatal("expected an error for an invalid spec, got nil")
	}
}

// TestGenerateCompilePlain is the T15 regression net: the Go output produced
// from a valid fixture must compile. This is what would have caught the broken
// wo/ templates. It needs network module resolution, so it is skipped under
// -short to keep hermetic CI jobs fast.
func TestGenerateCompilePlain(t *testing.T) {
	if testing.Short() {
		t.Skip("needs network for go mod resolution")
	}

	outdir := t.TempDir()
	if err := ExecuteGenerate(context.Background(), filepath.Join("testdata", "05_command_withfields.jsonc"), outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	// The generated tree is not a module yet; give it one. The module name must
	// match the default import path the generator derives from the spec name
	// (see prepareTemplateDataMap) so the api + impl + commands packages resolve.
	writeFile(t, filepath.Join(outdir, "go.mod"), "module test5Name\n\ngo 1.24\n")

	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	// tidy resolves deps; build asserts the generated code compiles.
	run(outdir, "go", "mod", "tidy")
	run(outdir, "go", "build", "./...")

	// Generated files must not be world-writable.
	assertNotWorldWritable(t, outdir)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertNotWorldWritable(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0o002 != 0 {
			t.Errorf("%s is world-writable (%v)", path, info.Mode().Perm())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
