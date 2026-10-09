package utils

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// diffContext is the number of unchanged lines shown around each change.
const diffContext = 3

type diffLine struct {
	kind byte // ' ' unchanged, '-' removed, '+' added
	text string
}

// UnifiedDiff renders a standard unified diff between oldText and newText.
// It returns an empty string when the texts are identical. The path is used
// only for the "---"/"+++" header lines.
func UnifiedDiff(oldText, newText, path string) string {
	if oldText == newText {
		return ""
	}

	dmp := diffmatchpatch.New()
	chars1, chars2, lineArray := dmp.DiffLinesToChars(oldText, newText)
	diffs := dmp.DiffMain(chars1, chars2, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	var lines []diffLine
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			lines = appendSplit(lines, ' ', d.Text)
		case diffmatchpatch.DiffDelete:
			lines = appendSplit(lines, '-', d.Text)
		case diffmatchpatch.DiffInsert:
			lines = appendSplit(lines, '+', d.Text)
		}
	}

	// Prefix sums of lines that exist in the old/new file before index i.
	oldBefore := make([]int, len(lines)+1)
	newBefore := make([]int, len(lines)+1)
	for i, l := range lines {
		oldBefore[i+1] = oldBefore[i]
		newBefore[i+1] = newBefore[i]
		if l.kind != '+' {
			oldBefore[i+1]++
		}
		if l.kind != '-' {
			newBefore[i+1]++
		}
	}

	// Positions of changed lines.
	var changed []int
	for i, l := range lines {
		if l.kind != ' ' {
			changed = append(changed, i)
		}
	}
	if len(changed) == 0 {
		return ""
	}

	// Group changes into hunks, merging those separated by <= 2*context lines.
	type span struct{ start, end int } // [start, end)
	spans := []span{}
	start := max(0, changed[0]-diffContext)
	for i := 1; i < len(changed); i++ {
		if changed[i]-changed[i-1]-1 > 2*diffContext {
			spans = append(spans, span{start, min(len(lines), changed[i-1]+diffContext+1)})
			start = max(0, changed[i]-diffContext)
		}
	}
	spans = append(spans, span{start, min(len(lines), changed[len(changed)-1]+diffContext+1)})

	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n", path)
	fmt.Fprintf(&b, "+++ b/%s\n", path)
	for _, s := range spans {
		oldStart, oldCount := oldBefore[s.start]+1, oldBefore[s.end]-oldBefore[s.start]
		newStart, newCount := newBefore[s.start]+1, newBefore[s.end]-newBefore[s.start]
		if oldCount == 0 {
			oldStart = oldBefore[s.start]
		}
		if newCount == 0 {
			newStart = newBefore[s.start]
		}
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
		for _, l := range lines[s.start:s.end] {
			b.WriteByte(l.kind)
			b.WriteString(l.text)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// appendSplit splits text into lines and appends each with the given prefix
// byte. A single trailing newline is treated as a line terminator, not an
// empty trailing line.
func appendSplit(dst []diffLine, kind byte, text string) []diffLine {
	if text == "" {
		return dst
	}
	parts := strings.Split(text, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	for _, p := range parts {
		dst = append(dst, diffLine{kind: kind, text: p})
	}
	return dst
}
