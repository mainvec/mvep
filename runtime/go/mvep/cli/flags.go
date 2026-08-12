package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/ugo/cli"
)

// flagBinding holds a parsed flag value and the accessor to write it into the
// command struct, for one command execution. The type-switch in bindFlags
// inspects the FieldType (and Repeated flag) to select the right FlagSet var
// helper.
type flagBinding struct {
	field *mvep.FieldDesc
	// apply writes the parsed value into the command struct via the field's
	// Ptr accessor. It is called after flag parsing completes. A non-nil
	// error indicates the parsed value could not be written (e.g. invalid
	// JSON for --*-json flags); the caller aborts before dispatch.
	apply func(cmd any) error
}

// bindFlags registers a flag for every FieldDesc on the command's FlagSet and
// returns the bindings so applyBindings can write parsed values into the struct.
// This does NOT mutate the descriptor's Ptr; parsed values are held in
// per-execution locals and written back via Ptr after parsing.
//
// The type-switch keys on FieldType plus the Repeated flag, because FieldType
// alone cannot distinguish FieldSint32 from FieldInt32 (both return *int32)
// or FieldString repeated from FieldString scalar.
//
// desc is the owning PackageDesc, needed to resolve name-only FieldDesc.Ref
// to the full RecordDesc in Records (codegen emits Ref name-only to avoid
// field duplication; see #28).
func bindFlags(fs *cli.FlagSet, cmdDesc *mvep.CommandDesc, desc *mvep.PackageDesc) []flagBinding {
	var bindings []flagBinding
	for i := range cmdDesc.Fields {
		f := &cmdDesc.Fields[i]
		bindings = append(bindings, registerFlag(fs, f, desc))
	}
	return bindings
}

// registerFlag registers a single flag for a field, type-switching on the
// FieldType (and Repeated) to select the right FlagSet var helper. This is the
// T9 full binding covering every spec FieldType.
func registerFlag(fs *cli.FlagSet, f *mvep.FieldDesc, desc *mvep.PackageDesc) flagBinding {
	if f.Repeated {
		return registerRepeatedFlag(fs, f, desc)
	}

	switch f.Type {
	case mvep.FieldString:
		p := new(string)
		fs.StringVar(p, f.Name, "", f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*string); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldBool:
		p := new(bool)
		fs.BoolVar(p, f.Name, false, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*bool); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldInt32, mvep.FieldSint32:
		p := new(int32)
		fs.Int32Var(p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*int32); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldInt64:
		p := new(int64)
		fs.Int64Var(p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*int64); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldUint32:
		p := new(uint32)
		Uint32Var(fs, p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*uint32); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldFloat:
		p := new(float32)
		Float32Var(fs, p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*float32); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldDouble:
		p := new(float64)
		fs.Float64Var(p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*float64); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldBytes:
		p := new([]byte)
		fs.BytesVar(p, f.Name, nil, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*[]byte); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldTimestamp:
		p := new(time.Time)
		fs.Var(&timeValue{p: p}, f.Name, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*time.Time); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldDuration:
		p := new(time.Duration)
		fs.DurationVar(p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*time.Duration); ok {
				*t = *p
			}
			return nil
		}}
	case mvep.FieldUUID:
		// UUID is bound as a string flag; the value is written via
		// encoding.TextUnmarshaler (uuid.UUID implements it), avoiding a
		// direct dependency on google/uuid in mvep/cli.
		p := new(string)
		fs.StringVar(p, f.Name, "", f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if *p == "" {
				return nil
			}
			target := f.Ptr(cmd)
			if u, ok := target.(interface{ UnmarshalText([]byte) error }); ok {
				return u.UnmarshalText([]byte(*p))
			}
			return nil
		}}
	case mvep.FieldMap:
		// Maps bind from a JSON object via --<name>-json.
		p := new(string)
		fs.StringVar(p, f.Name+"-json", "", f.Desc+" (JSON object)")
		return flagBinding{field: f, apply: func(cmd any) error {
			if *p == "" {
				return nil
			}
			target := f.Ptr(cmd)
			if err := json.Unmarshal([]byte(*p), target); err != nil {
				return fmt.Errorf("--%s-json: %w", f.Name, err)
			}
			return nil
		}}
	case mvep.FieldRecord:
		// Records flatten to depth 1: each record field becomes
		// --<name>-<field> and the record struct is constructed and filled.
		return registerRecordFlag(fs, f, desc)
	default:
		// An unhandled type is a generate-time bug (T5 catches it at codegen).
		// Register a string placeholder so the parser doesn't reject the flag.
		p := new(string)
		fs.StringVar(p, f.Name, "", f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error { return nil }}
	}
}

