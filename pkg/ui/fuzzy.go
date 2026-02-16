package ui

import (
	"sort"
	"strings"
	"unicode"
)

// FuzzyMatch returns true if the pattern fuzzy-matches the target string.
// Each character in pattern must appear in target in order (case-insensitive).
// Returns match result, score (higher is better), and matched positions.
func FuzzyMatch(pattern, target string) (bool, int, []int) {
	if pattern == "" {
		return true, 0, nil
	}
	if target == "" {
		return false, 0, nil
	}

	patternLower := strings.ToLower(pattern)
	targetLower := strings.ToLower(target)

	patternRunes := []rune(patternLower)
	targetRunes := []rune(targetLower)
	originalRunes := []rune(target)

	if len(patternRunes) > len(targetRunes) {
		return false, 0, nil
	}

	// Find matched positions (greedy left-to-right)
	positions := make([]int, 0, len(patternRunes))
	pi := 0
	for ti := 0; ti < len(targetRunes) && pi < len(patternRunes); ti++ {
		if targetRunes[ti] == patternRunes[pi] {
			positions = append(positions, ti)
			pi++
		}
	}

	if pi < len(patternRunes) {
		return false, 0, nil
	}

	// Calculate score
	score := 0

	// Exact match bonus
	if targetLower == patternLower {
		score += 100
	}

	// Prefix match bonus
	if strings.HasPrefix(targetLower, patternLower) {
		score += 50
	}

	// Consecutive character bonus
	for i := 1; i < len(positions); i++ {
		if positions[i] == positions[i-1]+1 {
			score += 10
		}
	}

	// Start-of-word bonus (after separator: -, _, /, .)
	for _, pos := range positions {
		if pos == 0 {
			score += 15
		} else {
			prev := originalRunes[pos-1]
			if prev == '-' || prev == '_' || prev == '/' || prev == '.' || prev == ' ' {
				score += 10
			}
			// CamelCase boundary bonus
			if unicode.IsLower(prev) && unicode.IsUpper(originalRunes[pos]) {
				score += 10
			}
		}
	}

	// Penalty for unmatched length (shorter targets rank higher)
	score -= len(targetRunes) - len(patternRunes)

	return true, score, positions
}

// fuzzyResult holds a row with its match score for sorting.
type fuzzyResult struct {
	row   []string
	score int
}

// FuzzyFilter filters rows using fuzzy matching on a specific column.
// Results are sorted by match score (best matches first).
func FuzzyFilter(rows [][]string, pattern string, nameCol int) [][]string {
	if pattern == "" {
		return rows
	}

	var results []fuzzyResult
	for _, row := range rows {
		if nameCol >= len(row) {
			continue
		}
		matched, score, _ := FuzzyMatch(pattern, row[nameCol])
		if matched {
			results = append(results, fuzzyResult{row: row, score: score})
		}
	}

	// Sort by score descending (best matches first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	filtered := make([][]string, len(results))
	for i, r := range results {
		filtered[i] = r.row
	}
	return filtered
}

// LabelFilter filters rows by checking if any cell matches the label expression.
// Supported formats:
//   - "key=value"  — exact match on any cell containing "key=value" or cell equals value
//   - "key!=value" — exclude rows where any cell matches
//   - "key"        — check if any cell contains the key string
func LabelFilter(rows [][]string, labelExpr string) [][]string {
	if labelExpr == "" {
		return rows
	}

	var filtered [][]string

	if idx := strings.Index(labelExpr, "!="); idx > 0 {
		// key!=value — exclude matching rows
		key := strings.TrimSpace(labelExpr[:idx])
		value := strings.TrimSpace(labelExpr[idx+2:])
		target := strings.ToLower(key + "=" + value)
		valueLower := strings.ToLower(value)

		for _, row := range rows {
			exclude := false
			for _, cell := range row {
				cellLower := strings.ToLower(cell)
				if cellLower == valueLower || strings.Contains(cellLower, target) {
					exclude = true
					break
				}
			}
			if !exclude {
				filtered = append(filtered, row)
			}
		}
	} else if idx := strings.Index(labelExpr, "="); idx > 0 {
		// key=value — include matching rows
		key := strings.TrimSpace(labelExpr[:idx])
		value := strings.TrimSpace(labelExpr[idx+1:])
		target := strings.ToLower(key + "=" + value)
		valueLower := strings.ToLower(value)

		for _, row := range rows {
			for _, cell := range row {
				cellLower := strings.ToLower(cell)
				if cellLower == valueLower || strings.Contains(cellLower, target) {
					filtered = append(filtered, row)
					break
				}
			}
		}
	} else {
		// key (exists check) — include rows where any cell contains the key
		keyLower := strings.ToLower(strings.TrimSpace(labelExpr))
		for _, row := range rows {
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), keyLower) {
					filtered = append(filtered, row)
					break
				}
			}
		}
	}

	return filtered
}

// highlightFuzzyMatch wraps fuzzy-matched characters with color tags for tview.
func highlightFuzzyMatch(text, pattern string) string {
	if pattern == "" {
		return text
	}

	matched, _, positions := FuzzyMatch(pattern, text)
	if !matched || len(positions) == 0 {
		return text
	}

	runes := []rune(text)
	posSet := make(map[int]bool, len(positions))
	for _, p := range positions {
		posSet[p] = true
	}

	var result strings.Builder
	inHighlight := false
	for i, r := range runes {
		if posSet[i] {
			if !inHighlight {
				result.WriteString("[yellow]")
				inHighlight = true
			}
		} else {
			if inHighlight {
				result.WriteString("[white]")
				inHighlight = false
			}
		}
		result.WriteRune(r)
	}
	if inHighlight {
		result.WriteString("[white]")
	}
	return result.String()
}
