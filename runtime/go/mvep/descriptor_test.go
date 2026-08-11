package mvep_test

import (
	"testing"

	"github.com/mainvec/mvep/runtime/go/mvep"
)

// testEchoCmd is a hand-written command standing in for a generated struct.
type testEchoCmd struct {
	In    string
	Count int32
	Tags  []string
}

// testEchoCmdResult is the matching result struct.
type testEchoCmdResult struct {
	Out string
}

// testRecord is a nested record referenced by a command field.
type testRecord struct {
	Host string
	Port uint32
}

// testPingCmd is a nameable command type with no fields.
type testPingCmd struct{}

// testPkgDesc is a hand-written descriptor exercising every descriptor type.
var testPkgDesc = mvep.PackageDesc{
	Name:        "testPackage",
	Namespace:   "test",
	Title:       "Test Package",
	Desc:        "A test package",
	Base:        "/test",
	SpecVersion: "0.2",
	Commands: []mvep.CommandDesc{
		{
			Name:  "EchoCmd",
			Alias: "echo",
			Desc:  "Echo a string",
			New:   func() any { return &testEchoCmd{} },
			Fields: []mvep.FieldDesc{
				{
					Name: "in", Fnum: 1, Type: mvep.FieldString, Required: true,
					Desc: "input string",
					Ptr:  func(c any) any { return &c.(*testEchoCmd).In },
				},
				{
					Name: "count", Fnum: 2, Type: mvep.FieldInt32,
					Desc: "repetition count",
					Ptr:  func(c any) any { return &c.(*testEchoCmd).Count },
				},
				{
					Name: "tags", Fnum: 3, Type: mvep.FieldString, Repeated: true,
					Desc: "tags",
					Ptr:  func(c any) any { return &c.(*testEchoCmd).Tags },
				},
			},
			Result: &mvep.ResultDesc{
				Name: "EchoCmdResult",
				New:  func() any { return &testEchoCmdResult{} },
				Fields: []mvep.FieldDesc{
					{
						Name: "out", Fnum: 1, Type: mvep.FieldString,
						Ptr: func(r any) any { return &r.(*testEchoCmdResult).Out },
					},
				},
			},
		},
		{
			Name: "PingCmd",
			New:  func() any { return &testPingCmd{} },
		},
	},
	Records: []mvep.RecordDesc{
		{
			Name: "Endpoint",
			Fields: []mvep.FieldDesc{
				{Name: "host", Fnum: 1, Type: mvep.FieldString, Ptr: func(r any) any { return &r.(*testRecord).Host }},
				{Name: "port", Fnum: 2, Type: mvep.FieldUint32, Ptr: func(r any) any { return &r.(*testRecord).Port }},
			},
		},
	},
}

// TestPackageDescIteratesInStableOrder verifies the descriptor's ordered slices
// iterate in declaration order across repeated passes (T1).
func TestPackageDescIteratesInStableOrder(t *testing.T) {
	t.Parallel()

	wantCmds := []string{"EchoCmd", "PingCmd"}
	wantFields := []string{"in", "count", "tags"}

	for pass := 0; pass < 50; pass++ {
		var gotCmds []string
		for _, c := range testPkgDesc.Commands {
			gotCmds = append(gotCmds, c.Name)
		}
		if len(gotCmds) != len(wantCmds) {
			t.Fatalf("pass %d: got %d commands, want %d", pass, len(gotCmds), len(wantCmds))
		}
		for i := range wantCmds {
			if gotCmds[i] != wantCmds[i] {
				t.Fatalf("pass %d: command[%d] = %q, want %q", pass, i, gotCmds[i], wantCmds[i])
			}
		}

		var gotFields []string
		for _, f := range testPkgDesc.Commands[0].Fields {
			gotFields = append(gotFields, f.Name)
		}
		for i := range wantFields {
			if gotFields[i] != wantFields[i] {
				t.Fatalf("pass %d: field[%d] = %q, want %q", pass, i, gotFields[i], wantFields[i])
			}
		}
	}
}