// registerRepeatedFlag handles repeated fields, which produce slice types.
func registerRepeatedFlag(fs *cli.FlagSet, f *mvep.FieldDesc, desc *mvep.PackageDesc) flagBinding {
	switch f.Type {
	case mvep.FieldString:
		p := new([]string)
		fs.StringSliceVar(p, f.Name, nil, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) error {
			if t, ok := f.Ptr(cmd).(*[]string); ok {
				*t = *p
			}
			return nil
		}}
	default:
		// Repeated non-string types: bind as a JSON array via --<name>-json.
		p := new(string)
		fs.StringVar(p, f.Name+"-json", "", f.Desc+" (JSON array)")
		return flagBinding{field: f, apply: func(cmd any) error {
			if *p == "" {
				return nil
			}
			target := f.Ptr(cmd)
			if err := json.Unmarshal([]byte(*p), target); err != nil {
				return fmt.Errorf("--%s-json: %w", f.Name, err)
			}
			return nil
		}}
	}
}

// registerRecordFlag flattens a $ref record field to depth 1. Each record
// field becomes --<name>-<subField>; the record struct is constructed and
// filled after parsing via JSON unmarshal (which handles nil pointer
// construction).
//
// Codegen emits Ref name-only (Ref.Fields is empty) to avoid duplicating field
// data; the full fields live in PackageDesc.Records. When Ref.Fields is empty,
// the record is resolved via desc.Record(Ref.Name) (#28). If the record is not
// found, the flag falls back to --<name>-json.
func registerRecordFlag(fs *cli.FlagSet, f *mvep.FieldDesc, desc *mvep.PackageDesc) flagBinding {
	// Resolve the record fields. Codegen emits Ref name-only; the full fields
	// are in PackageDesc.Records, resolvable via Record(name).
	var refFields []mvep.FieldDesc
	if f.Ref != nil {
		refFields = f.Ref.Fields
		if len(refFields) == 0 && desc != nil {
			if rec, ok := desc.Record(f.Ref.Name); ok {
				refFields = rec.Fields
			}
		}
	}

	if len(refFields) == 0 {
		// No record fields to flatten; bind as a JSON object via --<name>-json.
		p := new(string)
		fs.StringVar(p, f.Name+"-json", "", f.Desc+" (JSON object)")
		return flagBinding{field: f, apply: func(cmd any) error {
			if *p == "" {
				return nil
			}
			target := f.Ptr(cmd)
			if err := json.Unmarshal([]byte(*p), target); err != nil {
				return fmt.Errorf("--%s-json: %w", f.Name, err)
			}
			return nil
		}}
	}

	// Depth-1 flattening: register --<name>-<subField> for each record field.
	// Each sub-field is registered with the appropriate FlagSet helper based
	// on its FieldType, not as a blanket StringVar — so bool fields get
	// BoolVar (no argument needed), int fields get Int32Var, etc. The parsed
	// values are encoded as json.RawMessage per FieldType so the unmarshal
	// into the record struct uses the right JSON type (#36).
	type subBinding struct {
		field  *mvep.FieldDesc
		rawVal func() (json.RawMessage, error) // returns the JSON-encoded value
		isSet  func() bool
	}
	var subs []subBinding
	for i := range refFields {
		rf := &refFields[i]
		subs = append(subs, registerSubFieldFlag(fs, f.Name, rf))
	}

	return flagBinding{field: f, apply: func(cmd any) error {
		anySet := false
		for _, s := range subs {
			if s.isSet() {
				anySet = true
				break
			}
		}
		if !anySet {
			return nil
		}
		// Build a JSON object with per-field typed values and unmarshal into
		// the target (Ptr returns **Record; json.Unmarshal handles nil
		// pointer construction). Using json.RawMessage avoids the #36 bug
		// where all values were JSON strings.
		obj := make(map[string]json.RawMessage, len(subs))
		for _, s := range subs {
			if !s.isSet() {
				continue
			}
			raw, err := s.rawVal()
			if err != nil {
				// The error already names the sub-field flag (e.g.
				// --<record>-<field>-json) so it is surfaced as-is rather
				// than wrapped in the parent's opaque --<record> error.
				return err
			}
			obj[s.field.Name] = raw
		}
		jsonBytes, _ := json.Marshal(obj)
		if err := json.Unmarshal(jsonBytes, f.Ptr(cmd)); err != nil {
			return fmt.Errorf("--%s: %w", f.Name, err)
		}
		return nil
	}}
}

