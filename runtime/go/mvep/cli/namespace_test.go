package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// TestNamespaceAppearsInHelp verifies T2: the reserved namespace group appears
// in root help.
func TestNamespaceAppearsInHelp(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex)

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"--help"}, &stdout, &stderr)
	combined := strings.ToLower(stdout.String() + stderr.String())
	if !strings.Contains(combined, "mvep") {
		t.Errorf("help should list the mvep namespace; got: %s", combined)
	}
}

// TestNamespaceOverrideRelocates verifies T2: the WithNamespace option
// relocates the reserved namespace to a custom name.
func TestNamespaceOverrideRelocates(t *testing.T) {
	ex := &recordingExecutor{}
	app := New(&t9AllTypesDesc, ex, WithNamespace("acme"))

	var stdout, stderr bytes.Buffer
	_ = app.RunWithIO(context.Background(), []string{"--help"}, &stdout, &stderr)
	combined := strings.ToLower(stdout.String() + stderr.String())
	if !strings.Contains(combined, "acme") {
		t.Errorf("help should list the acme namespace; got: %s", combined)
	}
	if strings.Contains(combined, "mvep") {
		t.Errorf("help should not list the default mvep namespace after override; got: %s", combined)
	}
}

// TestNamespaceCommandCollisionPanics verifies T2: a descriptor declaring a
// command named after the namespace panics at construction, naming the
// collision.
func TestNamespaceCommandCollisionPanics(t *testing.T) {
	desc := mvep.PackageDesc{
		Name: "collide",
		Commands: []mvep.CommandDesc{
			{Name: "MvepCmd", Alias: "mvep", New: func() any { return &gRootCmd{} }},
		},
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for namespace command collision, got nil")
		}
		if !strings.Contains(r.(string), "mvep") {
			t.Errorf("panic should name the reserved word; got: %v", r)
		}
	}()
	New(&desc, &recordingExecutor{})
}

// TestNamespaceGroupCollisionPanics verifies T2: a descriptor declaring a group
// named after the namespace panics at construction, naming the collision.
func TestNamespaceGroupCollisionPanics(t *testing.T) {
	desc := mvep.PackageDesc{
		Name: "collide",
		Commands: []mvep.CommandDesc{
			{Name: "RootCmd", New: func() any { return &gRootCmd{} }},
		},
		Groups: []mvep.GroupDesc{
			{Path: "mvep", Name: "mvep"},
		},
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for namespace group collision, got nil")
		}
		if !strings.Contains(r.(string), "mvep") {
			t.Errorf("panic should name the reserved word; got: %v", r)
		}
	}()
	New(&desc, &recordingExecutor{})
}

// TestNamespaceNoCollisionUnchanged verifies T2: a spec using none of the
// namespace machinery produces an otherwise unchanged tree (existing groups
// still dispatch).
func TestNamespaceNoCollisionUnchanged(t *testing.T) {
	ex := &recordingExecutor{result: &gServerCmdResult{PID: 42}}
	app := New(&gDesc, ex)

	var stdout, stderr bytes.Buffer
	err := app.RunWithIO(context.Background(), []string{"server", "start", "--model", "llama3"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, ok := ex.gotCmd.(*gServerCmd)
	if !ok {
		t.Fatalf("executor received %T, want *gServerCmd", ex.gotCmd)
	}
	if cmd.Model != "llama3" {
		t.Errorf("Model = %q, want %q", cmd.Model, "llama3")
	}
}
