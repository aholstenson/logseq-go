// Package textmerge combines the changes made to a text with a version of the
// text that has other changes, line by line.
//
// The graph uses this to save pages. The writer does not always give back the
// exact text it read, so writing a whole page would change lines that the user
// did not edit. Instead the lines that the edit changed are merged into the
// file as it is on disk.
package textmerge

import "strings"

// maxEdits is the most line edits that Diff looks for between two texts. More
// edits than this gives up, as the time and memory grow with the square of the
// number of edits.
const maxEdits = 4000

// Merge applies the changes from base to changed onto original, where
// original is another version of base. The lines of original that the changes
// do not touch are kept as they are.
//
// When a change from base to changed overlaps a place where original differs
// from base, the lines of changed are used for that place.
//
// Returns false if the texts differ too much to compare.
func Merge(base string, original string, changed string) (string, bool) {
	baseLines := splitLines(base)
	originalLines := splitLines(original)
	changedLines := splitLines(changed)

	originalHunks, ok := diff(baseLines, originalLines)
	if !ok {
		return "", false
	}

	changedHunks, ok := diff(baseLines, changedLines)
	if !ok {
		return "", false
	}

	out := strings.Builder{}
	position := 0
	for _, group := range groupHunks(originalHunks, changedHunks) {
		writeLines(&out, baseLines[position:group.start])

		hunks := group.changed
		if len(group.changed) == 0 {
			hunks = group.original
		}

		writeHunks(&out, baseLines, group.start, group.end, hunks)
		position = group.end
	}

	writeLines(&out, baseLines[position:])
	return out.String(), true
}

// hunk replaces the lines of base from start up to end with other lines. A
// hunk with the same start and end inserts lines.
type hunk struct {
	start int
	end   int
	lines []string
}

// group is a place in base where hunks overlap. If both texts changed the
// place, the hunks of changed are used.
type group struct {
	start    int
	end      int
	original []hunk
	changed  []hunk
}

// splitLines splits a text into lines that keep their line breaks, so that
// joining them gives the text back.
func splitLines(str string) []string {
	if str == "" {
		return nil
	}

	return strings.SplitAfter(str, "\n")
}

func writeLines(out *strings.Builder, lines []string) {
	for _, line := range lines {
		out.WriteString(line)
	}
}

// writeHunks writes the lines of base from start up to end, with the hunks
// applied to them.
func writeHunks(out *strings.Builder, base []string, start int, end int, hunks []hunk) {
	position := start
	for _, h := range hunks {
		writeLines(out, base[position:h.start])
		writeLines(out, h.lines)
		position = h.end
	}

	writeLines(out, base[position:end])
}

// groupHunks puts the hunks of both texts in the order of base, and joins the
// hunks that overlap into one group.
func groupHunks(original []hunk, changed []hunk) []group {
	type sided struct {
		hunk
		isChanged bool
	}

	all := make([]sided, 0, len(original)+len(changed))
	i, j := 0, 0
	for i < len(original) || j < len(changed) {
		takeOriginal := j == len(changed) ||
			(i < len(original) && before(original[i], changed[j]))
		if takeOriginal {
			all = append(all, sided{hunk: original[i]})
			i++
		} else {
			all = append(all, sided{hunk: changed[j], isChanged: true})
			j++
		}
	}

	groups := make([]group, 0, len(all))
	for _, h := range all {
		if len(groups) > 0 && overlaps(h.hunk, groups[len(groups)-1]) {
			last := &groups[len(groups)-1]
			if h.end > last.end {
				last.end = h.end
			}
		} else {
			groups = append(groups, group{start: h.start, end: h.end})
		}

		last := &groups[len(groups)-1]
		if h.isChanged {
			last.changed = append(last.changed, h.hunk)
		} else {
			last.original = append(last.original, h.hunk)
		}
	}

	return groups
}

// before returns true if hunk a comes before hunk b in base. Lines that are
// inserted at a place come before lines that replace the lines there.
func before(a hunk, b hunk) bool {
	if a.start != b.start {
		return a.start < b.start
	}

	return a.start == a.end && b.start != b.end
}

