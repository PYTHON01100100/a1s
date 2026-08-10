//go:build linux

package ui

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

type lineEditor struct {
	history []string
	maxHist int
}

func newLineEditor() *lineEditor { return &lineEditor{maxHist: 200} }

func (e *lineEditor) ReadLine(prompt string, candidates func(string) []string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !isTerminal(fd) {
		fmt.Print(prompt)
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		return strings.TrimSpace(line), err
	}
	old, err := makeRaw(fd)
	if err != nil {
		return "", err
	}
	defer restore(fd, old)

	fmt.Print(prompt)
	buf := []rune{}
	pos := 0
	hist := len(e.history)
	for {
		var b [1]byte
		_, err := os.Stdin.Read(b[:])
		if err != nil {
			return "", err
		}
		switch b[0] {
		case 3: // Ctrl-C
			fmt.Print("^C\n")
			return "", nil
		case 4: // Ctrl-D
			if len(buf) == 0 {
				return "", ioEOF{}
			}
		case '\r', '\n':
			fmt.Print("\r\n")
			line := strings.TrimSpace(string(buf))
			if line != "" {
				if len(e.history) == 0 || e.history[len(e.history)-1] != line {
					e.history = append(e.history, line)
					if len(e.history) > e.maxHist {
						e.history = e.history[len(e.history)-e.maxHist:]
					}
				}
			}
			return line, nil
		case 127, 8:
			if pos > 0 {
				buf = append(buf[:pos-1], buf[pos:]...)
				pos--
				redraw(prompt, buf, pos)
			}
		case '\t':
			if candidates == nil {
				continue
			}
			matches := candidates(string(buf))
			if len(matches) == 0 {
				continue
			}
			if len(matches) == 1 {
				buf = []rune(matches[0])
				pos = len(buf)
				redraw(prompt, buf, pos)
				continue
			}
			sort.Strings(matches)
			fmt.Print("\r\n" + strings.Join(matches, "   ") + "\r\n")
			redraw(prompt, buf, pos)
		case 27:
			var seq [2]byte
			if _, err := os.Stdin.Read(seq[:]); err != nil {
				continue
			}
			if seq[0] != '[' {
				continue
			}
			switch seq[1] {
			case 'A': // up
				if len(e.history) > 0 && hist > 0 {
					hist--
					buf = []rune(e.history[hist])
					pos = len(buf)
					redraw(prompt, buf, pos)
				}
			case 'B': // down
				if hist < len(e.history)-1 {
					hist++
					buf = []rune(e.history[hist])
					pos = len(buf)
				} else {
					hist = len(e.history)
					buf = nil
					pos = 0
				}
				redraw(prompt, buf, pos)
			case 'C':
				if pos < len(buf) {
					pos++
					fmt.Print("\x1b[C")
				}
			case 'D':
				if pos > 0 {
					pos--
					fmt.Print("\x1b[D")
				}
			}
		default:
			if b[0] >= 32 {
				r := rune(b[0])
				buf = append(buf, 0)
				copy(buf[pos+1:], buf[pos:])
				buf[pos] = r
				pos++
				redraw(prompt, buf, pos)
			}
		}
	}
}

type ioEOF struct{}

func (ioEOF) Error() string { return "EOF" }

func redraw(prompt string, buf []rune, pos int) {
	fmt.Printf("\r\x1b[2K%s%s", prompt, string(buf))
	back := len(buf) - pos
	if back > 0 {
		fmt.Printf("\x1b[%dD", back)
	}
}

func isTerminal(fd int) bool {
	var t syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&t)), 0, 0, 0)
	return errno == 0
}

func makeRaw(fd int) (*syscall.Termios, error) {
	var old syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&old)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	raw := old
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	_, _, errno = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&raw)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return &old, nil
}

func restore(fd int, old *syscall.Termios) {
	if old == nil {
		return
	}
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(old)), 0, 0, 0)
}
