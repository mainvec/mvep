package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/mainvec/mvep/runtime/go/mvep"
	"github.com/mainvec/ugo/cli"
)

// describeProjection is the versioned, hand-written JSON projection of a
// command, decoupled from the internal *mvep.CommandDesc so descriptor changes
// cannot silently change the public contract.
type describeProjection struct {
	Version     string      `json:"version"`
	Name        string      `json:"name"`
	Alias       string      `json:"alias,omitempty"`
	Group       string      `json:"group,omitempty"`
	Description string      `json:"description,omitempty"`
	Fields      []fieldProj `json:"fields"`
	Result      *resultProj `json:"result,omitempty"`
}

type fieldProj struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Repeated bool   `json:"repeated,omitempty"`
	Required bool   `json:"required,omitempty"`
	Ref      string `json:"ref,omitempty"`
}

type resultProj struct {
	Name string `json:"name"`
}

// registerList adds the `list` verb: svc mvep list prints the command names.
func (a *App) registerList(ns *cli.Command) {
	list := &cli.Command{
		Usage: "list",
		Short: "list command names",
		RunE: func(ctx *cli.Context, args []string) error {
			names := a.commandNames()
			if a.outputMode == "json" {
				// Machine form: an array of {name, description} objects.
				entries := make([]map[string]string, 0, len(names))
				for _, n := range names {
					cmd := a.commands[n]
					desc := ""
					if cmd != nil {
						desc = cmd.Desc
					}
					entries = append(entries, map[string]string{"name": n, "description": desc})
				}
				b, _ := json.Marshal(entries)
				ctx.Write(b)
				return nil
			}
			for _, n := range names {
				cmd := a.commands[n]
				desc := ""
				if cmd != nil {
					desc = cmd.Desc
				}
				fmt.Fprintf(ctx, "%-30s %s\n", n, desc)
			}
			return nil
		},
	}
	ns.AddCommand(list)
}

// registerDescribe adds the `describe` verb: svc mvep describe [command] emits a
// versioned JSON projection of one or all commands.
func (a *App) registerDescribe(ns *cli.Command) {
	describe := &cli.Command{
		Usage: "describe [command]",
		Short: "show a command's schema",
		RunE: func(ctx *cli.Context, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("mvep describe takes at most one command name")
			}
			if len(args) == 1 {
				cmdDesc, ok := a.commands[args[0]]
				if !ok {
					return fmt.Errorf("unknown command %q (valid: %s)", args[0], joinComma(a.commandNames()))
				}
				return writeJSON(ctx, projectCommand(cmdDesc, a.desc))
			}
			// No argument: describe all, as a JSON array.
			all := make([]describeProjection, 0, len(a.commands))
			for _, name := range a.commandNames() {
				all = append(all, projectCommand(a.commands[name], a.desc))
			}
			return writeJSON(ctx, all)
		},
	}
	ns.AddCommand(describe)
}

// projectCommand builds the projection for a command, resolving record ref
// fields by name so the output stays minimal.
func projectCommand(cmd *mvep.CommandDesc, desc *mvep.PackageDesc) describeProjection {
	proj := describeProjection{
		Version:     "1",
		Name:        cmd.Name,
		Alias:       cmd.Alias,
		Group:       cmd.Group,
		Description: cmd.Desc,
		Fields:      make([]fieldProj, 0, len(cmd.Fields)),
	}
	for i := range cmd.Fields {
		f := &cmd.Fields[i]
		fp := fieldProj{
			Name:     f.Name,
			Type:     fieldTypeName(f.Type),
			Repeated: f.Repeated,
			Required: f.Required,
		}
		if f.Ref != nil {
			fp.Ref = f.Ref.Name
		}
		proj.Fields = append(proj.Fields, fp)
	}
	if cmd.Result != nil {
		proj.Result = &resultProj{Name: cmd.Result.Name}
	}
	return proj
}

// fieldTypeName maps a FieldType to its spec name for the projection.
func fieldTypeName(t mvep.FieldType) string {
	names := map[mvep.FieldType]string{
		mvep.FieldString:    "string",
		mvep.FieldBool:      "bool",
		mvep.FieldInt32:     "int32",
		mvep.FieldInt64:     "int64",
		mvep.FieldUint32:    "uint32",
		mvep.FieldSint32:    "sint32",
		mvep.FieldFloat:     "float",
		mvep.FieldDouble:    "double",
		mvep.FieldBytes:     "bytes",
		mvep.FieldTimestamp: "timestamp",
		mvep.FieldDuration:  "duration",
		mvep.FieldUUID:      "uuid",
		mvep.FieldMap:       "map",
		mvep.FieldRecord:    "record",
	}
	if n, ok := names[t]; ok {
		return n
	}
	return "unknown"
}

// joinComma joins a slice with commas for error messages.
func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}

// writeJSON marshals v as JSON to the writer.
func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	w.Write(b)
	return nil
}
