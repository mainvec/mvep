package toolkit_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mainvec/mvep/toolkit"
)

// TestFieldDefTagsReachCodegen reproduces the pre-existing defect where
// `tags` on a field validates against the JSON schema but is dropped on
// unmarshal because FieldDef has no Tags field. Required-ness must reach
// codegen for CLI required-flag enforcement (plan 023, C4).
func TestFieldDefTagsReachCodegen(t *testing.T) {
	spec := `{
  "$schema": "https://spec.mainvec.com/mvepspec/0.2/schema/2026-01-15",
  "$id": "tagsTestId",
  "name": "tagsTestName",
  "namespace": "tagsTestNamespace",
  "commands": {
    "RegisterCmd": {
      "title": "Register something",
      "fields": {
        "name": {
          "fnum": 1,
          "type": "string",
          "tags": ["required"]
        },
        "nickname": {
          "fnum": 2,
          "type": "string"
        }
      }
    }
  }
}`

	srvDef, err := toolkit.BuildSrvDefFromJSON(strings.NewReader(spec))
	if err != nil {
		t.Fatalf("BuildSrvDefFromJSON failed: %v", err)
	}

	register, ok := srvDef.Commands.Get("RegisterCmd")
	if !ok {
		t.Fatal("RegisterCmd not found in parsed spec")
	}

	name, ok := register.Fields.Get("name")
	if !ok {
		t.Fatal("name field not found in parsed spec")
	}
	if !reflect.DeepEqual(name.Tags, []string{"required"}) {
		t.Fatalf("expected Tags [required], got %v", name.Tags)
	}

	nickname, ok := register.Fields.Get("nickname")
	if !ok {
		t.Fatal("nickname field not found in parsed spec")
	}
	if len(nickname.Tags) != 0 {
		t.Fatalf("expected no tags on nickname, got %v", nickname.Tags)
	}
}