// overlaps returns true if the hunk touches the same lines of base as the
// group. Two insertions at the same place also overlap, as the order of their
// lines is not known.
func overlaps(h hunk, g group) bool {
	hEmpty := h.start == h.end
	gEmpty := g.start == g.end

	switch {
	case hEmpty && gEmpty:
		return h.start == g.start
	case hEmpty:
		return g.start < h.start && h.start < g.end
	case gEmpty:
		return h.start < g.start && g.start < h.end
	default:
		return h.start < g.end && g.start < h.end
	}
}

// diff finds the hunks that change the lines of a into the lines of b.
// Returns false if more than maxEdits edits are needed.
func diff(a []string, b []string) ([]hunk, bool) {
	// The lines are compared as numbers, which is faster than comparing them
	// as text.
	ids := map[string]int{}
	toIDs := func(lines []string) []int {
		out := make([]int, len(lines))
		for i, line := range lines {
			id, ok := ids[line]
			if !ok {
				id = len(ids)
				ids[line] = id
			}

			out[i] = id
		}
		return out
	}

	matches, ok := matchingLines(toIDs(a), toIDs(b))
	if !ok {
		return nil, false
	}

	hunks := []hunk{}
	nextA, nextB := 0, 0
	for _, m := range append(matches, [2]int{len(a), len(b)}) {
		if m[0] > nextA || m[1] > nextB {
			hunks = append(hunks, hunk{start: nextA, end: m[0], lines: b[nextB:m[1]]})
		}

		nextA, nextB = m[0]+1, m[1]+1
	}

	return hunks, true
}

// matchingLines finds the longest list of lines that a and b have in the same
// order, as pairs of positions in a and b, using the algorithm of Myers.
func matchingLines(a []int, b []int) ([][2]int, bool) {
	// The lines at the start and the end that are the same are matched first,
	// which keeps the search small when only a few lines differ.
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}

	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix &&
		a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}

	middle, ok := myers(a[prefix:len(a)-suffix], b[prefix:len(b)-suffix])
	if !ok {
		return nil, false
	}

	matches := make([][2]int, 0, prefix+len(middle)+suffix)
	for i := 0; i < prefix; i++ {
		matches = append(matches, [2]int{i, i})
	}

	for _, m := range middle {
		matches = append(matches, [2]int{m[0] + prefix, m[1] + prefix})
	}

	for i := suffix; i > 0; i-- {
		matches = append(matches, [2]int{len(a) - i, len(b) - i})
	}

	return matches, true
}

// myers finds the matching lines of a and b with the greedy algorithm from
// "An O(ND) Difference Algorithm and Its Variations" by Eugene W. Myers.
func myers(a []int, b []int) ([][2]int, bool) {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return nil, true
	}

	max := n + m
	offset := max + 1
	v := make([]int, 2*max+3)

	// trace keeps the furthest positions before each round d, for the
	// diagonals -d-1 to d+1, to find the path back.
	var trace [][]int

	for d := 0; d <= max; d++ {
		if d > maxEdits {
			return nil, false
		}

		trace = append(trace, append([]int(nil), v[offset-d-1:offset+d+2]...))

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			} else {
				x = v[offset+k-1] + 1
			}

			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}

			v[offset+k] = x

			if x >= n && y >= m {
				return backtrack(trace, n, m), true
			}
		}
	}

	return nil, false
}

// backtrack follows the path that myers found back from the end, collecting
// the lines that match on the way.
func backtrack(trace [][]int, n int, m int) [][2]int {
	var matches [][2]int
	x, y := n, m

	for d := len(trace) - 1; d > 0; d-- {
		v := trace[d]
		at := func(k int) int { return v[k+d+1] }

		k := x - y
		var previousK int
		if k == -d || (k != d && at(k-1) < at(k+1)) {
			previousK = k + 1
		} else {
			previousK = k - 1
		}

		previousX := at(previousK)
		previousY := previousX - previousK

		// The edit moves from the previous position to the start of the run
		// of matching lines that ends at x, y.
		startX := previousX
		if previousK == k-1 {
			startX = previousX + 1
		}

		for x > startX {
			x--
			y--
			matches = append(matches, [2]int{x, y})
		}

		x, y = previousX, previousY
	}

	// What is left is the run of matching lines at the start.
	for x > 0 && y > 0 {
		x--
		y--
		matches = append(matches, [2]int{x, y})
	}

	for i, j := 0, len(matches)-1; i < j; i, j = i+1, j-1 {
		matches[i], matches[j] = matches[j], matches[i]
	}

	return matches
}
