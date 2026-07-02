package db

import (
	"context"
	"testing"
)

func TestRebindFor(t *testing.T) {
	tests := []struct {
		name   string
		dbType DBType
		in     string
		want   string
	}{
		{"sqlite no-op", DBTypeSQLite, "SELECT * FROM t WHERE a = ? AND b = ?", "SELECT * FROM t WHERE a = ? AND b = ?"},
		{"mysql no-op", DBTypeMySQL, "INSERT INTO t (a,b) VALUES (?, ?)", "INSERT INTO t (a,b) VALUES (?, ?)"},
		{"mariadb no-op", DBTypeMariaDB, "UPDATE t SET a = ? WHERE id = ?", "UPDATE t SET a = ? WHERE id = ?"},
		{"postgres numbers placeholders", DBTypePostgres, "INSERT INTO t (a,b,c) VALUES (?, ?, ?)", "INSERT INTO t (a,b,c) VALUES ($1, $2, $3)"},
		{"postgres single placeholder", DBTypePostgres, "SELECT * FROM t WHERE id = ?", "SELECT * FROM t WHERE id = $1"},
		{"postgres none", DBTypePostgres, "SELECT COUNT(*) FROM t", "SELECT COUNT(*) FROM t"},
		{"postgres ignores ? in string literal", DBTypePostgres,
			"SELECT * FROM t WHERE label = 'is it ?' AND id = ?",
			"SELECT * FROM t WHERE label = 'is it ?' AND id = $1"},
		{"postgres handles escaped quote in literal", DBTypePostgres,
			"SELECT * FROM t WHERE s = 'a''b?' AND id = ?",
			"SELECT * FROM t WHERE s = 'a''b?' AND id = $1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rebindFor(tt.dbType, tt.in); got != tt.want {
				t.Errorf("rebindFor(%s)\n in:   %q\n got:  %q\n want: %q", tt.dbType, tt.in, got, tt.want)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 1: "1", 9: "9", 10: "10", 42: "42", 100: "100", 12345: "12345"}
	for n, want := range cases {
		if got := itoa(n); got != want {
			t.Errorf("itoa(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestDialectDDLPrimitives(t *testing.T) {
	// Save and restore the global so we don't disturb other tests.
	orig := currentDBType
	defer func() { currentDBType = orig }()

	checks := []struct {
		dbType   DBType
		pk       string
		suffix   string
		tsSubstr string
	}{
		{DBTypeSQLite, "id INTEGER PRIMARY KEY AUTOINCREMENT", "", "DATETIME"},
		{DBTypePostgres, "id SERIAL PRIMARY KEY", "", "TIMESTAMP"},
		{DBTypeMySQL, "id BIGINT AUTO_INCREMENT PRIMARY KEY", " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", "DATETIME"},
		{DBTypeMariaDB, "id BIGINT AUTO_INCREMENT PRIMARY KEY", " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", "DATETIME"},
	}
	for _, c := range checks {
		currentDBType = c.dbType
		if got := autoIncrementPK(); got != c.pk {
			t.Errorf("autoIncrementPK(%s) = %q, want %q", c.dbType, got, c.pk)
		}
		if got := tableSuffix(); got != c.suffix {
			t.Errorf("tableSuffix(%s) = %q, want %q", c.dbType, got, c.suffix)
		}
		if got := timestampDefault(); !contains(got, c.tsSubstr) {
			t.Errorf("timestampDefault(%s) = %q, want it to contain %q", c.dbType, got, c.tsSubstr)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TestDialectDDLSQLiteRoundTrip guards the dialect-aware DDL ports: on the
// default SQLite backend the three previously-SQLite-only tables must still be
// created and accept round-tripped data.
func TestDialectDDLSQLiteRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Init(tmpDir + "/dialect.db"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() { _ = Close() }()

	for _, tbl := range []string{"web_settings", "model_profiles", "llm_usage"} {
		var name string
		err := DB.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("%s table not created: %v", tbl, err)
		}
	}

	// web_settings upsert round-trip
	if err := SaveWebSetting("theme", "dark"); err != nil {
		t.Fatalf("SaveWebSetting insert error = %v", err)
	}
	if err := SaveWebSetting("theme", "light"); err != nil {
		t.Fatalf("SaveWebSetting update error = %v", err)
	}
	if got, _ := GetWebSetting("theme"); got != "light" {
		t.Errorf("GetWebSetting = %q, want light", got)
	}

	// model_profiles round-trip
	if err := SaveModelProfile(&ModelProfile{Name: "p1", Provider: "openai", Model: "gpt-4"}); err != nil {
		t.Fatalf("SaveModelProfile error = %v", err)
	}
	profiles, err := GetModelProfiles(context.Background(), false)
	if err != nil || len(profiles) == 0 {
		t.Fatalf("GetModelProfiles error = %v, n = %d", err, len(profiles))
	}

	// llm_usage round-trip
	if err := RecordLLMUsage(LLMUsageRecord{RequestID: "r1", User: "u", Provider: "openai", Model: "gpt-4"}); err != nil {
		t.Fatalf("RecordLLMUsage error = %v", err)
	}
}
