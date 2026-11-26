package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// SchemasFromPackage projects a PackageDesc into deterministic MCP Tool
// definitions. It returns an error if desc is nil, if any command produces an
// empty or colliding tool name, or if any field uses an unmapped FieldType.
func SchemasFromPackage(desc *mvep.PackageDesc, opts ...Option) ([]Tool, error) {
	entries, err := buildToolIndex(desc, opts...)
	if err != nil {
		return nil, err
	}
	tools := make([]Tool, len(entries))
	for i, e := range entries {
		tools[i] = e.tool
	}
	return tools, nil
}

type toolEntry struct {
	tool Tool
	cmd  *mvep.CommandDesc
}

func buildToolIndex(desc *mvep.PackageDesc, opts ...Option) ([]toolEntry, error) {
	if desc == nil {
		return nil, errors.New("mcp: package descriptor is nil")
	}
	cfg := defaultConfig(desc)
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	entries := make([]toolEntry, 0, len(desc.Commands))
	seen := make(map[string]string, len(desc.Commands))

	for i := range desc.Commands {
		cmd := &desc.Commands[i]
		name := cfg.namer(cmd)
		if name == "" {
			return nil, fmt.Errorf("mcp: command %q produced an empty tool name", cmd.Name)
		}
		if prev, dup := seen[name]; dup {
			return nil, fmt.Errorf("mcp: duplicate tool name %q for commands %q and %q", name, prev, cmd.Name)
		}
		seen[name] = cmd.Name

		schema, err := buildObjectSchema(cmd.Fields, desc, map[string]bool{})
		if err != nil {
			return nil, fmt.Errorf("mcp: command %q: %w", cmd.Name, err)
		}

		entries = append(entries, toolEntry{
			tool: Tool{
				Name:        name,
				Description: cmd.Desc,
				InputSchema: schema,
			},
			cmd: cmd,
		})
	}
	return entries, nil
}

func buildObjectSchema(fields []mvep.FieldDesc, desc *mvep.PackageDesc, visited map[string]bool) (map[string]any, error) {
	props := make(map[string]any, len(fields))
	var required []string

	for i := range fields {
		f := &fields[i]
		propSchema, err := buildFieldSchema(f, desc, visited)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", f.Name, err)
		}
		props[f.Name] = propSchema
		if f.Required {
			required = append(required, f.Name)
		}
	}

	schema := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema, nil
}

func buildFieldSchema(f *mvep.FieldDesc, desc *mvep.PackageDesc, visited map[string]bool) (map[string]any, error) {
	elem, err := buildScalarOrCompositeSchema(f, desc, visited)
	if err != nil {
		return nil, err
	}

	if f.Repeated {
		arr := map[string]any{
			"type":  "array",
			"items": elem,
		}
		if f.Desc != "" {
			arr["description"] = f.Desc
		}
		return arr, nil
	}

	if f.Desc != "" {
		if existing, ok := elem["description"].(string); ok && existing != "" {
			elem["description"] = f.Desc + " (" + existing + ")"
		} else {
			elem["description"] = f.Desc
		}
	}
	return elem, nil
}

func buildScalarOrCompositeSchema(f *mvep.FieldDesc, desc *mvep.PackageDesc, visited map[string]bool) (map[string]any, error) {
	switch f.Type {
	case mvep.FieldString:
		return map[string]any{"type": "string"}, nil
	case mvep.FieldBool:
		return map[string]any{"type": "boolean"}, nil
	case mvep.FieldInt32, mvep.FieldInt64, mvep.FieldSint32:
		return map[string]any{"type": "integer"}, nil
	case mvep.FieldUint32:
		return map[string]any{"type": "integer", "minimum": 0}, nil
	case mvep.FieldFloat, mvep.FieldDouble:
		return map[string]any{"type": "number"}, nil
	case mvep.FieldBytes:
		return map[string]any{"type": "string", "format": "byte"}, nil
	case mvep.FieldTimestamp:
		return map[string]any{"type": "string", "format": "date-time"}, nil
	case mvep.FieldDuration:
		// Plain Go structs encode/decode time.Duration via encoding/json as
		// int64 nanoseconds. Match the actual wire decoder so LLM calls decode cleanly.
		return map[string]any{"type": "integer", "description": "duration in nanoseconds"}, nil
	case mvep.FieldUUID:
		return map[string]any{"type": "string", "format": "uuid"}, nil
	case mvep.FieldMap:
		// FieldDesc does not carry map value types, so maps stay permissive.
		return map[string]any{"type": "object"}, nil
	case mvep.FieldRecord:
		return buildRecordSchema(f, desc, visited)
	default:
		return nil, fmt.Errorf("unsupported field type %d", f.Type)
	}
}

