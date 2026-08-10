package cli

import (
	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/ugo/cli"
)

// flagBinding holds a parsed flag value and the field descriptor it binds to,
// for one command execution. T9 replaces this minimal binding with a full
// Ptr-driven type-switch; T8 only needs string + int32 to prove the flow.
type flagBinding struct {
	field  *mvep.FieldDesc
	strVal *string
	intVal *int32
}

// bindFlags registers a flag for every FieldDesc on the command's FlagSet and
// returns the bindings so applyBindings can write parsed values into the struct.
// This does NOT mutate the descriptor's Ptr; the parsed values are held in
// per-execution locals and written back via Ptr after parsing.
func bindFlags(fs *cli.FlagSet, cmdDesc *mvep.CommandDesc) []flagBinding {
	var bindings []flagBinding
	for i := range cmdDesc.Fields {
		f := &cmdDesc.Fields[i]
		switch f.Type {
		case mvep.FieldString:
			p := new(string)
			fs.StringVar(p, f.Name, "", f.Desc)
			bindings = append(bindings, flagBinding{field: f, strVal: p})
		case mvep.FieldInt32:
			p := new(int32)
			fs.Int32Var(p, f.Name, 0, f.Desc)
			bindings = append(bindings, flagBinding{field: f, intVal: p})
		default:
			// T9 covers the remaining types. For T8, unhandled types get a
			// string flag so the parser doesn't reject them.
			p := new(string)
			fs.StringVar(p, f.Name, "", f.Desc)
			bindings = append(bindings, flagBinding{field: f, strVal: p})
		}
	}
	return bindings
}

// applyBindings writes the parsed flag values into the command struct via the
// FieldDesc.Ptr accessors.
func applyBindings(cmd any, bindings []flagBinding) {
	for _, b := range bindings {
		target := b.field.Ptr(cmd)
		switch p := target.(type) {
		case *string:
			if b.strVal != nil {
				*p = *b.strVal
			}
		case *int32:
			if b.intVal != nil {
				*p = *b.intVal
			}
		}
	}
}