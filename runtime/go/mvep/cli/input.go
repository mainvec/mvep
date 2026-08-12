package cli

import (
	"fmt"
	"io"
	"os"
)

// resolveInput resolves a command payload from the --input flag and stdin,
// returning the raw payload bytes. Empty input yields the empty payload "{}",
// not a parse error; checkRequired then applies normally.
//
// Resolution order:
//   - input == "-": read stdin explicitly. If stdin is also not a character
//     device, the implicit-pipe path would claim it too — two consumers of "-"
//     in one invocation is an error, reported before any read.
//   - input == "": no --input flag. If stdin is not a character device, read
//     it implicitly (a pipe). If stdin is a TTY, do not block — yield "{}".
//   - otherwise: read the named file.
func resolveInput(input string, stdin io.Reader, stdinIsTTY bool) ([]byte, error) {
	switch {
	case input == "-":
		if !stdinIsTTY {
			return nil, fmt.Errorf("cannot use '-' twice: --input - and implicit stdin pipe both claim stdin")
		}
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
