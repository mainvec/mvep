package toolkit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateEmitsDescriptor verifies T3: generation emits a mvep.PackageDesc
// literal into <name>_package.go, derives the Package methods via
// NewPackageFromDesc, and removes the three hand-shaped switch statements.
func TestGenerateEmitsDescriptor(t *testing.T) {
	outdir := t.TempDir()
	spec := filepath.Join("testdata", "05_command_withfields.jsonc")
	if err := ExecuteGenerate(context.Background(), spec, outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}

	pkgFile := filepath.Join(outdir, "api", "test5Name_package.go")
	src, err := os.ReadFile(pkgFile)
	if err != nil {
		t.Fatalf("read generated package: %v", err)
	}
	s := string(src)

	mustContain := []string{
		"mvep.PackageDesc{",
		"mvep.CommandDesc{",
		"mvep.FieldDesc{",
		"mvep.NewPackageFromDesc",
		"Ptr:",
		"SpecVersion:",
	}
	for _, want := range mustContain {
		if !strings.Contains(s, want) {
			t.Errorf("generated package missing %q", want)
		}
	}

	// The derived package is built once and shared (singleton), so the exported
	// delegators and NewPackage must use a package-level `pkg`, not rebuild via
	// NewPackageFromDesc on every call.
	if !strings.Contains(s, "var pkg = mvep.NewPackageFromDesc(&pkgDesc)") {
		t.Error("generated package should build a single shared pkg via NewPackageFromDesc")
	}
	if strings.Contains(s, "return mvep.NewPackageFromDesc(&pkgDesc)") {
		t.Error("NewPackage must return the shared pkg, not rebuild per call")
	}

	// The derived path must not emit the old switch bodies or the empty struct.
	// The exported InstanceOf / NameOf symbols are kept as one-line delegators
	// (plan 025 constraint 2), so only the switch statements are asserted gone.
	mustNotContain := []string{
		"switch compName",
		"switch comp.(type)",
	}
	for _, bad := range mustNotContain {
		if strings.Contains(s, bad) {
			t.Errorf("generated package still contains %q", bad)
		}
	}
}

// TestGenerateDescriptorIsByteStable verifies T3: the descriptor is emitted via
// ordered iterators, so repeated generation of the same spec is byte-stable.
func TestGenerateDescriptorIsByteStable(t *testing.T) {
	spec := filepath.Join("testdata", "05_command_withfields.jsonc")

	read := func() string {
		outdir := t.TempDir()
		if err := ExecuteGenerate(context.Background(), spec, outdir, "go", true, "plain"); err != nil {
			t.Fatalf("generate: %v", err)
		}
		b, err := os.ReadFile(filepath.Join(outdir, "api", "test5Name_package.go"))
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		return string(b)
	}

	first := read()
	for i := 0; i < 5; i++ {
		if got := read(); got != first {
			t.Fatalf("generation pass %d differs from first (non-deterministic output)", i+1)
		}
	}
}