// TestFieldTypeCoversSpecTypes asserts the FieldType enum mirrors every type the
// spec can express, so a codegen type-switch can be total (T1).
func TestFieldTypeCoversSpecTypes(t *testing.T) {
	t.Parallel()

	types := map[mvep.FieldType]bool{
		mvep.FieldString:    true,
		mvep.FieldBool:      true,
		mvep.FieldInt32:     true,
		mvep.FieldInt64:     true,
		mvep.FieldUint32:    true,
		mvep.FieldSint32:    true,
		mvep.FieldFloat:     true,
		mvep.FieldDouble:    true,
		mvep.FieldBytes:     true,
		mvep.FieldTimestamp: true,
		mvep.FieldDuration:  true,
		mvep.FieldUUID:      true,
		mvep.FieldMap:       true,
		mvep.FieldRecord:    true,
	}
	if len(types) != 14 {
		t.Errorf("expected 14 spec field types, got %d", len(types))
	}
}

// TestPackageDescriberIsOptional asserts PackageDescriber is a standalone,
// satisfiable interface (T1). Derivation of Package from a descriptor is T2
// and is tested separately.
func TestPackageDescriberIsOptional(t *testing.T) {
	t.Parallel()

	var d mvep.PackageDescriber = staticDescriber{desc: &testPkgDesc}
	if got := d.Describe(); got != &testPkgDesc {
		t.Errorf("Describe() = %p, want %p", got, &testPkgDesc)
	}
}

// staticDescriber is a minimal PackageDescriber implementation.
type staticDescriber struct{ desc *mvep.PackageDesc }

func (s staticDescriber) Describe() *mvep.PackageDesc { return s.desc }

// --- T2: derivation of InstanceOf / NameOf / CommandNames -------------------

// TestNewPackageFromDescDerivesPackage verifies the derived Package matches the
// behaviour of today's generated switch-based methods (T2).
func TestNewPackageFromDescDerivesPackage(t *testing.T) {
	t.Parallel()

	pkg := mvep.NewPackageFromDesc(&testPkgDesc)

	// GetName reapplies the legacy "Package" suffix (spec name + "Package"),
	// because it feeds HTTP routing and must not move routes.
	if got := pkg.GetName(); got != "testPackagePackage" {
		t.Errorf("GetName() = %q, want %q", got, "testPackagePackage")
	}

	// InstanceOf constructs commands AND results by struct name.
	for _, name := range []string{"EchoCmd", "EchoCmdResult", "PingCmd"} {
		v, ok := pkg.InstanceOf(name)
		if !ok {
			t.Errorf("InstanceOf(%q) = not found", name)
			continue
		}
		if v == nil {
			t.Errorf("InstanceOf(%q) = nil", name)
		}
	}
	if _, ok := pkg.InstanceOf("NopeCmd"); ok {
		t.Error("InstanceOf(NopeCmd) should be not found")
	}

	// NameOf is the inverse, keyed on concrete pointer type, O(1).
	for name, comp := range map[string]any{
		"EchoCmd":       &testEchoCmd{},
		"EchoCmdResult": &testEchoCmdResult{},
		"PingCmd":       &testPingCmd{},
	} {
		if got := pkg.NameOf(comp); got != name {
			t.Errorf("NameOf(%T) = %q, want %q", comp, got, name)
		}
	}
	if got := pkg.NameOf(&struct{}{}); got != "" {
		t.Errorf("NameOf(unknown) = %q, want empty", got)
	}
}

// TestNewPackageFromDescSatisfiesOptionalInterfaces verifies the derived package
// implements CommandLister and PackageDescriber (T2).
func TestNewPackageFromDescSatisfiesOptionalInterfaces(t *testing.T) {
	t.Parallel()

	pkg := mvep.NewPackageFromDesc(&testPkgDesc)

	cl, ok := pkg.(mvep.CommandLister)
	if !ok {
		t.Fatal("derived package should implement CommandLister")
	}
	names := cl.CommandNames()
	want := []string{"EchoCmd", "PingCmd"}
	if len(names) != len(want) {
		t.Fatalf("CommandNames() = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("CommandNames()[%d] = %q, want %q", i, names[i], want[i])
		}
	}

	pd, ok := pkg.(mvep.PackageDescriber)
	if !ok {
		t.Fatal("derived package should implement PackageDescriber")
	}
	if pd.Describe() != &testPkgDesc {
		t.Error("Describe() should return the descriptor passed to NewPackageFromDesc")
	}
}

// --- T4: record resolution -------------------------------------------------

