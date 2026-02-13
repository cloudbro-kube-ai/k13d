package ui

import (
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		target    string
		wantMatch bool
		wantMin   int // minimum expected score (0 means don't check)
	}{
		{"empty pattern matches anything", "", "hello", true, 0},
		{"empty target no match", "a", "", false, 0},
		{"exact match", "nginx", "nginx", true, 100},
		{"prefix match", "ngi", "nginx-deployment", true, 50},
		{"case insensitive", "NGI", "nginx", true, 0},
		{"fuzzy match scattered", "nd", "nginx-deployment", true, 0},
		{"fuzzy match first letters", "np", "nginx-proxy", true, 0},
		{"no match", "xyz", "nginx", false, 0},
		{"pattern longer than target", "nginx-deployment-extra", "nginx", false, 0},
		{"single char match", "n", "nginx", true, 0},
		{"consecutive chars bonus", "ngin", "nginx", true, 30},
		{"word boundary bonus", "dp", "nginx-deployment-proxy", true, 0},
		{"all chars must appear in order", "ba", "abc", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score, positions := FuzzyMatch(tt.pattern, tt.target)
			if matched != tt.wantMatch {
				t.Errorf("FuzzyMatch(%q, %q) matched = %v, want %v", tt.pattern, tt.target, matched, tt.wantMatch)
			}
			if matched && tt.wantMin > 0 && score < tt.wantMin {
				t.Errorf("FuzzyMatch(%q, %q) score = %d, want >= %d", tt.pattern, tt.target, score, tt.wantMin)
			}
			if matched && len(tt.pattern) > 0 && len(positions) != len([]rune(tt.pattern)) {
				t.Errorf("FuzzyMatch(%q, %q) positions count = %d, want %d", tt.pattern, tt.target, len(positions), len([]rune(tt.pattern)))
			}
		})
	}
}

func TestFuzzyMatchScoring(t *testing.T) {
	// Exact match should score higher than prefix match
	_, exactScore, _ := FuzzyMatch("nginx", "nginx")
	_, prefixScore, _ := FuzzyMatch("nginx", "nginx-deployment")
	if exactScore <= prefixScore {
		t.Errorf("exact score (%d) should be > prefix score (%d)", exactScore, prefixScore)
	}

	// Prefix match should score higher than scattered match
	_, scatteredScore, _ := FuzzyMatch("nd", "nginx-deployment")
	_, prefScore2, _ := FuzzyMatch("ng", "nginx-deployment")
	if prefScore2 <= scatteredScore {
		t.Errorf("prefix score (%d) should be > scattered score (%d)", prefScore2, scatteredScore)
	}
}

func TestFuzzyFilter(t *testing.T) {
	rows := [][]string{
		{"default", "nginx-deployment", "Running"},
		{"default", "redis-master", "Running"},
		{"kube-system", "coredns", "Running"},
		{"default", "nginx-proxy", "Pending"},
		{"default", "my-app", "Running"},
	}

	t.Run("empty pattern returns all", func(t *testing.T) {
		result := FuzzyFilter(rows, "", 1)
		if len(result) != len(rows) {
			t.Errorf("got %d rows, want %d", len(result), len(rows))
		}
	})

	t.Run("filter by name column", func(t *testing.T) {
		result := FuzzyFilter(rows, "nginx", 1)
		if len(result) != 2 {
			t.Errorf("got %d rows, want 2", len(result))
		}
	})

	t.Run("fuzzy filter", func(t *testing.T) {
		result := FuzzyFilter(rows, "nd", 1)
		// Should match "nginx-deployment" (n...d)
		if len(result) == 0 {
			t.Error("expected at least one match for 'nd'")
		}
		// First result should be best match
		if result[0][1] != "nginx-deployment" {
			t.Errorf("expected first result to be nginx-deployment, got %s", result[0][1])
		}
	})

	t.Run("no matches", func(t *testing.T) {
		result := FuzzyFilter(rows, "zzz", 1)
		if len(result) != 0 {
			t.Errorf("got %d rows, want 0", len(result))
		}
	})

	t.Run("out of bounds column", func(t *testing.T) {
		result := FuzzyFilter(rows, "test", 99)
		if len(result) != 0 {
			t.Errorf("got %d rows, want 0 for out of bounds column", len(result))
		}
	})

	t.Run("sorted by score", func(t *testing.T) {
		result := FuzzyFilter(rows, "nginx", 1)
		if len(result) < 2 {
			t.Fatal("expected at least 2 results")
		}
		// Both should contain nginx but scores should be ordered
		_, score1, _ := FuzzyMatch("nginx", result[0][1])
		_, score2, _ := FuzzyMatch("nginx", result[1][1])
		if score1 < score2 {
			t.Errorf("results not sorted by score: first=%d, second=%d", score1, score2)
		}
	})
}

func TestLabelFilter(t *testing.T) {
	rows := [][]string{
		{"default", "nginx", "app=nginx"},
		{"default", "redis", "app=redis,tier=backend"},
		{"kube-system", "coredns", "app=coredns"},
		{"default", "frontend", "app=web,tier=frontend"},
	}

	t.Run("empty expression returns all", func(t *testing.T) {
		result := LabelFilter(rows, "")
		if len(result) != len(rows) {
			t.Errorf("got %d rows, want %d", len(result), len(rows))
		}
	})

	t.Run("key=value match", func(t *testing.T) {
		result := LabelFilter(rows, "app=nginx")
		if len(result) != 1 {
			t.Errorf("got %d rows, want 1", len(result))
		}
		if len(result) > 0 && result[0][1] != "nginx" {
			t.Errorf("got %s, want nginx", result[0][1])
		}
	})

	t.Run("key=value case insensitive", func(t *testing.T) {
		result := LabelFilter(rows, "APP=NGINX")
		if len(result) != 1 {
			t.Errorf("got %d rows, want 1", len(result))
		}
	})

	t.Run("key!=value exclude", func(t *testing.T) {
		result := LabelFilter(rows, "app!=nginx")
		if len(result) != 3 {
			t.Errorf("got %d rows, want 3", len(result))
		}
		for _, row := range result {
			if row[1] == "nginx" {
				t.Error("nginx should have been excluded")
			}
		}
	})

	t.Run("key exists check", func(t *testing.T) {
		result := LabelFilter(rows, "tier")
		if len(result) != 2 {
			t.Errorf("got %d rows, want 2 (redis and frontend have tier)", len(result))
		}
	})

	t.Run("no match", func(t *testing.T) {
		result := LabelFilter(rows, "env=production")
		if len(result) != 0 {
			t.Errorf("got %d rows, want 0", len(result))
		}
	})
}

func TestHighlightFuzzyMatch(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		pattern string
		want    string
	}{
		{"empty pattern", "nginx", "", "nginx"},
		{"no match", "nginx", "xyz", "nginx"},
		{"single char", "nginx", "n", "[yellow]n[white]ginx"},
		{"consecutive", "nginx", "ng", "[yellow]ng[white]inx"},
		{"scattered", "nginx-dep", "nd", "[yellow]n[white]ginx-[yellow]d[white]ep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightFuzzyMatch(tt.text, tt.pattern)
			if got != tt.want {
				t.Errorf("highlightFuzzyMatch(%q, %q) = %q, want %q", tt.text, tt.pattern, got, tt.want)
			}
		})
	}
}
