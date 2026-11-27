package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// interactive reports whether init and uninstall may ask: both stdin and stdout are a terminal. An agent or CI
// runs them without one, so they never wait for an answer.
var interactive = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// makeRaw puts the terminal in raw mode for the arrow-key menus; an error falls back to numbered questions.
var makeRaw = func() (restore func(), err error) {
	fd := int(os.Stdin.Fd())
	st, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() { term.Restore(fd, st) }, nil
}

var errCancelled = errors.New("cancelled")

// chooser asks init's and uninstall's questions on one reader, so no answer is read ahead and lost.
type chooser struct {
	in  *bufio.Reader
	out io.Writer
}

func newChooser() *chooser { return &chooser{in: bufio.NewReader(os.Stdin), out: os.Stdout} }

// many asks which of labels to pick, at least one; on are the ones ticked at the start.
func (c *chooser) many(title string, labels []string, on []bool) ([]bool, error) {
	if restore, err := makeRaw(); err == nil {
		defer restore()
		return c.menu(title, labels, append([]bool(nil), on...), true)
	}
	return c.numbered(title, labels, on, true)
}

// one asks for one of labels; def is the answer an enter gives.
func (c *chooser) one(title string, labels []string, def int) (int, error) {
	on := make([]bool, len(labels))
	on[def] = true
	var picked []bool
	var err error
	if restore, rerr := makeRaw(); rerr == nil {
		defer restore()
		picked, err = c.menu(title, labels, on, false)
	} else {
		picked, err = c.numbered(title, labels, on, false)
	}
	if err != nil {
		return 0, err
	}
	for i, p := range picked {
		if p {
			return i, nil
		}
	}
	return def, nil
}

// menu is the raw-mode question: arrows (or j/k) move, space ticks (many) or the cursor picks (one), enter
// confirms, Ctrl-C cancels. A lone Esc does nothing: telling it from an arrow's Esc [ would need a timeout.
func (c *chooser) menu(title string, labels []string, on []bool, many bool) ([]bool, error) {
	hint := "↑/↓ to move, enter to choose"
	if many {
		hint = "↑/↓ to move, space to tick, enter to confirm"
	}
	fmt.Fprintf(c.out, "%s (%s)\r\n", title, hint)
	cur := 0
	for i, o := range on {
		if o {
			cur = i
			break
		}
	}
	status := "" // shown under the options: why enter didn't confirm
	draw := func() {
		for i, l := range labels {
			mark := "  "
			if i == cur {
				mark = "> "
			}
			box := ""
			if many {
				box = "[ ] "
				if on[i] {
					box = "[x] "
				}
			}
			fmt.Fprintf(c.out, "\r\x1b[2K%s%s%s\r\n", mark, box, l)
		}
		fmt.Fprintf(c.out, "\r\x1b[2K%s\r\n", status)
	}
	draw()
	for {
		b, err := c.in.ReadByte()
		if err != nil {
			return nil, errCancelled
		}
		status = ""
		switch b {
		case 3, 4: // Ctrl-C, Ctrl-D
			return nil, errCancelled
		case 27: // an arrow: Esc [ A/B. Anything else after Esc is read as its own key.
			if next, _ := c.in.Peek(1); len(next) == 0 || next[0] != '[' {
				continue
			}
			c.in.ReadByte()
			switch k, _ := c.in.ReadByte(); k {
			case 'A':
				cur = (cur + len(labels) - 1) % len(labels)
			case 'B':
				cur = (cur + 1) % len(labels)
			}
		case 'k':
			cur = (cur + len(labels) - 1) % len(labels)
		case 'j':
			cur = (cur + 1) % len(labels)
		case ' ':
			if many {
				on[cur] = !on[cur]
			}
		case '\r', '\n':
			if !many {
				for i := range on {
					on[i] = i == cur
				}
				return on, nil
			}
			if anyOn(on) {
				return on, nil
			}
			status = "  pick at least one agent (space ticks one)"
		}
		fmt.Fprintf(c.out, "\x1b[%dA", len(labels)+1) // back to the first option
		draw()
	}
}

// numbered is the question without raw mode: options by number, a comma list for many, enter for the default.
func (c *chooser) numbered(title string, labels []string, on []bool, many bool) ([]bool, error) {
	var def []string
	for i := range labels {
		if on[i] {
			def = append(def, strconv.Itoa(i+1))
		}
	}
	for {
		fmt.Fprintln(c.out, title)
		for i, l := range labels {
			fmt.Fprintf(c.out, "  %d) %s\n", i+1, l)
		}
		prompt := "Number"
		if many {
			prompt = "Numbers, comma-separated"
		}
		fmt.Fprintf(c.out, "%s [%s]: ", prompt, strings.Join(def, ","))
		line, err := c.in.ReadString('\n')
		if err != nil && line == "" {
			fmt.Fprintln(c.out)
			return nil, errCancelled
		}
		if picked, ok := parsePicks(strings.TrimSpace(line), len(labels), many); ok {
			if picked == nil {
				return on, nil // enter: the default
			}
			return picked, nil
		}
		fmt.Fprintf(c.out, "Pick %s from 1 to %d.\n", map[bool]string{true: "numbers", false: "a number"}[many], len(labels))
	}
}

// parsePicks reads "1", "1,2" or "" (nil: the default); ok is false for anything else, or for more than one
// number when one is asked.
func parsePicks(s string, n int, many bool) ([]bool, bool) {
	if s == "" {
		return nil, true
	}
	picked := make([]bool, n)
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' })
	if !many && len(parts) != 1 {
		return nil, false
	}
	for _, p := range parts {
		i, err := strconv.Atoi(p)
		if err != nil || i < 1 || i > n {
			return nil, false
		}
		picked[i-1] = true
	}
	return picked, anyOn(picked)
}

func anyOn(on []bool) bool {
	for _, o := range on {
		if o {
			return true
		}
	}
	return false
}
