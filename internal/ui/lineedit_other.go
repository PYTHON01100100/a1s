//go:build !linux

package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type lineEditor struct{ history []string }

func newLineEditor() *lineEditor { return &lineEditor{} }
func (e *lineEditor) ReadLine(prompt string, candidates func(string) []string) (string, error) {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	return strings.TrimSpace(line), err
}
