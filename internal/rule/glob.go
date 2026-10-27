package rule

import (
	"fmt"
	"regexp"
	"strings"
)

// globRegexp compiles a file glob into a regexp over an absolute path.
//
//	**       any characters, across directories
//	*        any characters within one path segment
//	?        one character within a segment
//	{a,b}    either alternative (no nesting)
//
// A relative glob matches at any directory boundary, so "src/**/*.ts" matches
// /home/me/repo/src/a/b.ts. An absolute glob is anchored at the root.
func globRegexp(glob string) (*regexp.Regexp, error) {
	var b strings.Builder
	if strings.HasPrefix(glob, "/") {
		b.WriteString("^")
	} else {
		b.WriteString("(^|/)")
	}
	inBrace := false
	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch {
		case c == '*' && i+1 < len(glob) && glob[i+1] == '*':
			i++
			if i+1 < len(glob) && glob[i+1] == '/' {
				i++
				b.WriteString("(.*/)?") // "**/" also matches zero directories
			} else {
				b.WriteString(".*")
			}
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		case c == '{':
			if inBrace {
				return nil, fmt.Errorf("nested { in %q", glob)
			}
			inBrace = true
			b.WriteString("(")
		case c == '}' && inBrace:
			inBrace = false
			b.WriteString(")")
		case c == ',' && inBrace:
			b.WriteString("|")
		default:
			b.WriteString(regexp.QuoteMeta(glob[i : i+1])) // byte-wise: keeps UTF-8 sequences intact
		}
	}
	if inBrace {
		return nil, fmt.Errorf("unclosed { in %q", glob)
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
