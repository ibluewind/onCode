package diff

import (
	"fmt"
	"strings"
)

// UnifiedFile produces a stable unified diff for one path (a/ and b/ prefixes).
// oldContent/newContent are full file texts; empty means file absent.
func UnifiedFile(path, oldContent, newContent string) string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		path = "unknown"
	}
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	if oldContent == "" && newContent == "" {
		return ""
	}

	var b strings.Builder
	if oldContent == "" {
		fmt.Fprintf(&b, "--- /dev/null\n+++ b/%s\n", path)
	} else if newContent == "" {
		fmt.Fprintf(&b, "--- a/%s\n+++ /dev/null\n", path)
	} else {
		fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", path, path)
	}

	ops := lcsOps(oldLines, newLines)
	oldStart := 1
	newStart := 1
	if len(oldLines) == 0 {
		oldStart = 0
	}
	if len(newLines) == 0 {
		newStart = 0
	}
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart, len(oldLines), newStart, len(newLines))
	for _, op := range ops {
		switch op.Kind {
		case opKeep:
			b.WriteByte(' ')
			b.WriteString(op.Line)
			b.WriteByte('\n')
		case opDel:
			b.WriteByte('-')
			b.WriteString(op.Line)
			b.WriteByte('\n')
		case opAdd:
			b.WriteByte('+')
			b.WriteString(op.Line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// UnifiedJoin concatenates per-file diffs in path order with no extra separators.
func UnifiedJoin(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(p)
		if !strings.HasSuffix(p, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

type opKind int

const (
	opKeep opKind = iota
	opDel
	opAdd
)

type diffOp struct {
	Kind opKind
	Line string
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	// Preserve empty trailing line semantics: strings.Split keeps trailing empty after final \n.
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func lcsOps(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			ops = append(ops, diffOp{Kind: opKeep, Line: a[i]})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{Kind: opDel, Line: a[i]})
			i++
		} else {
			ops = append(ops, diffOp{Kind: opAdd, Line: b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{Kind: opDel, Line: a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{Kind: opAdd, Line: b[j]})
	}
	return ops
}
