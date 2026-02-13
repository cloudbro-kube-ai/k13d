package anonymizer

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Anonymizer handles masking/unmasking of sensitive data in text.
// Thread-safe for concurrent use.
type Anonymizer struct {
	mu       sync.RWMutex
	enabled  bool
	mappings map[string]string // masked -> original
	reverse  map[string]string // original -> masked
	counters map[string]int    // per-category counters
	patterns []pattern
}

type pattern struct {
	re       *regexp.Regexp
	category string
}

var defaultPatterns = []pattern{
	// URLs must come before IPs/emails to avoid partial matches
	{re: regexp.MustCompile(`https?://[^\s"'<>]+`), category: "URL"},
	// Docker image refs: registry/org/image:tag or org/image:tag (with optional @sha256:...)
	{re: regexp.MustCompile(`[\w.-]+(?:\.[\w.-]+)+/[\w./-]+(:[a-zA-Z][\w.-]*)?(@sha256:[a-f0-9]{64})?`), category: "IMAGE"},
	// Email addresses
	{re: regexp.MustCompile(`[\w.+-]+@[\w.-]+\.\w{2,}`), category: "EMAIL"},
	// IPv4 addresses
	{re: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), category: "IP"},
	// API keys / tokens: 32+ character alphanumeric strings (with _-)
	{re: regexp.MustCompile(`\b[a-zA-Z0-9_-]{32,}\b`), category: "TOKEN"},
}

// New creates a new Anonymizer.
// If enabled is false, Anonymize/Deanonymize are no-ops.
func New(enabled bool) *Anonymizer {
	return &Anonymizer{
		enabled:  enabled,
		mappings: make(map[string]string),
		reverse:  make(map[string]string),
		counters: make(map[string]int),
		patterns: defaultPatterns,
	}
}

// Anonymize replaces sensitive patterns in text with placeholders.
func (a *Anonymizer) Anonymize(text string) string {
	if !a.enabled {
		return text
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Collect all matches with their positions
	type match struct {
		start    int
		end      int
		value    string
		category string
	}
	var matches []match

	for _, p := range a.patterns {
		locs := p.re.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			matches = append(matches, match{
				start:    loc[0],
				end:      loc[1],
				value:    text[loc[0]:loc[1]],
				category: p.category,
			})
		}
	}

	if len(matches) == 0 {
		return text
	}

	// Sort by start position descending, then by length descending (longer matches first)
	// so we can replace from the end without invalidating earlier positions
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start == matches[j].start {
			return (matches[i].end - matches[i].start) > (matches[j].end - matches[j].start)
		}
		return matches[i].start > matches[j].start
	})

	// Remove overlapping matches (keep longer/earlier-pattern match)
	filtered := make([]match, 0, len(matches))
	occupied := make([]bool, len(text))
	// Process longest matches first by sorting by length desc
	sort.Slice(matches, func(i, j int) bool {
		li := matches[i].end - matches[i].start
		lj := matches[j].end - matches[j].start
		if li == lj {
			return matches[i].start < matches[j].start
		}
		return li > lj
	})
	for _, m := range matches {
		overlap := false
		for i := m.start; i < m.end; i++ {
			if occupied[i] {
				overlap = true
				break
			}
		}
		if !overlap {
			filtered = append(filtered, m)
			for i := m.start; i < m.end; i++ {
				occupied[i] = true
			}
		}
	}

	// Sort by start position descending for safe replacement
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].start > filtered[j].start
	})

	result := text
	for _, m := range filtered {
		placeholder := a.getOrCreatePlaceholder(m.value, m.category)
		result = result[:m.start] + placeholder + result[m.end:]
	}

	return result
}

// Deanonymize restores all placeholders to original values.
func (a *Anonymizer) Deanonymize(text string) string {
	if !a.enabled {
		return text
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	result := text
	// Replace longer placeholders first to avoid partial matches
	placeholders := make([]string, 0, len(a.mappings))
	for p := range a.mappings {
		placeholders = append(placeholders, p)
	}
	sort.Slice(placeholders, func(i, j int) bool {
		return len(placeholders[i]) > len(placeholders[j])
	})

	for _, placeholder := range placeholders {
		result = strings.ReplaceAll(result, placeholder, a.mappings[placeholder])
	}
	return result
}

// Reset clears all mappings (call between conversations).
func (a *Anonymizer) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mappings = make(map[string]string)
	a.reverse = make(map[string]string)
	a.counters = make(map[string]int)
}

// MappingCount returns the number of active mappings.
func (a *Anonymizer) MappingCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.mappings)
}

// getOrCreatePlaceholder returns existing or creates new placeholder.
// Must be called with a.mu held.
func (a *Anonymizer) getOrCreatePlaceholder(value, category string) string {
	if placeholder, ok := a.reverse[value]; ok {
		return placeholder
	}

	a.counters[category]++
	placeholder := fmt.Sprintf("<%s_%d>", category, a.counters[category])

	a.mappings[placeholder] = value
	a.reverse[value] = placeholder
	return placeholder
}