// TestGenerateDescriptorFieldTypes verifies the descriptor carries correct
// FieldType enums, Required derivation, and ordering for a fields-heavy spec.
func TestGenerateDescriptorFieldTypes(t *testing.T) {
	outdir := t.TempDir()
	spec := filepath.Join("testdata", "05_command_withfields.jsonc")
	if err := ExecuteGenerate(context.Background(), spec, outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(outdir, "api", "test5Name_package.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(b)

	// int32, string, and repeated string fields must appear with their enums.
	for _, want := range []string{"mvep.FieldInt32", "mvep.FieldString", "Repeated: true"} {
		if !strings.Contains(s, want) {
			t.Errorf("generated descriptor missing %q", want)
		}
	}

	// Fields must be emitted in fnum order: size(1) before type(2) before toppings(3).
	iSize := strings.Index(s, `Name: "size"`)
	iType := strings.Index(s, `Name: "type"`)
	iToppings := strings.Index(s, `Name: "toppings"`)
	if iSize < 0 || iType < 0 || iToppings < 0 {
		t.Fatalf("missing field entries: size=%d type=%d toppings=%d", iSize, iType, iToppings)
	}
	if !(iSize < iType && iType < iToppings) {
		t.Errorf("fields not in fnum order: size=%d type=%d toppings=%d", iSize, iType, iToppings)
	}
}

// TestGenerateDescriptorFullTypeCoverage verifies T4: every spec field type
// (string, bool, int32, int64, uint32, sint32, float, double, bytes, timestamp,
// duration, uuid, repeated, map, recRef) is emitted with its correct FieldType
// enum. The fixture 11 carries one field per type, ordered by fnum.
func TestGenerateDescriptorFullTypeCoverage(t *testing.T) {
	outdir := t.TempDir()
	spec := filepath.Join("testdata", "11_descriptor_type_coverage.jsonc")
	if err := ExecuteGenerate(context.Background(), spec, outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(outdir, "api", "test11_package.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(b)

	// Each field type must emit its runtime FieldType enum, in fnum order.
	type wantField struct {
		name string
		enum string
	}
	wantFields := []wantField{
		{"str", "mvep.FieldString"},
		{"flag", "mvep.FieldBool"},
		{"i32", "mvep.FieldInt32"},
		{"i64", "mvep.FieldInt64"},
		{"u32", "mvep.FieldUint32"},
		{"s32", "mvep.FieldSint32"},
		{"f32", "mvep.FieldFloat"},
		{"f64", "mvep.FieldDouble"},
		{"byts", "mvep.FieldBytes"},
		{"ts", "mvep.FieldTimestamp"},
		{"dur", "mvep.FieldDuration"},
		{"uid", "mvep.FieldUUID"},
		{"repeateds", "mvep.FieldString"}, // repeated string stays FieldString + Repeated: true
		{"headers", "mvep.FieldMap"},
		{"addr", "mvep.FieldRecord"},
	}

	prev := -1
	for _, wf := range wantFields {
		iName := strings.Index(s, `Name: "`+wf.name+`"`)
		if iName < 0 {
			t.Errorf("missing field %q in generated descriptor", wf.name)
			continue
		}
		if iName < prev {
			t.Errorf("field %q at %d precedes previous field (not in fnum order)", wf.name, iName)
		}
		prev = iName

		// The enum must appear after the field's Name line.
		tail := s[iName:]
		if !strings.Contains(tail, wf.enum) {
			t.Errorf("field %q: enum %q not found after its Name line", wf.name, wf.enum)
		}
	}

	// Repeated must set Repeated: true on the repeateds field.
	if !strings.Contains(s, "Repeated: true") {
		t.Error("repeated field should emit Repeated: true")
	}

	// Required-ness is tag-derived: str and addr carry tags:[\"required\"].
	for _, want := range []string{
		`Tags: []string{"required"}`,
	} {
		if strings.Count(s, want) < 2 {
			t.Errorf("expected at least 2 required-tagged fields, found %d of %q", strings.Count(s, want), want)
		}
	}
}

// TestGenerateDescriptorRefIsNameOnlyResolvable verifies T4: a recRef field
// emits Ref as a name-only RecordDesc (Ref: &RecordDesc{Name: "Address"}) and
// the record's full Fields are emitted separately in PackageDesc.Records, so
// the runtime can resolve them via PackageDesc.Record(name) without codegen
// duplicating field data into the Ref (a drift hazard).
func TestGenerateDescriptorRefIsNameOnlyResolvable(t *testing.T) {
	outdir := t.TempDir()
	spec := filepath.Join("testdata", "06_command_with_ref.jsonc")
	if err := ExecuteGenerate(context.Background(), spec, outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(outdir, "api", "test6Name_package.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(b)

	// The addr field's Ref must be name-only: Name set, no inner Fields block
	// directly after it (the full fields live under Records). The emitted
	// shape is `Ref: &mvep.RecordDesc{Name: "Address"},` on one line.
	if !strings.Contains(s, `Ref:  &mvep.RecordDesc{Name: "Address"}`) &&
		!strings.Contains(s, `Ref: &mvep.RecordDesc{Name: "Address"}`) {
		t.Error("recRef field should emit Ref: &mvep.RecordDesc{Name: \"Address\"}")
	}

	// The Address record must be fully described under Records with its fields.
	if !strings.Contains(s, `Name: "Address"`) {
		t.Error("Address record name missing from Records")
	}
	for _, want := range []string{`Name: "street"`, `Name: "city"`, `Name: "country"`} {
		if !strings.Contains(s, want) {
			t.Errorf("Address record missing field %q", want)
		}
	}
}

// TestIsRequiredFieldAgreesWithFieldIsRequired verifies T4: the two near-
// identical required-ness helpers must agree. isRequiredField (registered for
// JS/TS templates) previously hardcoded false while fieldIsRequired (Go) was
// tag-derived, so Go and JS output disagreed silently. Both are now tag-driven.
func TestIsRequiredFieldAgreesWithFieldIsRequired(t *testing.T) {
	cases := []struct {
		name string
		tags []string
		want bool
	}{
		{"no tags", nil, false},
		{"unrelated tag", []string{"deprecated"}, false},
		{"required", []string{"required"}, true},
		{"required among others", []string{"deprecated", "required"}, true},
	}
	for _, tc := range cases {
		f := FieldDef{Tags: tc.tags}
		if got := fieldIsRequired(f); got != tc.want {
			t.Errorf("fieldIsRequired(%s) = %v, want %v", tc.name, got, tc.want)
		}
		if got := isRequiredField(f); got != tc.want {
			t.Errorf("isRequiredField(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestExecuteGenerateUnsupportedConstructReturnsError verifies T5: a spec using
// a construct the descriptor cannot represent (here, an inline "record" field
// type, which the runtime descriptor has no FieldType for) fails at
// ExecuteGenerate with a returned error naming the offending command and
// field — rather than panicking deep in template execution.
func TestExecuteGenerateUnsupportedConstructReturnsError(t *testing.T) {
	outdir := t.TempDir()
	err := ExecuteGenerate(context.Background(), filepath.Join("testdata", "12_unsupported_construct.jsonc"), outdir, "go", true, "plain")
	if err == nil {
		t.Fatal("expected an error for an unsupported construct, got nil")
	}
	msg := err.Error()
	// The error must identify the offending command and field, not just "unsupported".
	if !strings.Contains(msg, "BadCmd") {
		t.Errorf("error should name the offending command; got: %s", msg)
	}
	if !strings.Contains(msg, "inlineRec") {
		t.Errorf("error should name the offending field; got: %s", msg)
	}
}

// TestDescriptorOutputLeaksNoGenOptionsPaths verifies #30: generated descriptor
// output for a spec whose gen_options set go_package and go_api_package must
// contain neither value. This guards the stated rationale for excluding
// GenOpts from PackageDesc: those are internal module and filesystem paths,
// and any discovery endpoint serialising the descriptor would publish
// repository layout to every client. That is a security argument, so it
// deserves a regression test rather than a one-time manual check.
func TestDescriptorOutputLeaksNoGenOptionsPaths(t *testing.T) {
	outdir := t.TempDir()
	// Fixture 06 sets go_package in gen_options.
	spec := filepath.Join("testdata", "06_command_with_ref.jsonc")
	if err := ExecuteGenerate(context.Background(), spec, outdir, "go", true, "plain"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(outdir, "api", "test6Name_package.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(b)

	// The go_package value from fixture 06's gen_options.
	const goPackage = "github.com/mainvec/wo/mvep/wopdb/wopdb2api"
	if strings.Contains(s, goPackage) {
		t.Errorf("descriptor output leaks go_package path: %q appears in generated code", goPackage)
	}

	// Also verify the literal gen_options key names don't appear in the
	// descriptor (they could appear in comments, but the descriptor literal
	// should not reference them).
	for _, key := range []string{"go_package", "go_api_package"} {
		if strings.Contains(s, `"`+key+`"`) {
			t.Errorf("descriptor output references gen_options key %q", key)
		}
	}
}
