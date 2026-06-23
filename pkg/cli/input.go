package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

const (
	keyEnter     = 0x0d
	keyBackspace = 0x7f
	keyCtrlC     = 0x03
	keyCtrlD     = 0x04
	keyCtrlL     = 0x0c
	keyEscape    = 0x1b
	keyDelete    = 0x08
)

// readLine reads a line of input with history navigation support.
// It uses raw terminal mode so arrow keys can be captured.
// Falls back to bufio.Scanner when stdin is not a terminal.
func (c *CLI) readLine() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return c.readLineFallback()
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return c.readLineFallback()
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	var buf []byte
	cursorPos := 0
	histIdx := c.history.Len()

	for {
		key, err := readKey()
		if err != nil {
			return "", err
		}

		switch len(key) {
		case 1:
			switch key[0] {
			case keyEnter:
				fmt.Print("\r\n")
				return string(buf), nil

			case keyBackspace, keyDelete:
				if cursorPos > 0 && len(buf) > 0 {
					// Walk back to find the start of the UTF-8 character
					start := cursorPos - 1
					for start > 0 && buf[start]&0xC0 == 0x80 {
						start--
					}
					buf = append(buf[:start], buf[cursorPos:]...)
					cursorPos = start
					refreshLine("", buf, cursorPos)
				}

			case keyCtrlC:
				return "", fmt.Errorf("interrupt")

			case keyCtrlD:
				if len(buf) == 0 {
					return "", io.EOF
				}

			case keyCtrlL:
				fmt.Print(ClearScreen())
				PrintSplash(c.version.Version)
				refreshLine("", buf, cursorPos)

			default:
				if key[0] >= 0x20 {
					if key[0] <= 0x7e {
						// ASCII printable
						buf = insertAt(buf, cursorPos, key[0])
						cursorPos++
					} else {
						// Multi-byte UTF-8: read continuation bytes
						n := utf8SequenceLen(key[0])
						seq := make([]byte, n)
						seq[0] = key[0]
						for i := 1; i < n; i++ {
							var b [1]byte
							os.Stdin.Read(b[:])
							seq[i] = b[0]
						}
						buf = append(buf[:cursorPos], append(seq, buf[cursorPos:]...)...)
						cursorPos += len(seq)
					}
					refreshLine("", buf, cursorPos)
				}
			}

		case 3:
			if key[0] == keyEscape && key[1] == '[' {
				switch key[2] {
				case 'A': // Up arrow
					entry, ok := c.history.PreviousWithIdx(&histIdx)
					if ok {
						buf = []byte(entry)
						cursorPos = len(buf)
					}
					refreshLine("", buf, cursorPos)

				case 'B': // Down arrow
					entry, ok := c.history.NextWithIdx(&histIdx)
					if ok {
						buf = []byte(entry)
						cursorPos = len(buf)
					} else {
						buf = nil
						cursorPos = 0
					}
					refreshLine("", buf, cursorPos)
				}
			}
		}
	}
}

// readKey reads a single key press. Returns 1-3 bytes.
// Arrow keys return 3 bytes: ESC [ A/B/C/D
func readKey() ([]byte, error) {
	var buf [1]byte
	_, err := os.Stdin.Read(buf[:])
	if err != nil {
		return nil, err
	}

	if buf[0] == keyEscape {
		var seq [2]byte
		n, _ := os.Stdin.Read(seq[:])
		if n == 0 {
			return []byte{keyEscape}, nil
		}
		if n == 2 && seq[0] == '[' {
			return []byte{keyEscape, '[', seq[1]}, nil
		}
		return []byte{keyEscape, seq[0]}, nil
	}

	return buf[:], nil
}

// refreshLine clears the current line and reprints the prompt and buffer.
func refreshLine(prompt string, buf []byte, cursorPos int) {
	if prompt == "" {
		prompt = RenderPrompt()
	}
	line := prompt + string(buf)
	fmt.Print("\r\033[K" + line)
	// Use visual width for cursor positioning (essential for CJK/UTF-8)
	cursorVisual := runewidth.StringWidth(prompt) + runewidth.StringWidth(string(buf[:cursorPos]))
	lineVisual := runewidth.StringWidth(line)
	if move := lineVisual - cursorVisual; move > 0 {
		fmt.Printf("\033[%dD", move)
	}
}

// utf8SequenceLen returns the length (in bytes) of a UTF-8 sequence
// given its first byte.
func utf8SequenceLen(first byte) int {
	switch {
	case first >= 0xF0:
		return 4
	case first >= 0xE0:
		return 3
	case first >= 0xC0:
		return 2
	default:
		return 1
	}
}

// insertAt inserts b at position pos in the slice.
func insertAt(s []byte, pos int, b byte) []byte {
	s = append(s, 0)
	copy(s[pos+1:], s[pos:])
	s[pos] = b
	return s
}

// readLineFallback reads a line using bufio.Scanner (used when stdin is not a terminal).
func (c *CLI) readLineFallback() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}
