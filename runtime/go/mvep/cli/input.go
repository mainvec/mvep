package cli

import (
	"io"
	"os"
)

// resolveInput resolves a command payload from the --input flag and stdin,
// returning the raw payload bytes. Empty input yields the empty payload "{}",
// not a parse error; checkRequired then applies normally.
//
// Resolution order:
//   - input == "-": read stdin explicitly. Explicit "-" and the implicit pipe
//     are the same single consumer of stdin (the implicit path exists because
//     --input was absent), so this reads stdin regardless of whether it is a
//     terminal or a pipe (#53).
//   - input == "": no --input flag. If stdin is not a character device, read
//     it implicitly (a pipe). If stdin is a TTY, do not block — yield "{}".
//   - otherwise: read the named file.
func resolveInput(input string, stdin io.Reader, stdinIsTTY bool) ([]byte, error) {
	switch {
	case input == "-":
		return readAll(stdin)
	case input != "":
		return os.ReadFile(input)
	default:
		// No --input flag. Read stdin implicitly only when it is not a
		// character device (i.e. a pipe); a TTY stdin must not block.
		if !stdinIsTTY {
			return readAll(stdin)
		}
		return []byte("{}"), nil
	}
}

// readAll reads all of r, returning "{}" when the input is empty.
func readAll(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return []byte("{}"), nil
	}
	return b, nil
}
