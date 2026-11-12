package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Terminal output helpers: home paths as ~, wrapped text, and bold or dim only on a terminal (never with NO_COLOR).

// tilde shows a path under the home directory as ~/...
func tilde(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if rel, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok {
		return "~/" + filepath.ToSlash(rel)
	}
	return p
}

func tildeAll(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = tilde(p)
	}
	return out
}

// wrap breaks text into lines of at most width runes at spaces, keeping at most maxLines; a cut ends in "…".
func wrap(text string, width, maxLines int) []string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(text) {
		if line != "" && len([]rune(line))+1+len([]rune(w)) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += w
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		last := []rune(strings.TrimRight(lines[maxLines-1], " ,;:."))
		if len(last) > width-1 {
			last = last[:width-1]
		}
		lines[maxLines-1] = string(last) + "…"
	}
	return lines
}

var styled = func() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	st, err := os.Stdout.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}()

func style(code, s string) string {
	if !styled {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func bold(s string) string { return style("1", s) }
func dim(s string) string  { return style("2", s) }
func cyan(s string) string { return style("36", s) }

// count is "1 rule" or "3 rules".
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
