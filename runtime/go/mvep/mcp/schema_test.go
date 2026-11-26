package mcp

import (
	"strings"
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

func TestSchemasFromPackage_AllFieldTypesAndNesting(t *testing.T) {
	desc := &mvep.PackageDesc{
		Name:        "demo",
		SpecVersion: "v1.0",
		Records: []mvep.RecordDesc{
			{
				Name: "Address",
				Fields: []mvep.FieldDesc{
					{Name: "street", Type: mvep.FieldString, Required: true, Desc: "Street line"},
					{Name: "zip", Type: mvep.FieldInt32},
				},
			},
			{
				Name: "TreeNode",
				Fields: []mvep.FieldDesc{
					{Name: "label", Type: mvep.FieldString, Required: true},
					{Name: "child", Type: mvep.FieldRecord, Ref: &mvep.RecordDesc{Name: "TreeNode"}},
				},
			},
		},
		Commands: []mvep.CommandDesc{
			{
				Name:  "AllTypesCmd",
				Alias: "all_types",
				Group: "admin/tools",
				Desc:  "Exercises every field type",
				Fields: []mvep.FieldDesc{
					{Name: "str", Type: mvep.FieldString, Required: true, Desc: "A string"},
					{Name: "flag", Type: mvep.FieldBool},
					{Name: "i32", Type: mvep.FieldInt32},
					{Name: "i64", Type: mvep.FieldInt64},
					{Name: "u32", Type: mvep.FieldUint32},
					{Name: "s32", Type: mvep.FieldSint32},
					{Name: "f32", Type: mvep.FieldFloat},
					{Name: "f64", Type: mvep.FieldDouble},
					{Name: "raw", Type: mvep.FieldBytes},
					{Name: "at", Type: mvep.FieldTimestamp},
					{Name: "ttl", Type: mvep.FieldDuration, Desc: "Timeout"},
					{Name: "id", Type: mvep.FieldUUID},
					{Name: "meta", Type: mvep.FieldMap},
					{Name: "addr", Type: mvep.FieldRecord, Ref: &mvep.RecordDesc{Name: "Address"}},
					{Name: "tags", Type: mvep.FieldString, Repeated: true, Desc: "List of tags"},
					{Name: "tree", Type: mvep.FieldRecord, Ref: &mvep.RecordDesc{Name: "TreeNode"}},
				},
			},
		},
	}

	tools, err := SchemasFromPackage(desc)
	if err != nil {
		t.Fatalf("SchemasFromPackage: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name != "admin_tools_all_types" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "admin_tools_all_types")
	}
	if tool.Description != "Exercises every field type" {
		t.Errorf("tool.Description = %q", tool.Description)
	}

	props := tool.InputSchema["properties"].(map[string]any)
	req := tool.InputSchema["required"].([]string)
	if len(req) != 1 || req[0] != "str" {
		t.Errorf("required = %v, want [str]", req)
	}

	wantTypes := map[string]string{
		"str":  "string",
		"flag": "boolean",
		"i32":  "integer",
		"i64":  "integer",
		"u32":  "integer",
		"s32":  "integer",
		"f32":  "number",
		"f64":  "number",
		"raw":  "string",
		"at":   "string",
		"ttl":  "integer",
		"id":   "string",
		"meta": "object",
		"addr": "object",
		"tags": "array",
		"tree": "object",
	}
	for field, wantType := range wantTypes {
		p, ok := props[field].(map[string]any)
		if !ok {
			t.Fatalf("missing property %q", field)
		}
		if p["type"] != wantType {
			t.Errorf("field %q type = %v, want %q", field, p["type"], wantType)
		}
	}

	// Format hints and constraints.
	if props["u32"].(map[string]any)["minimum"] != 0 {
		t.Errorf("u32 should have minimum=0, got %v", props["u32"])
	}
	if props["raw"].(map[string]any)["format"] != "byte" {
		t.Errorf("raw format = %v, want byte", props["raw"])
	}
	if props["at"].(map[string]any)["format"] != "date-time" {
		t.Errorf("at format = %v, want date-time", props["at"])
	}
	if props["id"].(map[string]any)["format"] != "uuid" {
		t.Errorf("id format = %v, want uuid", props["id"])
	}
	if !strings.Contains(props["ttl"].(map[string]any)["description"].(string), "nanoseconds") {
		t.Errorf("ttl description should mention nanoseconds: %v", props["ttl"])
	}

	// Map stays permissive (no additionalProperties: false).
	if _, hasAP := props["meta"].(map[string]any)["additionalProperties"]; hasAP {
		t.Errorf("map field should remain permissive, got %v", props["meta"])
	}

	// Nested record resolves fields and required from PackageDesc.Records.
	addrProps := props["addr"].(map[string]any)["properties"].(map[string]any)
	if addrProps["street"].(map[string]any)["type"] != "string" {
		t.Errorf("addr.street type = %v", addrProps["street"])
	}
	addrReq := props["addr"].(map[string]any)["required"].([]string)
	if len(addrReq) != 1 || addrReq[0] != "street" {
		t.Errorf("addr required = %v, want [street]", addrReq)
	}

	// Repeated scalar wraps items.
	tagsSchema := props["tags"].(map[string]any)
	if tagsSchema["items"].(map[string]any)["type"] != "string" {
		t.Errorf("tags items = %v", tagsSchema["items"])
	}

	// Self-referential record terminates cleanly via cycle guard.
	treeProps := props["tree"].(map[string]any)["properties"].(map[string]any)
	childSchema := treeProps["child"].(map[string]any)
	if childSchema["type"] != "object" {
		t.Errorf("recursive child type = %v, want object", childSchema["type"])
	}
	if _, expanded := childSchema["properties"]; expanded {
		t.Errorf("recursive child should terminate without infinite expansion")
	}
}

func TestSchemasFromPackage_ToolNamingAndCollisions(t *testing.T) {
	desc := &mvep.PackageDesc{
		Name: "svc",
		Commands: []mvep.CommandDesc{
			{Name: "StartServerCmd", Alias: "start", Group: "server"},
			{Name: "StopServerCmd", Group: "server"},
			{Name: "PingCmd"},
		},
	}

	tools, err := SchemasFromPackage(desc)
	if err != nil {
		t.Fatalf("default namer: %v", err)
	}
	want := []string{"server_start", "server_stop_server_cmd", "ping_cmd"}
	for i, w := range want {
		if tools[i].Name != w {
			t.Errorf("tools[%d].Name = %q, want %q", i, tools[i].Name, w)
		}
	}

	// DescriptorNameToolNamer exposes canonical Name verbatim.
	descTools, err := SchemasFromPackage(desc, WithToolNamer(DescriptorNameToolNamer))
	if err != nil {
		t.Fatalf("descriptor namer: %v", err)
	}
	if descTools[0].Name != "StartServerCmd" {
		t.Errorf("descTools[0].Name = %q, want StartServerCmd", descTools[0].Name)
	}

	// Collision after group flattening must fail loudly.
	colliding := &mvep.PackageDesc{
		Name: "svc",
		Commands: []mvep.CommandDesc{
			{Name: "FirstCmd", Alias: "start", Group: "server"},
			{Name: "SecondCmd", Alias: "server_start"},
		},
	}
	_, err = SchemasFromPackage(colliding)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	if !strings.Contains(err.Error(), "server_start") || !strings.Contains(err.Error(), "FirstCmd") || !strings.Contains(err.Error(), "SecondCmd") {
		t.Errorf("collision error should name tool and both commands, got: %v", err)
	}
}

func TestSchemasFromPackage_Errors(t *testing.T) {
	if _, err := SchemasFromPackage(nil); err == nil {
		t.Error("expected error for nil descriptor")
	}

	badType := &mvep.PackageDesc{
		Name: "svc",
		Commands: []mvep.CommandDesc{
			{
				Name: "BadCmd",
				Fields: []mvep.FieldDesc{
					{Name: "mystery", Type: mvep.FieldType(999)},
				},
			},
		},
	}
	_, err := SchemasFromPackage(badType)
	if err == nil {
		t.Fatal("expected error for unsupported FieldType, got nil")
	}
	if !strings.Contains(err.Error(), "BadCmd") || !strings.Contains(err.Error(), "mystery") {
		t.Errorf("error should name command and field, got: %v", err)
	}
}