func buildRecordSchema(f *mvep.FieldDesc, desc *mvep.PackageDesc, visited map[string]bool) (map[string]any, error) {
	if f.Ref == nil {
		return map[string]any{"type": "object"}, nil
	}

	recName := f.Ref.Name
	if recName != "" && visited[recName] {
		// Cycle guard for self-referential records: terminate with an open object.
		return map[string]any{"type": "object"}, nil
	}

	var fields []mvep.FieldDesc
	if f.Ref.Fields != nil {
		fields = f.Ref.Fields
	} else if recName != "" {
		if rec, ok := desc.Record(recName); ok {
			fields = rec.Fields
		}
	}
	if fields == nil {
		return map[string]any{"type": "object"}, nil
	}

	nextVisited := make(map[string]bool, len(visited)+1)
	for k, v := range visited {
		nextVisited[k] = v
	}
	if recName != "" {
		nextVisited[recName] = true
	}
	return buildObjectSchema(fields, desc, nextVisited)
}

// validateArgs enforces strict key and required-field checks against the
// command descriptor before dispatching to PackageHandler.ServeCmdReq, because
// encoding/json silently ignores unknown fields and allows omitted required
// fields.
func validateArgs(cmd *mvep.CommandDesc, desc *mvep.PackageDesc, payload []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(payload, &obj); err != nil {
		return fmt.Errorf("arguments must be a JSON object: %w", err)
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	return validateFields(obj, cmd.Fields, desc, "")
}

func validateFields(obj map[string]json.RawMessage, fields []mvep.FieldDesc, desc *mvep.PackageDesc, path string) error {
	known := make(map[string]*mvep.FieldDesc, len(fields))
	for i := range fields {
		known[normalizeKey(fields[i].Name)] = &fields[i]
	}

	present := make(map[string]bool, len(obj))
	for key, raw := range obj {
		nk := normalizeKey(key)
		f, ok := known[nk]
		if !ok {
			if path != "" {
				return fmt.Errorf("unknown argument %q in %s", key, strings.TrimSuffix(path, "."))
			}
			return fmt.Errorf("unknown argument %q", key)
		}
		if len(raw) > 0 && string(raw) != "null" {
			present[nk] = true
		}
		if f.Type == mvep.FieldRecord && f.Ref != nil && string(raw) != "null" {
			if err := validateRecordValue(raw, f, desc, path+key); err != nil {
				return err
			}
		}
	}

	for i := range fields {
		f := &fields[i]
		if f.Required && !present[normalizeKey(f.Name)] {
			if path != "" {
				return fmt.Errorf("missing required argument %q in %s", f.Name, strings.TrimSuffix(path, "."))
			}
			return fmt.Errorf("missing required argument %q", f.Name)
		}
	}
	return nil
}

func validateRecordValue(raw json.RawMessage, f *mvep.FieldDesc, desc *mvep.PackageDesc, path string) error {
	var refFields []mvep.FieldDesc
	if f.Ref.Fields != nil {
		refFields = f.Ref.Fields
	} else if rec, ok := desc.Record(f.Ref.Name); ok {
		refFields = rec.Fields
	}
	if len(refFields) == 0 {
		return nil
	}

	if f.Repeated {
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil // let the decoder produce a type mismatch error
		}
		for _, elem := range arr {
			if err := validateRecordObject(elem, refFields, desc, path); err != nil {
				return err
			}
		}
		return nil
	}
	return validateRecordObject(raw, refFields, desc, path)
}

func validateRecordObject(raw json.RawMessage, refFields []mvep.FieldDesc, desc *mvep.PackageDesc, path string) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	return validateFields(obj, refFields, desc, path+".")
}

// normalizeKey lower-cases and strips underscores so snake_case and camelCase
// keys compare equal, matching mvep/cli's payload validation.
func normalizeKey(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "_", "")
}
