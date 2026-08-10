package cli

import (
	"testing"

	"github.com/mainvec/ugo/cli"
)

// TestUint32Var verifies T6: mvep/cli provides a uint32 flag binding via a
// custom flag.Value type, since ugo v0.7.0 ships no Uint32Var. The flag must
// parse valid unsigned 32-bit values and reject out-of-range/negative input.
func TestUint32Var(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    []string
		want    uint32
		wantErr bool
	}{
		{"default", []string{}, 0, false},
		{"zero", []string{"--n", "0"}, 0, false},
		{"value", []string{"--n", "42"}, 42, false},
		{"max", []string{"--n", "4294967295"}, 4294967295, false},
		{"overflow", []string{"--n", "4294967296"}, 0, true},
		{"negative", []string{"--n", "-1"}, 0, true},
		{"not a number", []string{"--n", "abc"}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var n uint32
			cmd := &cli.Command{
				Usage: "test",
				RunE: func(ctx *cli.Context, args []string) error {
					if n != tc.want {
						t.Errorf("got %d, want %d", n, tc.want)
					}
					return nil
				},
			}
			Uint32Var(cmd.Flags(), &n, "n", 0, "uint32 flag")
			err := cli.Execute(t.Context(), cmd, tc.args)
			switch {
			case err != nil && !tc.wantErr:
				t.Errorf("unexpected error: %v", err)
			case err == nil && tc.wantErr:
				t.Errorf("expected error, got nil")
			}
		})
	}
}

// TestFloat32Var verifies T6: mvep/cli provides a float32 flag binding via a
// custom flag.Value type, since ugo v0.7.0 ships no Float32Var. The flag must
// parse valid 32-bit floats and reject non-numeric input.
func TestFloat32Var(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    []string
		want    float32
		wantErr bool
	}{
		{"default", []string{}, 0, false},
		{"zero", []string{"--f", "0"}, 0, false},
		{"value", []string{"--f", "3.14"}, 3.14, false},
		{"negative", []string{"--f", "-2.5"}, -2.5, false},
		{"not a number", []string{"--f", "abc"}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f float32
			cmd := &cli.Command{
				Usage: "test",
				RunE: func(ctx *cli.Context, args []string) error {
					if f != tc.want {
						t.Errorf("got %g, want %g", f, tc.want)
					}
					return nil
				},
			}
			Float32Var(cmd.Flags(), &f, "f", 0, "float32 flag")
			err := cli.Execute(t.Context(), cmd, tc.args)
			switch {
			case err != nil && !tc.wantErr:
				t.Errorf("unexpected error: %v", err)
			case err == nil && tc.wantErr:
				t.Errorf("expected error, got nil")
			}
		})
	}
}