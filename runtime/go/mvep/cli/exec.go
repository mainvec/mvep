package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/ugo/cli"
)

// registerExec adds the `exec` verb to the reserved namespace group:
// `svc mvep exec <command> [--input <path>|-]`. It reads a complete command
// payload from a file, explicit stdin, or implicitly from a pipe, validates the
// payload keys against the descriptor, decodes via the server's encoding
// registry, and runs the shared tail.
func (a *App) registerExec(ns *cli.Command) {
	var input string
	exec := &cli.Command{
		Usage: "exec <command>",
		Short: "run a command from a JSON payload",
		Long: `Reads a complete command payload from a file (--input <path>),
explicit stdin (--input -), or implicitly from a pipe when stdin is not a
terminal. The payload is validated against the command descriptor and decoded
with the same encoder the server uses.

Flags must precede the command name (stdlib flag semantics): the payload is
read from stdin when --input is absent and stdin is not a terminal.`,
		Args: func(cmd *cli.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("mvep exec requires exactly one command name")
			}
			return nil
		},
	}
	exec.Flags().StringVar(&input, "input", "", "read payload from a file, or '-' for stdin")
	exec.RunE = func(ctx *cli.Context, args []string) error {
		return a.execVerb(ctx, args[0], input)
	}
	ns.AddCommand(exec)
}

// execVerb resolves the payload from --input or implicit stdin, dispatches it,
// and renders the result (already handled by the shared tail).
func (a *App) execVerb(ctx *cli.Context, name, input string) error {
	payload, err := resolveInput(input, a.stdin, a.stdinIsTTY())
	if err != nil {
		return err
	}
	_, err = a.dispatch(ctx, name, payload)
	return err
}

// stdinIsTTY reports whether the configured stdin is backed by a character
// device (a TTY), guarding against Stat errors which occur on some CI runners.
// A non-*os.File reader (e.g. one injected for tests) is treated as a pipe so
// implicit stdin is read.
func (a *App) stdinIsTTY() bool {
	f, ok := a.stdin.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// validatePayloadKeys enforces strict key checking against the descriptor.
// oenc's json decoder cannot reject unknown fields, so unknown keys would
// otherwise be silently dropped — the worst failure mode for a scripting
// surface. Keys are normalised (case + underscores) on both sides because
// protojson accepts both snake_case and lowerCamelCase and encoding/json
// already matches case-insensitively.
func validatePayloadKeys(cmdDesc *mvep.CommandDesc, desc *mvep.PackageDesc, payload []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(payload, &obj); err != nil {
		return nil // not a JSON object; the decoder will produce a clearer error
	}
	return validateFields(obj, cmdDesc.Fields, desc, "")
}

// validateFields checks the keys of a payload object against a set of
// FieldDesc, recursing into record Ref fields. Unknown keys are a hard error.
// Map field values are entered but their keys are not validated (arbitrary by
// definition). An unresolved Ref stops the walk for that subtree.
func validateFields(obj map[string]json.RawMessage, fields []mvep.FieldDesc, desc *mvep.PackageDesc, path string) error {
	known := make(map[string]bool, len(fields))
	for i := range fields {
		known[normalizeKey(fields[i].Name)] = true
	}

	for key, raw := range obj {
		nk := normalizeKey(key)
		if !known[nk] {
			if path != "" {
				return fmt.Errorf("unknown key %q in %s", key, path)
			}
			return fmt.Errorf("unknown key %q", key)
		}

		// Recurse into record fields for fields carrying a Ref, including each
		// element of a repeated record.
		for i := range fields {
			f := &fields[i]
			if f.Type != mvep.FieldRecord || f.Ref == nil {
				continue
			}
			if normalizeKey(f.Name) != nk {
				continue
			}
			if err := validateRecordValue(raw, f, desc, path+key); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// validateRecordValue walks a record field's payload value, entering each
// element of a repeated record. Map field values inside the record are entered
// without validating their keys.
func validateRecordValue(raw json.RawMessage, f *mvep.FieldDesc, desc *mvep.PackageDesc, path string) error {
	// Resolve the record fields, mirroring registerRecordFlag: Ref may carry
	// fields directly or only a name resolved via desc.Record.
	var refFields []mvep.FieldDesc
	if f.Ref.Fields != nil {
		refFields = f.Ref.Fields
	} else if rec, ok := desc.Record(f.Ref.Name); ok {
		refFields = rec.Fields
	}
	if len(refFields) == 0 {
		// Unresolved Ref: stop the walk rather than reject the subtree.
		return nil
	}

	if f.Repeated {
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil // not an array; the decoder will report it
		}
		for _, elem := range arr {
			if err := validateObjectValue(elem, refFields, desc, path); err != nil {
				return err
			}
		}
		return nil
	}

	return validateObjectValue(raw, refFields, desc, path)
}

// validateObjectValue validates a single record object's keys against refFields.
func validateObjectValue(raw json.RawMessage, refFields []mvep.FieldDesc, desc *mvep.PackageDesc, path string) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil // not an object; the decoder will report it
	}
	return validateFields(obj, refFields, desc, path+".")
}

// normalizeKey lowers the string and removes underscores, so "args_template"
// and "ArgsTemplate" compare equal (matching protojson's dual naming and
// encoding/json's case-insensitive fallback).
func normalizeKey(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "_", "")
}
