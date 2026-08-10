package cli

import (
	"encoding/json"
	"fmt"
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
	// Ptr accessor. It is called after flag parsing completes.
	apply func(cmd any)
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
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*string); ok {
				*t = *p
			}
		}}
	case mvep.FieldBool:
		p := new(bool)
		fs.BoolVar(p, f.Name, false, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*bool); ok {
				*t = *p
			}
		}}
	case mvep.FieldInt32, mvep.FieldSint32:
		p := new(int32)
		fs.Int32Var(p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*int32); ok {
				*t = *p
			}
		}}
	case mvep.FieldInt64:
		p := new(int64)
		fs.Int64Var(p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*int64); ok {
				*t = *p
			}
		}}
	case mvep.FieldUint32:
		p := new(uint32)
		Uint32Var(fs, p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*uint32); ok {
				*t = *p
			}
		}}
	case mvep.FieldFloat:
		p := new(float32)
		Float32Var(fs, p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*float32); ok {
				*t = *p
			}
		}}
	case mvep.FieldDouble:
		p := new(float64)
		fs.Float64Var(p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*float64); ok {
				*t = *p
			}
		}}
	case mvep.FieldBytes:
		p := new([]byte)
		fs.BytesVar(p, f.Name, nil, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*[]byte); ok {
				*t = *p
			}
		}}
	case mvep.FieldTimestamp:
		p := new(time.Time)
		fs.Var(&timeValue{p: p}, f.Name, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*time.Time); ok {
				*t = *p
			}
		}}
	case mvep.FieldDuration:
		p := new(time.Duration)
		fs.DurationVar(p, f.Name, 0, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*time.Duration); ok {
				*t = *p
			}
		}}
	case mvep.FieldUUID:
		// UUID is bound as a string flag; the value is written via
		// encoding.TextUnmarshaler (uuid.UUID implements it), avoiding a
		// direct dependency on google/uuid in mvep/cli.
		p := new(string)
		fs.StringVar(p, f.Name, "", f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if *p == "" {
				return
			}
			target := f.Ptr(cmd)
			if u, ok := target.(interface{ UnmarshalText([]byte) error }); ok {
				_ = u.UnmarshalText([]byte(*p))
			}
		}}
	case mvep.FieldMap:
		// Maps bind from a JSON object via --<name>-json.
		p := new(string)
		fs.StringVar(p, f.Name+"-json", "", f.Desc+" (JSON object)")
		return flagBinding{field: f, apply: func(cmd any) {
			if *p == "" {
				return
			}
			target := f.Ptr(cmd)
			// Unmarshal into the map the Ptr points at.
			_ = json.Unmarshal([]byte(*p), target)
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
		return flagBinding{field: f, apply: func(cmd any) {}}
	}
}

// registerRepeatedFlag handles repeated fields, which produce slice types.
func registerRepeatedFlag(fs *cli.FlagSet, f *mvep.FieldDesc, desc *mvep.PackageDesc) flagBinding {
	switch f.Type {
	case mvep.FieldString:
		p := new([]string)
		fs.StringSliceVar(p, f.Name, nil, f.Desc)
		return flagBinding{field: f, apply: func(cmd any) {
			if t, ok := f.Ptr(cmd).(*[]string); ok {
				*t = *p
			}
		}}
	default:
		// Repeated non-string types: bind as a JSON array via --<name>-json.
		p := new(string)
		fs.StringVar(p, f.Name+"-json", "", f.Desc+" (JSON array)")
		return flagBinding{field: f, apply: func(cmd any) {
			if *p == "" {
				return
			}
			target := f.Ptr(cmd)
			_ = json.Unmarshal([]byte(*p), target)
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
	refFields := f.Ref.Fields
	if len(refFields) == 0 && f.Ref != nil && desc != nil {
		if rec, ok := desc.Record(f.Ref.Name); ok {
			refFields = rec.Fields
		}
	}

	if len(refFields) == 0 {
		// No record fields to flatten; bind as a JSON object via --<name>-json.
		p := new(string)
		fs.StringVar(p, f.Name+"-json", "", f.Desc+" (JSON object)")
		return flagBinding{field: f, apply: func(cmd any) {
			if *p == "" {
				return
			}
			target := f.Ptr(cmd)
			_ = json.Unmarshal([]byte(*p), target)
		}}
	}

	// Depth-1 flattening: register --<name>-<subField> for each record field.
	type subBinding struct {
		fieldName string
		strVal    *string
	}
	var subs []subBinding
	for i := range refFields {
		rf := &refFields[i]
		p := new(string)
		flagName := f.Name + "-" + rf.Name
		fs.StringVar(p, flagName, "", rf.Desc)
		subs = append(subs, subBinding{fieldName: rf.Name, strVal: p})
	}

	return flagBinding{field: f, apply: func(cmd any) {
		anySet := false
		for _, s := range subs {
			if *s.strVal != "" {
				anySet = true
				break
			}
		}
		if !anySet {
			return
		}
		// Build a JSON object from the sub-field values and unmarshal into
		// the target (Ptr returns **Record; json.Unmarshal handles nil
		// pointer construction).
		obj := make(map[string]string, len(subs))
		for _, s := range subs {
			if *s.strVal != "" {
				obj[s.fieldName] = *s.strVal
			}
		}
		jsonBytes, _ := json.Marshal(obj)
		_ = json.Unmarshal(jsonBytes, f.Ptr(cmd))
	}}
}

// applyBindings writes the parsed flag values into the command struct via the
// FieldDesc.Ptr accessors.
func applyBindings(cmd any, bindings []flagBinding) {
	for _, b := range bindings {
		b.apply(cmd)
	}
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