// registerSubFieldFlag registers a single depth-1 record sub-field flag with the
// appropriate FlagSet helper based on the field's FieldType. It returns closures
// for reading the parsed value as json.RawMessage (so the JSON object built for
// unmarshal uses the right type per field — #36) and checking whether the flag
// was set.
func registerSubFieldFlag(fs *cli.FlagSet, parentName string, rf *mvep.FieldDesc) struct {
	field  *mvep.FieldDesc
	rawVal func() (json.RawMessage, error)
	isSet  func() bool
} {
	flagName := parentName + "-" + rf.Name
	result := struct {
		field  *mvep.FieldDesc
		rawVal func() (json.RawMessage, error)
		isSet  func() bool
	}{field: rf}

	// Repeated sub-fields bind via their own branch BEFORE the type switch,
	// mirroring registerFlag: the scalar switch below would otherwise fall a
	// repeated string into the FieldString case and bind a single value,
	// which cannot unmarshal into the []string the record struct expects.
	if rf.Repeated {
		return registerRepeatedSubFieldFlag(fs, parentName, rf)
	}

	switch rf.Type {
	case mvep.FieldString, mvep.FieldUUID, mvep.FieldTimestamp, mvep.FieldBytes:
		p := new(string)
		fs.StringVar(p, flagName, "", rf.Desc)
		result.isSet = func() bool { return *p != "" }
		result.rawVal = func() (json.RawMessage, error) { return encodeJSONString(*p), nil }
	case mvep.FieldBool:
		p := new(bool)
		fs.BoolVar(p, flagName, false, rf.Desc)
		result.isSet = func() bool { return *p }
		result.rawVal = func() (json.RawMessage, error) {
			if *p {
				return json.RawMessage("true"), nil
			}
			return json.RawMessage("false"), nil
		}
	case mvep.FieldInt32, mvep.FieldSint32:
		p := new(int32)
		fs.Int32Var(p, flagName, 0, rf.Desc)
		result.isSet = func() bool { return *p != 0 }
		result.rawVal = func() (json.RawMessage, error) { return encodeJSONNumber(int64(*p)), nil }
	case mvep.FieldInt64:
		p := new(int64)
		fs.Int64Var(p, flagName, 0, rf.Desc)
		result.isSet = func() bool { return *p != 0 }
		result.rawVal = func() (json.RawMessage, error) { return encodeJSONNumber(*p), nil }
	case mvep.FieldUint32:
		p := new(uint32)
		Uint32Var(fs, p, flagName, 0, rf.Desc)
		result.isSet = func() bool { return *p != 0 }
		result.rawVal = func() (json.RawMessage, error) { return encodeJSONNumber(int64(*p)), nil }
	case mvep.FieldFloat:
		p := new(float32)
		Float32Var(fs, p, flagName, 0, rf.Desc)
		result.isSet = func() bool { return *p != 0 }
		result.rawVal = func() (json.RawMessage, error) { return encodeJSONFloat(float64(*p)), nil }
	case mvep.FieldDouble:
		p := new(float64)
		fs.Float64Var(p, flagName, 0, rf.Desc)
		result.isSet = func() bool { return *p != 0 }
		result.rawVal = func() (json.RawMessage, error) { return encodeJSONFloat(*p), nil }
	case mvep.FieldDuration:
		p := new(string)
		fs.StringVar(p, flagName, "", rf.Desc)
		result.isSet = func() bool { return *p != "" }
		result.rawVal = func() (json.RawMessage, error) { return encodeJSONString(*p), nil }
	default:
		// Fallback: treat as string.
		p := new(string)
		fs.StringVar(p, flagName, "", rf.Desc)
		result.isSet = func() bool { return *p != "" }
		result.rawVal = func() (json.RawMessage, error) { return encodeJSONString(*p), nil }
	}
	return result
}