// TestPackageDescRecordResolvesByName verifies that a FieldDesc.Ref carrying
// only the record name can be resolved to the full RecordDesc in
// PackageDesc.Records, so T9's flag flattening can reach the record's fields
// without codegen duplicating them into the Ref (T4).
func TestPackageDescRecordResolvesByName(t *testing.T) {
	t.Parallel()

	rec, ok := testPkgDesc.Record("Endpoint")
	if !ok {
		t.Fatal("Record(Endpoint) = not found")
	}
	if rec.Name != "Endpoint" {
		t.Errorf("Record(Endpoint).Name = %q, want %q", rec.Name, "Endpoint")
	}
	if len(rec.Fields) != 2 {
		t.Fatalf("Record(Endpoint).Fields = %d, want 2", len(rec.Fields))
	}
	if rec.Fields[0].Name != "host" || rec.Fields[1].Name != "port" {
		t.Errorf("Record(Endpoint).Fields = %v, want host/port", rec.Fields)
	}

	if _, ok := testPkgDesc.Record("Missing"); ok {
		t.Error("Record(Missing) should be not found")
	}
}

// --- Plan 040 T3: command groups -------------------------------------------

// groupedCmd is a command struct placed under a group.
type groupedCmd struct {
	Model string
}

// testGroupedPkgDesc is a descriptor exercising the group surface (T3): a
// nested group at depth 2, an alias, and a hidden group, plus a root command.
var testGroupedPkgDesc = mvep.PackageDesc{
	Name: "testgrouped",
	Commands: []mvep.CommandDesc{
		{
			Name:  "StartServerCmd",
			Alias: "start",
			Group: "server",
			Desc:  "Start an LLM server",
			New:   func() any { return &groupedCmd{} },
			Fields: []mvep.FieldDesc{
				{Name: "model", Fnum: 1, Type: mvep.FieldString, Ptr: func(c any) any { return &c.(*groupedCmd).Model }},
			},
		},
		{
			Name: "RootCmd",
			New:  func() any { return &testPingCmd{} },
		},
	},
	Groups: []mvep.GroupDesc{
		{Path: "server", Name: "server", Title: "LLM Servers"},
		{Path: "server/keys", Name: "keys", Title: "API Keys", Aliases: []string{"key"}},
		{Path: "hidden", Name: "hidden", Hidden: true},
	},
}

// TestCommandGroupSurface verifies the descriptor carries group metadata and
// that commands resolve their group paths, with no change to root commands
// (T3). Additive fields must not disturb NewPackageFromDesc.
func TestCommandGroupSurface(t *testing.T) {
	t.Parallel()

	pkg := mvep.NewPackageFromDesc(&testGroupedPkgDesc)

	cl, ok := pkg.(mvep.CommandLister)
	if !ok {
		t.Fatal("derived package should implement CommandLister")
	}
	if len(cl.CommandNames()) != 2 {
		t.Fatalf("CommandNames() = %v, want 2 commands", cl.CommandNames())
	}

	// Root command unaffected by group presence.
	if _, ok := pkg.InstanceOf("RootCmd"); !ok {
		t.Error("InstanceOf(RootCmd) should resolve")
	}

	// The grouped command still resolves by its descriptor Name.
	if _, ok := pkg.InstanceOf("StartServerCmd"); !ok {
		t.Error("InstanceOf(StartServerCmd) should resolve")
	}
}

// TestGroupDescFlatListOrder verifies Groups is a flat, ordered slice (T3): the
// full path is carried, the Name is the final segment, and ordering is the
// declaration order used by cli.New to build the tree.
func TestGroupDescFlatListOrder(t *testing.T) {
	t.Parallel()

	want := []struct{ path, name string }{
		{"server", "server"},
		{"server/keys", "keys"},
		{"hidden", "hidden"},
	}
	if len(testGroupedPkgDesc.Groups) != len(want) {
		t.Fatalf("Groups = %d, want %d", len(testGroupedPkgDesc.Groups), len(want))
	}
	for i, w := range want {
		if testGroupedPkgDesc.Groups[i].Path != w.path {
			t.Errorf("Groups[%d].Path = %q, want %q", i, testGroupedPkgDesc.Groups[i].Path, w.path)
		}
		if testGroupedPkgDesc.Groups[i].Name != w.name {
			t.Errorf("Groups[%d].Name = %q, want %q", i, testGroupedPkgDesc.Groups[i].Name, w.name)
		}
	}
}
