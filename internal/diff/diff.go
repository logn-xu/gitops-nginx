package diff

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// Result represents the result of a diff operation.
type Result struct {
	UnifiedDiff  string
	AddedLines   int
	RemovedLines int
}

// GenerateUnifiedDiff generates a unified diff between two strings and provides statistics.
func GenerateUnifiedDiff(from, to, fromFileName, toFileName string) (*Result, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(from),
		B:        difflib.SplitLines(to),
		FromFile: fromFileName,
		ToFile:   toFileName,
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unified diff: %w", err)
	}

	result := &Result{
		UnifiedDiff: text,
	}

	if text != "" {
		lines := difflib.SplitLines(text)
		// Skip header lines
		for _, line := range lines[2:] {
			if strings.HasPrefix(line, "+") {
				result.AddedLines++
			} else if strings.HasPrefix(line, "-") {
				result.RemovedLines++
			}
		}
	}

	return result, nil
}
