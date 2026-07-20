package prreview

import (
	"regexp"
	"strconv"
	"strings"
)

// DiffLine is one line of a unified-diff hunk with its resolved line numbers.
// OldLine is 0 for added lines; NewLine is 0 for deleted lines.
type DiffLine struct {
	Origin  byte // '+', '-', or ' '
	OldLine int
	NewLine int
	Text    string // line content without the origin marker
}

// FileDiff is the parsed diff of a single file.
type FileDiff struct {
	Path    string // new path ("b/" side); for deleted files, the old path
	Deleted bool
	Lines   []DiffLine
}

var hunkRE = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// parseDiff parses a unified diff into per-file line records. It tolerates
// git extended headers and binary-file notices.
func parseDiff(diff string) []FileDiff {
	var files []FileDiff
	var cur *FileDiff
	var oldN, newN int
	inHunk := false

	for raw := range strings.SplitSeq(diff, "\n") {
		switch {
		case strings.HasPrefix(raw, "diff --git "):
			files = append(files, FileDiff{})
			cur = &files[len(files)-1]
			inHunk = false
		case cur != nil && strings.HasPrefix(raw, "--- "):
			if cur.Path == "" && raw != "--- /dev/null" {
				cur.Path = strings.TrimPrefix(strings.TrimPrefix(raw, "--- "), "a/")
			}
		case cur != nil && strings.HasPrefix(raw, "+++ "):
			if raw == "+++ /dev/null" {
				cur.Deleted = true
			} else {
				cur.Path = strings.TrimPrefix(strings.TrimPrefix(raw, "+++ "), "b/")
			}
		case cur != nil && strings.HasPrefix(raw, "@@"):
			m := hunkRE.FindStringSubmatch(raw)
			if m == nil {
				continue
			}
			oldN, _ = strconv.Atoi(m[1])
			newN, _ = strconv.Atoi(m[2])
			inHunk = true
		case cur != nil && inHunk && len(raw) > 0:
			switch raw[0] {
			case '+':
				cur.Lines = append(cur.Lines, DiffLine{Origin: '+', NewLine: newN, Text: raw[1:]})
				newN++
			case '-':
				cur.Lines = append(cur.Lines, DiffLine{Origin: '-', OldLine: oldN, Text: raw[1:]})
				oldN++
			case ' ':
				cur.Lines = append(cur.Lines, DiffLine{Origin: ' ', OldLine: oldN, NewLine: newN, Text: raw[1:]})
				oldN++
				newN++
			case '\\': // "\ No newline at end of file"
			default:
				inHunk = false
			}
		}
	}
	return files
}

// anchorSet holds every (path, side, line) a review comment may attach to.
type anchorSet map[string]bool

func anchorKey(path, side string, line int) string {
	return path + "\x00" + side + "\x00" + strconv.Itoa(line)
}

func diffAnchors(files []FileDiff) anchorSet {
	set := anchorSet{}
	for _, f := range files {
		for _, l := range f.Lines {
			if l.OldLine > 0 {
				set[anchorKey(f.Path, "LEFT", l.OldLine)] = true
			}
			if l.NewLine > 0 {
				set[anchorKey(f.Path, "RIGHT", l.NewLine)] = true
			}
		}
	}
	return set
}
