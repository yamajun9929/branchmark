package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

type terminal struct {
	state string
}

var pendingInput []byte
var suppressMouseTerminator bool

const (
	mouseTrackingEnable  = "\x1b[?1000h\x1b[?1006h"
	mouseTrackingDisable = "\x1b[?1000l\x1b[?1006l"
	escapeReadTimeout    = 60 * time.Millisecond
	escapeReadPollDelay  = 2 * time.Millisecond
)

func enterTerminal() (*terminal, error) {
	state, err := stty("-g")
	if err != nil {
		return nil, err
	}
	if _, err := stty("raw", "-echo"); err != nil {
		return nil, err
	}
	fmt.Print("\x1b[?1049h\x1b[?25l" + mouseTrackingEnable)
	return &terminal{state: strings.TrimSpace(state)}, nil
}

func (t *terminal) restore() {
	fmt.Print(mouseTrackingDisable + "\x1b[?25h\x1b[?1049l")
	if t != nil && t.state != "" {
		_, _ = stty(t.state)
	}
}

func stty(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	return string(out), err
}

func terminalSize() (int, int) {
	out, err := stty("size")
	if err != nil {
		return 80, 24
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 80, 24
	}
	rows, err1 := strconv.Atoi(fields[0])
	cols, err2 := strconv.Atoi(fields[1])
	if err1 != nil || err2 != nil || rows <= 0 || cols <= 0 {
		return 80, 24
	}
	return cols, rows
}

type key struct {
	name         string
	r            rune
	mouseButton  int
	mouseX       int
	mouseY       int
	mouseRelease bool
}

func readKey() key {
	ch, err := readByte()
	if err != nil {
		return key{name: "error"}
	}
	switch ch {
	case 3:
		return key{name: "ctrl-c"}
	case 9:
		return key{name: "tab"}
	case 13:
		return key{name: "enter"}
	case 27:
		return readEscape()
	case 32:
		return key{name: "space", r: ' '}
	case 127:
		return key{name: "backspace"}
	}
	if ch < 32 {
		return key{name: fmt.Sprintf("ctrl-%d", ch)}
	}
	if ch >= utf8.RuneSelf {
		return readUTF8Rune(ch)
	}
	return key{name: "rune", r: rune(ch)}
}

func readByte() (byte, error) {
	for {
		var ch byte
		if len(pendingInput) > 0 {
			ch = pendingInput[0]
			pendingInput = pendingInput[1:]
		} else {
			var b [1]byte
			_, err := os.Stdin.Read(b[:])
			if err != nil {
				return b[0], err
			}
			ch = b[0]
		}
		if suppressMouseTerminator {
			suppressMouseTerminator = false
			if ch == 'm' || ch == 'M' {
				continue
			}
		}
		return ch, nil
	}
}

func readUTF8Rune(first byte) key {
	buf := []byte{first}
	for len(buf) < utf8.UTFMax && !utf8.FullRune(buf) {
		ch, err := readByte()
		if err != nil {
			break
		}
		buf = append(buf, ch)
	}
	r, _ := utf8.DecodeRune(buf)
	if r == utf8.RuneError {
		return key{name: "rune", r: rune(first)}
	}
	return key{name: "rune", r: r}
}

func readEscape() key {
	payload := pendingInput
	pendingInput = nil
	if len(payload) == 0 {
		time.Sleep(15 * time.Millisecond)
	}
	fd := int(os.Stdin.Fd())
	_ = syscall.SetNonblock(fd, true)
	defer func() { _ = syscall.SetNonblock(fd, false) }()
	if len(payload) == 0 {
		buf := make([]byte, 64)
		n, _ := os.Stdin.Read(buf)
		if n == 0 {
			return key{name: "esc"}
		}
		payload = buf[:n]
	}
	payload = completeEscapePayload(payload, time.Now().Add(escapeReadTimeout))
	if isIncompleteSGRMousePayload(payload) {
		suppressMouseTerminator = true
		return key{name: "esc"}
	}
	first, rest := splitEscapePayload(payload)
	if len(rest) > 0 {
		pendingInput = append(append([]byte(nil), rest...), pendingInput...)
	}
	seq := string(first)
	if mouse, ok := parseSGRMouse(seq); ok {
		return mouse
	}
	switch {
	case strings.HasPrefix(seq, "[A"):
		return key{name: "up"}
	case strings.HasPrefix(seq, "[B"):
		return key{name: "down"}
	case strings.HasPrefix(seq, "[C"):
		return key{name: "right"}
	case strings.HasPrefix(seq, "[D"):
		return key{name: "left"}
	case strings.HasPrefix(seq, "[H"):
		return key{name: "home"}
	case strings.HasPrefix(seq, "[F"):
		return key{name: "end"}
	case strings.HasPrefix(seq, "[Z"):
		return key{name: "shift-tab"}
	case strings.HasPrefix(seq, "[1~"):
		return key{name: "home"}
	case strings.HasPrefix(seq, "[3~"):
		return key{name: "delete"}
	case strings.HasPrefix(seq, "[4~"):
		return key{name: "end"}
	case strings.HasPrefix(seq, "[5~"):
		return key{name: "pageup"}
	case strings.HasPrefix(seq, "[6~"):
		return key{name: "pagedown"}
	}
	return key{name: "esc"}
}

func completeEscapePayload(payload []byte, deadline time.Time) []byte {
	for isIncompleteSGRMousePayload(payload) && time.Now().Before(deadline) {
		time.Sleep(escapeReadPollDelay)
		var buf [64]byte
		n, _ := os.Stdin.Read(buf[:])
		if n > 0 {
			payload = append(payload, buf[:n]...)
		}
	}
	return payload
}

func splitEscapePayload(payload []byte) ([]byte, []byte) {
	if len(payload) == 0 {
		return payload, nil
	}
	if len(payload) >= 2 && payload[0] == '[' {
		switch payload[1] {
		case 'A', 'B', 'C', 'D', 'H', 'F', 'Z':
			return payload[:2], payload[2:]
		case '1', '3', '4', '5', '6':
			if len(payload) >= 3 && payload[2] == '~' {
				return payload[:3], payload[3:]
			}
		case '<':
			for i := 2; i < len(payload); i++ {
				if payload[i] == 'M' || payload[i] == 'm' {
					return payload[:i+1], payload[i+1:]
				}
			}
		}
	}
	return payload, nil
}

func isIncompleteSGRMousePayload(payload []byte) bool {
	if len(payload) < 2 || payload[0] != '[' || payload[1] != '<' {
		return false
	}
	for i := 2; i < len(payload); i++ {
		if payload[i] == 'M' || payload[i] == 'm' {
			return false
		}
	}
	return true
}

func parseSGRMouse(seq string) (key, bool) {
	if !strings.HasPrefix(seq, "[<") {
		return key{}, false
	}
	if len(seq) < 7 {
		return key{}, false
	}
	final := seq[len(seq)-1]
	if final != 'M' && final != 'm' {
		return key{}, false
	}
	parts := strings.Split(seq[2:len(seq)-1], ";")
	if len(parts) != 3 {
		return key{}, false
	}
	button, err1 := strconv.Atoi(parts[0])
	x, err2 := strconv.Atoi(parts[1])
	y, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil || x <= 0 || y <= 0 {
		return key{}, false
	}
	return key{
		name:         "mouse",
		mouseButton:  button,
		mouseX:       x,
		mouseY:       y,
		mouseRelease: final == 'm',
	}, true
}
