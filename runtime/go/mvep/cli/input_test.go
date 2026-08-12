package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveInputFromFile verifies T3: --input <path> reads the payload from
// the named file.
func TestResolveInputFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.json")
	if err := os.WriteFile(path, []byte(`{"in":"spec.json"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveInput(path, strings.NewReader("ignored"), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"in":"spec.json"}` {
		t.Errorf("payload = %q, want %q", got, `{"in":"spec.json"}`)
	}
}

// TestResolveInputExplicitStdin verifies T3: --input - reads from stdin.
func TestResolveInputExplicitStdin(t *testing.T) {
	got, err := resolveInput("-", strings.NewReader(`{"in":"stdin"}`), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"in":"stdin"}` {
		t.Errorf("payload = %q, want %q", got, `{"in":"stdin"}`)
	}
}

// TestResolveInputExplicitStdinPipe verifies #53: --input - reads stdin even
// when stdin is a pipe (the realistic way to use it). Explicit - and the
// implicit pipe are the same single consumer, so this must not error.
func TestResolveInputExplicitStdinPipe(t *testing.T) {
	got, err := resolveInput("-", strings.NewReader(`{"in":"pipe"}`), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"in":"pipe"}` {
		t.Errorf("payload = %q, want %q", got, `{"in":"pipe"}`)
	}
}

// TestResolveInputImplicitPipe verifies T3: with no --input and stdin not a
// character device, the payload is read implicitly from stdin.
func TestResolveInputImplicitPipe(t *testing.T) {
	got, err := resolveInput("", strings.NewReader(`{"in":"pipe"}`), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"in":"pipe"}` {
		t.Errorf("payload = %q, want %q", got, `{"in":"pipe"}`)
	}
}

// TestResolveInputEmptyStdin verifies T3: empty stdin yields an empty payload
// ({}), not a parse error.
func TestResolveInputEmptyStdin(t *testing.T) {
	got, err := resolveInput("", strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "{}" {
		t.Errorf("payload = %q, want %q", got, "{}")
	}
}

// TestResolveInputExplicitStdinEmpty verifies #53: empty stdin under --input -
// still yields {}.
func TestResolveInputExplicitStdinEmpty(t *testing.T) {
	got, err := resolveInput("-", strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "{}" {
		t.Errorf("payload = %q, want %q", got, "{}")
	}
}

// TestResolveInputTTYNoInput verifies T3: a TTY stdin with no --input does not
// block and yields an empty payload.
func TestResolveInputTTYNoInput(t *testing.T) {
	got, err := resolveInput("", strings.NewReader("should not be read"), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "{}" {
		t.Errorf("payload = %q, want %q", got, "{}")
	}
}

// TestResolveInputFileMissing verifies T3: a missing --input file errors.
func TestResolveInputFileMissing(t *testing.T) {
	_, err := resolveInput(filepath.Join(t.TempDir(), "nope.json"), bytes.NewReader(nil), true)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