// registerRepeatedSubFieldFlag binds a repeated depth-1 record sub-field. The
// string-like set must match the scalar switch (FieldString, FieldUUID,
// FieldTimestamp, FieldDuration) or the two paths drift again for the same
// field type.
//
// - Repeated string-like types bind via StringSliceVar, with rawVal emitting a
//   JSON array (json.Marshal of the slice).
// - Repeated non-string types (numeric, bool, bytes, map, recRef) bind from
//   --<record>-<field>-json, passing the array through as json.RawMessage.
func registerRepeatedSubFieldFlag(fs *cli.FlagSet, parentName string, rf *mvep.FieldDesc) struct {
	field  *mvep.FieldDesc
	rawVal func() (json.RawMessage, error)
	isSet  func() bool
} {
	flagName := parentName + "-" + rf.Name
	result := struct {
		field  *mvep.FieldDesc
		rawVal func() (json.RawMessage, error)
		isSet  func() bool
	}{field: rf}

	switch rf.Type {
	case mvep.FieldString, mvep.FieldUUID, mvep.FieldTimestamp, mvep.FieldDuration:
		p := new([]string)
		fs.StringSliceVar(p, flagName, nil, rf.Desc)
		result.isSet = func() bool { return len(*p) > 0 }
		result.rawVal = func() (json.RawMessage, error) {
			b, _ := json.Marshal(*p)
			return b, nil
		}
	default:
		// Repeated non-string: bind as a JSON array via --<record>-<field>-json.
		// Malformed JSON is validated in the rawVal closure so the error names
		// the sub-field flag rather than surfacing as the parent's opaque
		// --<record>: json: cannot unmarshal ... error.
		p := new(string)
		fs.StringVar(p, flagName+"-json", "", rf.Desc+" (JSON array)")
		result.isSet = func() bool { return *p != "" }
		result.rawVal = func() (json.RawMessage, error) {
			var raw json.RawMessage
			if err := json.Unmarshal([]byte(*p), &raw); err != nil {
				return nil, fmt.Errorf("--%s-json: %w", flagName, err)
			}
			return raw, nil
		}
	}
	return result
}

// encodeJSONString returns a JSON-encoded string literal.
func encodeJSONString(s string) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

// encodeJSONNumber returns a JSON-encoded integer.
func encodeJSONNumber(n int64) json.RawMessage {
	return json.RawMessage(strconv.FormatInt(n, 10))
}

// encodeJSONFloat returns a JSON-encoded float.
func encodeJSONFloat(f float64) json.RawMessage {
	return json.RawMessage(strconv.FormatFloat(f, 'g', -1, 64))
}

// FieldDesc.Ptr accessors. Returns the first error encountered (e.g. invalid
// JSON for a --*-json flag); remaining bindings are still applied.
func applyBindings(cmd any, bindings []flagBinding) error {
	var firstErr error
	for _, b := range bindings {
		if err := b.apply(cmd); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// timeValue is a flag.Value for time.Time, parsing RFC3339 timestamps.
type timeValue struct{ p *time.Time }

func (t *timeValue) Set(s string) error {
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("invalid timestamp %q: %w", s, err)
	}
	*t.p = parsed
	return nil
}

func (t *timeValue) Get() any { return *t.p }

func (t *timeValue) String() string {
	if t == nil || t.p == nil || t.p.IsZero() {
		return ""
	}
	return t.p.Format(time.RFC3339)
}
