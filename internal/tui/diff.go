package tui

import (
	"fmt"
	"strings"
)

// FormatDiff computes a unified diff between oldContent and newContent,
// returning a color-coded string suitable for terminal display.
// Additions are green, deletions are red, and context lines are dim.
func FormatDiff(oldContent, newContent string) string {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	hunks := computeHunks(oldLines, newLines, 3)

	if len(hunks) == 0 {
		return Dim("(no changes)")
	}

	var b strings.Builder
	for _, h := range hunks {
		b.WriteString(formatHunkHeader(h))
		b.WriteByte('\n')
		for _, dl := range h.lines {
			b.WriteString(formatDiffLine(dl))
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatFileDiff renders a unified diff with a file-path header.
// The header shows the file path in bold, followed by the diff body.
func FormatFileDiff(path string, oldContent, newContent string) string {
	var b strings.Builder
	b.WriteString(Bold("--- " + path))
	b.WriteByte('\n')
	b.WriteString(Bold("+++ " + path))
	b.WriteByte('\n')
	b.WriteString(FormatDiff(oldContent, newContent))
	return b.String()
}

type diffOp int

const (
	diffCtx diffOp = iota
	diffAdd
	diffDel
)

type diffLine struct {
	op   diffOp
	text string
}

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []diffLine
}

func formatHunkHeader(h hunk) string {
	return Cyan(fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.oldStart+1, h.oldCount, h.newStart+1, h.newCount))
}

func formatDiffLine(dl diffLine) string {
	switch dl.op {
	case diffAdd:
		return Green("+ " + dl.text)
	case diffDel:
		return Red("- " + dl.text)
	default:
		return Dim("  " + dl.text)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// lcs computes the longest common subsequence table via classic DP.
func lcs(a, b []string) [][]int {
	m, n := len(a), len(b)
	table := make([][]int, m+1)
	for i := range table {
		table[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				table[i][j] = table[i-1][j-1] + 1
			} else {
				table[i][j] = max(table[i-1][j], table[i][j-1])
			}
		}
	}
	return table
}

type editOp int

const (
	editEqual editOp = iota
	editInsert
	editDelete
)

type edit struct {
	op   editOp
	oldI int
	newI int
}

// backtrack reconstructs the edit sequence from the LCS table.
func backtrack(table [][]int, a, b []string) []edit {
	var edits []edit
	i, j := len(a), len(b)

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			edits = append(edits, edit{op: editEqual, oldI: i - 1, newI: j - 1})
			i--
			j--
		} else if j > 0 && (i == 0 || table[i][j-1] >= table[i-1][j]) {
			edits = append(edits, edit{op: editInsert, newI: j - 1})
			j--
		} else {
			edits = append(edits, edit{op: editDelete, oldI: i - 1})
			i--
		}
	}

	reverseEdits(edits)
	return edits
}

func reverseEdits(edits []edit) {
	for l, r := 0, len(edits)-1; l < r; l, r = l+1, r-1 {
		edits[l], edits[r] = edits[r], edits[l]
	}
}

// computeHunks groups edits into unified-diff hunks with the given context radius.
func computeHunks(oldLines, newLines []string, ctx int) []hunk {
	table := lcs(oldLines, newLines)
	edits := backtrack(table, oldLines, newLines)

	if len(edits) == 0 {
		return nil
	}

	changeIdxs := changeIndices(edits)
	if len(changeIdxs) == 0 {
		return nil
	}

	groups := groupChanges(changeIdxs, edits, ctx)
	hunks := make([]hunk, 0, len(groups))
	for _, g := range groups {
		hunks = append(hunks, buildHunk(g, edits, oldLines, newLines, ctx))
	}
	return hunks
}

func changeIndices(edits []edit) []int {
	var idxs []int
	for i, e := range edits {
		if e.op != editEqual {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

// groupChanges merges change indices that are within 2*ctx of each other.
func groupChanges(idxs []int, edits []edit, ctx int) [][]int {
	if len(idxs) == 0 {
		return nil
	}
	groups := [][]int{{idxs[0]}}
	for _, idx := range idxs[1:] {
		cur := groups[len(groups)-1]
		last := cur[len(cur)-1]
		if idx-last <= 2*ctx {
			groups[len(groups)-1] = append(cur, idx)
		} else {
			groups = append(groups, []int{idx})
		}
	}
	return groups
}

func buildHunk(group []int, edits []edit, oldLines, newLines []string, ctx int) hunk {
	first, last := group[0], group[len(group)-1]
	lo := clamp(first-ctx, 0, len(edits))
	hi := clamp(last+ctx+1, 0, len(edits))

	var h hunk
	h.oldStart, h.newStart = editPos(edits, lo)
	oldEnd, newEnd := editEndPos(edits, lo, hi, oldLines, newLines)
	h.oldCount = oldEnd - h.oldStart
	h.newCount = newEnd - h.newStart

	for i := lo; i < hi; i++ {
		e := edits[i]
		switch e.op {
		case editEqual:
			h.lines = append(h.lines, diffLine{op: diffCtx, text: oldLines[e.oldI]})
		case editInsert:
			h.lines = append(h.lines, diffLine{op: diffAdd, text: newLines[e.newI]})
		case editDelete:
			h.lines = append(h.lines, diffLine{op: diffDel, text: oldLines[e.oldI]})
		}
	}
	return h
}

func editPos(edits []edit, idx int) (int, int) {
	if idx >= len(edits) {
		return 0, 0
	}
	e := edits[idx]
	switch e.op {
	case editEqual:
		return e.oldI, e.newI
	case editDelete:
		return e.oldI, e.newI
	case editInsert:
		return e.oldI, e.newI
	}
	return 0, 0
}

func editEndPos(edits []edit, lo, hi int, oldLines, newLines []string) (int, int) {
	oldEnd, newEnd := 0, 0
	for i := lo; i < hi; i++ {
		e := edits[i]
		switch e.op {
		case editEqual:
			if e.oldI+1 > oldEnd {
				oldEnd = e.oldI + 1
			}
			if e.newI+1 > newEnd {
				newEnd = e.newI + 1
			}
		case editDelete:
			if e.oldI+1 > oldEnd {
				oldEnd = e.oldI + 1
			}
		case editInsert:
			if e.newI+1 > newEnd {
				newEnd = e.newI + 1
			}
		}
	}
	return oldEnd, newEnd
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
