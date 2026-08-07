package mvep

import "reflect"

// EXPERIMENTAL: the descriptor types in this file are core public API from day
// one but their shape may change for one release cycle. The marker is removed
// once the CLI builder (T16 of plan 025) has dogfooded the design.

// PackageDesc is a complete runtime description of a generated package:
// its commands, fields, results, types, tags, and ordering. Codegen emits one
// as a Go literal into the generated mvep_package.go, from the same run that
// produces the command structs, so the two can never disagree.
//
// It describes what a package *is* at runtime, not how it was *built*:
// build-time inputs (GenOpts, ProtocOpts) are deliberately excluded.
//
// All collections are ordered slices, never maps, so help output and iteration
// order are deterministic by construction.
type PackageDesc struct {
	Name        string
	Namespace   string
	Title       string
	Desc        string
	Base        string
	SpecVersion string // informational only (e.g. --version output)
	Commands    []CommandDesc
	Records     []RecordDesc
}

// CommandDesc describes a single command: how to construct it, its fields in
// declaration order, and its result.
type CommandDesc struct {
	Name   string
	Alias  string
	Desc   string
	New    func() any // constructs a new command struct
	Fields []FieldDesc
	Result *ResultDesc
}

// ResultDesc describes a command's result struct. It is not optional garnish:
// InstanceOf returns both command types and *XxxCmdResult types, so the
// descriptor cannot back it without describing results.
type ResultDesc struct {
	Name   string
	New    func() any
	Fields []FieldDesc
}

// FieldDesc describes a single field. Ptr returns a typed pointer to the field
// inside the owning struct (*string, *int64, *[]string and so on); consumers
// type-switch on it. Because Ptr closes over a real struct field, a codegen
// mistake is a compile error, not a silent runtime drop. The same accessor
// serves binding (cli), validation (zero-value test) and redaction (scrub).
type FieldDesc struct {
	Name     string
	Alias    string
	Desc     string
	Fnum     int32
	Type     FieldType
	Repeated bool
	Required bool
	Tags     []string
	Ptr      func(any) any
	Ref      *RecordDesc // set when Type == FieldRecord
}

// RecordDesc describes a nested record referenced by a $ref field.
type RecordDesc struct {
	Name   string
	Fields []FieldDesc
}

// FieldType is a runtime-owned enum mirroring the spec's field types. It is
// deliberately not toolkit.FieldDataType: core mvep depends on nothing, and
// the toolkit keeps full ownership of the spec model.
type FieldType int

const (
	FieldString FieldType = iota
	FieldBool
	FieldInt32
	FieldInt64
	FieldUint32
	FieldSint32
	FieldFloat
	FieldDouble
	FieldBytes
	FieldTimestamp
	FieldDuration
	FieldUUID
	FieldMap
	FieldRecord
)

// PackageDescriber is an optional interface a Package can implement to expose
// its descriptor, mirroring the existing CommandLister idiom.
type PackageDescriber interface {
	Describe() *PackageDesc
}

// Record looks up a record descriptor by name. Codegen emits FieldDesc.Ref
// carrying only the record name (Ref: &RecordDesc{Name: "Address"}); this
// method resolves it to the full RecordDesc in Records so consumers — flag
// flattening in mvep/cli (T9), validation, redaction — can reach the record's
// fields without codegen duplicating them into the Ref (a drift hazard).
// Returns ok=false if no record with that name is described.
func (d *PackageDesc) Record(name string) (*RecordDesc, bool) {
	for i := range d.Records {
		if d.Records[i].Name == name {
			return &d.Records[i], true
		}
	}
	return nil, false
}

// NewPackageFromDesc builds a Package from a descriptor. The returned value
// also satisfies CommandLister and PackageDescriber.
//
// It is additive, never mandatory: Package stays implementable by hand. The
// derivation logic lives here once, in the runtime, instead of being
// re-emitted per generated package.
//
// GetName returns desc.Name + "Package" to preserve today's generated
// behaviour (spec name "mvep" -> "mvepPackage"), which feeds HTTP routing;
// returning the bare name would move every route and 404 existing clients.
// NameOf uses a one-time map[reflect.Type]string built from the New()
// closures: a single reflect.TypeOf per type at construction, O(1) per call.
func NewPackageFromDesc(desc *PackageDesc) Package {
	p := &descPackage{desc: desc}

	p.byName = make(map[string]func() any, len(desc.Commands)*2)
	p.nameByType = make(map[reflect.Type]string, len(desc.Commands)*2)
	p.cmdNames = make([]string, 0, len(desc.Commands))

	register := func(name string, newFn func() any) {
		if newFn == nil {
			return
		}
		p.byName[name] = newFn
		p.nameByType[reflect.TypeOf(newFn())] = name
	}
	for _, c := range desc.Commands {
		register(c.Name, c.New)
		p.cmdNames = append(p.cmdNames, c.Name)
		if c.Result != nil {
			register(c.Result.Name, c.Result.New)
		}
	}
	return p
}

// descPackage is the Package implementation derived from a PackageDesc.
type descPackage struct {
	desc *PackageDesc

	byName     map[string]func() any
	nameByType map[reflect.Type]string
	cmdNames   []string
}

func (p *descPackage) GetName() string { return p.desc.Name + "Package" }

func (p *descPackage) InstanceOf(compName string) (any, bool) {
	newFn, ok := p.byName[compName]
	if !ok {
		return nil, false
	}
	return newFn(), true
}

func (p *descPackage) NameOf(comp any) string {
	name, _ := p.nameByType[reflect.TypeOf(comp)]
	return name
}

func (p *descPackage) CommandNames() []string {
	out := make([]string, len(p.cmdNames))
	copy(out, p.cmdNames)
	return out
}

func (p *descPackage) Describe() *PackageDesc { return p.desc }
