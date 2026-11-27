package codex

import "strings"

// patchFile is one file an apply_patch call touches, with the lines it adds.
type patchFile struct {
	Path  string
	New   bool   // *** Add File: the call writes a new file (Write); anything else edits one (Edit)
	Added string // the patch's + lines for this file, without the +
}

// parsePatch reads the files out of Codex's apply_patch text:
//
//	*** Begin Patch
//	*** Add File: <path>       a new file; every line after it is +<text>
//	*** Update File: <path>    an edit, with @@ hunks of " ", "-" and "+" lines
//	*** Move to: <path>        the updated file's new name: an edit target too
//	*** Delete File: <path>
//	*** End Patch
//
// A file named twice (an update, then a move) is one target per name.
func parsePatch(patch string) []patchFile {
	var files []patchFile
	var added []string
	cur := -1
	flush := func() {
		if cur >= 0 {
			files[cur].Added = strings.Join(added, "\n")
		}
		added = nil
	}
	start := func(path string, isNew bool) {
		flush()
		files = append(files, patchFile{Path: strings.TrimSpace(path), New: isNew})
		cur = len(files) - 1
	}
	inUpdate := false // in an Update File's hunks, where a line that starts with " ", "+" or "-" is the file's text
	for _, line := range strings.Split(strings.ReplaceAll(patch, "\r\n", "\n"), "\n") {
		if inUpdate && line != "" && strings.ContainsRune(" +-", rune(line[0])) {
			if cur >= 0 && line[0] == '+' {
				added = append(added, line[1:])
			}
			continue // " *** Update File: x" here is a context line, as Codex reads it
		}
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "*** ") {
			line = t // Codex reads a hunk's first line, its header, with spaces around it too
		}
		inUpdate = inUpdate && !strings.HasPrefix(line, "*** ") ||
			strings.HasPrefix(line, "*** Update File: ") || strings.HasPrefix(line, "*** Move to: ")
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			start(strings.TrimPrefix(line, "*** Add File: "), true)
		case strings.HasPrefix(line, "*** Update File: "):
			start(strings.TrimPrefix(line, "*** Update File: "), false)
		case strings.HasPrefix(line, "*** Delete File: "):
			start(strings.TrimPrefix(line, "*** Delete File: "), false)
		case strings.HasPrefix(line, "*** Move to: "):
			// It follows its Update File line: the new name is a target too, and the hunks after it are its text.
			start(strings.TrimPrefix(line, "*** Move to: "), false)
		case strings.HasPrefix(line, "*** "):
			flush()
			cur = -1 // Begin Patch, End Patch, End of File
		case cur >= 0 && strings.HasPrefix(line, "+"):
			added = append(added, line[1:])
		}
	}
	flush()
	var out []patchFile
	for _, f := range files {
		if f.Path != "" {
			out = append(out, f)
		}
	}
	return out
}
