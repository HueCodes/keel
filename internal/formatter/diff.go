package formatter

import (
	"fmt"
	"strings"
)

// Diff generates a unified diff between original and formatted content
func Diff(filename, original, formatted string) string {
	if original == formatted {
		return ""
	}

	origLines := strings.Split(original, "\n")
	fmtLines := strings.Split(formatted, "\n")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s\n", filename))
	sb.WriteString(fmt.Sprintf("+++ %s\n", filename))

	// Generate hunks using a simple diff algorithm
	hunks := generateHunks(origLines, fmtLines)

	for _, hunk := range hunks {
		sb.WriteString(hunk.String())
	}

	return sb.String()
}

// DiffLine represents a line in a diff
type DiffLine struct {
	Type byte   // ' ', '+', '-'
	Text string
}

// Hunk represents a diff hunk
type Hunk struct {
	OrigStart, OrigCount int
	NewStart, NewCount   int
	Lines                []DiffLine
}

// String formats a hunk as unified diff
func (h *Hunk) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
		h.OrigStart, h.OrigCount, h.NewStart, h.NewCount))
	for _, line := range h.Lines {
		sb.WriteByte(line.Type)
		sb.WriteString(line.Text)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// generateHunks generates diff hunks between two sets of lines
func generateHunks(orig, new []string) []*Hunk {
	// Compute LCS (Longest Common Subsequence) for diffing
	lcs := computeLCS(orig, new)

	var hunks []*Hunk
	var currentHunk *Hunk

	origIdx, newIdx, lcsIdx := 0, 0, 0
	contextLines := 3 // Lines of context around changes

	for origIdx < len(orig) || newIdx < len(new) {
		// Check if we're on a matching line
		if lcsIdx < len(lcs) && origIdx < len(orig) && newIdx < len(new) &&
			orig[origIdx] == lcs[lcsIdx] && new[newIdx] == lcs[lcsIdx] {
			// Matching line
			if currentHunk != nil {
				// Add context line to current hunk
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{Type: ' ', Text: orig[origIdx]})
				currentHunk.OrigCount++
				currentHunk.NewCount++

				// Check if we should close the hunk
				if shouldCloseHunk(orig, new, lcs, origIdx, newIdx, lcsIdx, contextLines) {
					hunks = append(hunks, currentHunk)
					currentHunk = nil
				}
			}
			origIdx++
			newIdx++
			lcsIdx++
		} else {
			// Difference found
			if currentHunk == nil {
				// Start new hunk with context
				start := max(0, origIdx-contextLines)
				currentHunk = &Hunk{
					OrigStart: start + 1, // 1-based
					NewStart:  max(0, newIdx-contextLines) + 1,
				}
				// Add leading context
				for i := start; i < origIdx; i++ {
					currentHunk.Lines = append(currentHunk.Lines, DiffLine{Type: ' ', Text: orig[i]})
					currentHunk.OrigCount++
					currentHunk.NewCount++
				}
			}

			// Add removed lines
			for origIdx < len(orig) && (lcsIdx >= len(lcs) || orig[origIdx] != lcs[lcsIdx]) {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{Type: '-', Text: orig[origIdx]})
				currentHunk.OrigCount++
				origIdx++
			}

			// Add added lines
			for newIdx < len(new) && (lcsIdx >= len(lcs) || new[newIdx] != lcs[lcsIdx]) {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{Type: '+', Text: new[newIdx]})
				currentHunk.NewCount++
				newIdx++
			}
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, currentHunk)
	}

	return hunks
}

// computeLCS computes the Longest Common Subsequence
func computeLCS(a, b []string) []string {
	m, n := len(a), len(b)

	// Create DP table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Fill DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find LCS
	lcs := make([]string, dp[m][n])
	i, j, k := m, n, dp[m][n]-1
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs[k] = a[i-1]
			i--
			j--
			k--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// shouldCloseHunk checks if we should close the current hunk
func shouldCloseHunk(orig, new, lcs []string, origIdx, newIdx, lcsIdx, contextLines int) bool {
	// Look ahead to see if there are more changes coming
	lookAhead := contextLines * 2

	for i := 1; i <= lookAhead; i++ {
		o := origIdx + i
		n := newIdx + i
		l := lcsIdx + i

		if o >= len(orig) && n >= len(new) {
			return true // End of both
		}

		if l >= len(lcs) {
			return false // More changes coming
		}

		if o < len(orig) && n < len(new) {
			if orig[o] != lcs[l] || new[n] != lcs[l] {
				return false // More changes coming
			}
		}
	}

	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
