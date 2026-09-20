//go:build !linux

package ui

import (
	"fmt"
	"os"
	"strings"
)

type lineEditor struct{ history []string }

func newLineEditor() *lineEditor { return &lineEditor{} }

// ReadLine reads exactly one line, one byte at a time, with no read-ahead
// buffering. A buffered reader (bufio.Reader, as an earlier version used)
// can pull more than one line out of the OS pipe/tty in a single syscall;
// those extra bytes then sit in Go-level memory and are lost to any child
// process a1s later hands stdin to (e.g. `aliyun configure`), which would
// see EOF instead of the remaining input — exactly the case when a user
// pastes a profile name, region, and keys in quick succession. Reading
// unbuffered means nothing is ever consumed from the OS stdin handle beyond
// the current line, so a subsequent child process sees the rest intact.
func (e *lineEditor) ReadLine(prompt string, candidates func(string) []string) (string, error) {
	fmt.Print(prompt)
	var sb strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return strings.TrimSpace(sb.String()), nil
			}
			sb.WriteByte(buf[0])
		}
		if err != nil {
			return strings.TrimSpace(sb.String()), err
		}
	}
}
