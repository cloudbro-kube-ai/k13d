package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCustomRolesTable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-roles-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "roles_test.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() { _ = Close() }()

	if err := InitCustomRolesTable(); err != nil {
		t.Fatalf("InitCustomRolesTable() error = %v", err)
	}

	// Verify table was created
	var tableName string
	row := DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='custom_roles'")
	if err := row.Scan(&tableName); err != nil {
		t.Errorf("custom_roles table not created: %v", err)
	}
}

func TestInitCustomRolesTable_NilDB(t *testing.T) {
	savedDB := DB
	DB = nil
	defer func() { DB = savedDB }()

	err := InitCustomRolesTable()
	if err == nil {
		t.Error("InitCustomRolesTable should error when DB is nil")
	}
}

func TestCustomRoles_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-roles-crud-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "roles_crud.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() { _ = Close() }()

	if err := InitCustomRolesTable(); err != nil {
		t.Fatalf("InitCustomRolesTable() error = %v", err)
	}

	// Save
	definition := `{"name":"developer","allowed_features":["dashboard","topology"]}`
	err = SaveCustomRole("developer", definition)
	if err != nil {
		t.Fatalf("SaveCustomRole failed: %v", err)
	}

	// Load
	def, err := GetCustomRole("developer")
	if err != nil {
		t.Fatalf("GetCustomRole failed: %v", err)
	}
	if def != definition {
		t.Errorf("expected definition %q, got %q", definition, def)
	}

	// List
	roles, err := ListCustomRoles()
	if err != nil {
		t.Fatalf("ListCustomRoles failed: %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(roles))
	}
	if roles[0].Name != "developer" {
		t.Errorf("expected role name 'developer', got %s", roles[0].Name)
	}

	// Delete
	err = DeleteCustomRole("developer")
	if err != nil {
		t.Fatalf("DeleteCustomRole failed: %v", err)
	}
	roles, _ = ListCustomRoles()
	if len(roles) != 0 {
		t.Errorf("expected 0 roles after delete, got %d", len(roles))
	}
}

func TestCustomRoles_SaveUpsert(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-roles-upsert-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := Init(filepath.Join(tmpDir, "upsert.db")); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() { _ = Close() }()

	if err := InitCustomRolesTable(); err != nil {
		t.Fatalf("InitCustomRolesTable() error = %v", err)
	}

	// Initial save
	err = SaveCustomRole("ops", `{"name":"ops","version":"1"}`)
	if err != nil {
		t.Fatalf("SaveCustomRole failed: %v", err)
	}

	// Upsert (update) same name
	err = SaveCustomRole("ops", `{"name":"ops","version":"2"}`)
	if err != nil {
		t.Fatalf("SaveCustomRole upsert failed: %v", err)
	}

	// Should still have only 1 role
	roles, _ := ListCustomRoles()
	if len(roles) != 1 {
		t.Errorf("expected 1 role after upsert, got %d", len(roles))
	}

	// Should have updated definition
	def, _ := GetCustomRole("ops")
	if def != `{"name":"ops","version":"2"}` {
		t.Errorf("expected updated definition, got %s", def)
	}
}

func TestCustomRoles_GetNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-roles-notfound-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := Init(filepath.Join(tmpDir, "notfound.db")); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() { _ = Close() }()

	if err := InitCustomRolesTable(); err != nil {
		t.Fatalf("InitCustomRolesTable() error = %v", err)
	}

	_, err = GetCustomRole("nonexistent")
	if err == nil {
		t.Error("GetCustomRole should error for non-existent role")
	}
}

func TestCustomRoles_DeleteNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-roles-delnf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := Init(filepath.Join(tmpDir, "delnf.db")); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() { _ = Close() }()

	if err := InitCustomRolesTable(); err != nil {
		t.Fatalf("InitCustomRolesTable() error = %v", err)
	}

	err = DeleteCustomRole("nonexistent")
	if err == nil {
		t.Error("DeleteCustomRole should error for non-existent role")
	}
	if err.Error() != "custom role not found: nonexistent" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestCustomRoles_ListEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-roles-empty-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := Init(filepath.Join(tmpDir, "empty.db")); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() { _ = Close() }()

	if err := InitCustomRolesTable(); err != nil {
		t.Fatalf("InitCustomRolesTable() error = %v", err)
	}

	roles, err := ListCustomRoles()
	if err != nil {
		t.Fatalf("ListCustomRoles failed: %v", err)
	}
	// Empty table returns nil slice, not empty slice
	if len(roles) != 0 {
		t.Errorf("expected nil or empty roles, got %d", len(roles))
	}
}

func TestCustomRoles_MultipleRoles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k13d-roles-multi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := Init(filepath.Join(tmpDir, "multi.db")); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer func() { _ = Close() }()

	if err := InitCustomRolesTable(); err != nil {
		t.Fatalf("InitCustomRolesTable() error = %v", err)
	}

	// Save multiple roles
	roleDefs := map[string]string{
		"alpha":   `{"name":"alpha"}`,
		"beta":    `{"name":"beta"}`,
		"charlie": `{"name":"charlie"}`,
	}
	for name, def := range roleDefs {
		if err := SaveCustomRole(name, def); err != nil {
			t.Fatalf("SaveCustomRole(%s) failed: %v", name, err)
		}
	}

	// List should return all, ordered by name
	roles, err := ListCustomRoles()
	if err != nil {
		t.Fatalf("ListCustomRoles failed: %v", err)
	}
	if len(roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(roles))
	}

	// Verify ordering (ORDER BY name)
	if roles[0].Name != "alpha" {
		t.Errorf("expected first role 'alpha', got %s", roles[0].Name)
	}
	if roles[1].Name != "beta" {
		t.Errorf("expected second role 'beta', got %s", roles[1].Name)
	}
	if roles[2].Name != "charlie" {
		t.Errorf("expected third role 'charlie', got %s", roles[2].Name)
	}
}

func TestCustomRoles_NilDB(t *testing.T) {
	savedDB := DB
	DB = nil
	defer func() { DB = savedDB }()

	// All operations should return errors
	err := SaveCustomRole("test", "{}")
	if err == nil {
		t.Error("SaveCustomRole should error with nil DB")
	}

	_, err = GetCustomRole("test")
	if err == nil {
		t.Error("GetCustomRole should error with nil DB")
	}

	_, err = ListCustomRoles()
	if err == nil {
		t.Error("ListCustomRoles should error with nil DB")
	}

	err = DeleteCustomRole("test")
	if err == nil {
		t.Error("DeleteCustomRole should error with nil DB")
	}
}

func TestCustomRoleRow_Fields(t *testing.T) {
	row := CustomRoleRow{
		Name:       "test-role",
		Definition: `{"name":"test-role","features":["dashboard"]}`,
	}

	if row.Name != "test-role" {
		t.Errorf("expected name 'test-role', got %s", row.Name)
	}
	if row.Definition == "" {
		t.Error("expected non-empty definition")
	}
}
