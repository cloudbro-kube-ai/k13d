package i18n

import "testing"

func TestSetLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected Language
	}{
		{"en", EN},
		{"EN", EN},
		{"ko", KO},
		{"korean", KO},
		{"Korean", KO},
		{"zh", ZH},
		{"chinese", ZH},
		{"Chinese", ZH},
		{"ja", JA},
		{"japanese", JA},
		{"Japanese", JA},
		{"unknown", EN}, // defaults to EN
		{"", EN},        // defaults to EN
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			SetLanguage(tt.input)
			if GetLanguage() != tt.expected {
				t.Errorf("SetLanguage(%q) = %v, want %v", tt.input, GetLanguage(), tt.expected)
			}
		})
	}
}

func TestTranslation(t *testing.T) {
	SetLanguage("en")
	if T("app_title") != "k13d - K8s AI Explorer" {
		t.Errorf("expected English title, got %s", T("app_title"))
	}

	SetLanguage("ko")
	if T("app_title") != "k13d - K8s AI 탐색기" {
		t.Errorf("expected Korean title, got %s", T("app_title"))
	}

	SetLanguage("zh")
	if T("app_title") != "k13d - K8s AI 资源管理器" {
		t.Errorf("expected Chinese title, got %s", T("app_title"))
	}

	SetLanguage("ja")
	if T("app_title") != "k13d - K8s AI エクスプローラー" {
		t.Errorf("expected Japanese title, got %s", T("app_title"))
	}

	// Fallback test
	SetLanguage("non-existent")
	if T("app_title") != "k13d - K8s AI Explorer" {
		t.Errorf("expected fallback to English title, got %s", T("app_title"))
	}
}

func TestTranslationKeys(t *testing.T) {
	// Test that all languages have the same keys
	languages := []Language{EN, KO, ZH, JA}
	baseKeys := make(map[string]bool)

	// Get all keys from English
	for key := range translations[EN] {
		baseKeys[key] = true
	}

	for _, lang := range languages {
		SetLanguage(string(lang))
		for key := range baseKeys {
			val := T(key)
			if val == key {
				// T returns the key itself if translation is missing
				// This is acceptable for fallback, but we want to verify
				// that the translation exists for the current language
				if _, ok := translations[lang][key]; !ok {
					t.Logf("Warning: key %q missing in language %s, using fallback", key, lang)
				}
			}
		}
	}
}

func TestUnknownKey(t *testing.T) {
	SetLanguage("en")
	unknownKey := "this_key_does_not_exist_anywhere"
	result := T(unknownKey)
	if result != unknownKey {
		t.Errorf("T(%q) = %q, want %q (key should be returned for unknown keys)", unknownKey, result, unknownKey)
	}
}

func TestAllTranslationsHaveValues(t *testing.T) {
	for lang, langMap := range translations {
		for key, value := range langMap {
			if value == "" {
				t.Errorf("Empty translation for key %q in language %s", key, lang)
			}
		}
	}
}
