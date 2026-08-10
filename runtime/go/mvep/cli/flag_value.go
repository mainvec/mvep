// Package cli provides a runtime CLI builder that drives a generated package's
// commands via a *mvep.PackageDesc and an Executor (local or remote). It is
// built on ugo/cli v0.7.0.
package cli

import (
	"strconv"

	"github.com/mainvec/ugo/cli"
)

// Uint32Var registers a uint32 flag on the given FlagSet. ugo v0.7.0 (and the
// stdlib flag.FlagSet it embeds) ships no Uint32Var, so mvep/cli carries this
// small custom flag.Value type. If ugo later ships the helper, swap it in.
func Uint32Var(fs *cli.FlagSet, p *uint32, name string, value uint32, usage string) {
	*p = value
	fs.Var(&uint32Value{p: p}, name, usage)
}

// Float32Var registers a float32 flag on the given FlagSet. ugo v0.7.0 (and the
// stdlib flag.FlagSet it embeds) ships no Float32Var, so mvep/cli carries this
// small custom flag.Value type. If ugo later ships the helper, swap it in.
func Float32Var(fs *cli.FlagSet, p *float32, name string, value float32, usage string) {
	*p = value
	fs.Var(&float32Value{p: p}, name, usage)
}

// -- uint32 Value -----------------------------------------------------------

type uint32Value struct{ p *uint32 }

func (u *uint32Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 32)
	if err != nil {
		return err
	}
	*u.p = uint32(v)
	return nil
}

func (u *uint32Value) Get() any { return uint32(*u.p) }

func (u *uint32Value) String() string {
	if u == nil || u.p == nil {
		return "0"
	}
	return strconv.FormatUint(uint64(*u.p), 10)
}

// -- float32 Value ----------------------------------------------------------

type float32Value struct{ p *float32 }

func (f *float32Value) Set(s string) error {
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return err
	}
	*f.p = float32(v)
	return nil
}

func (f *float32Value) Get() any { return float32(*f.p) }

func (f *float32Value) String() string {
	if f == nil || f.p == nil {
		return "0"
	}
	return strconv.FormatFloat(float64(*f.p), 'g', -1, 32)
}
