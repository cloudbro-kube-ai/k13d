package ui

import (
	"reflect"
	"testing"
)

func TestResourceStore_SetFull_Snapshot(t *testing.T) {
	s := NewResourceStore()
	s.SetFull([]string{"NS", "NAME", "STATUS"}, []KeyedRow{
		{Key: "default/p1", Row: []string{"default", "p1", "Running"}},
		{Key: "default/p2", Row: []string{"default", "p2", "Pending"}},
	})

	headers, keys, rows := s.Snapshot()
	if !reflect.DeepEqual(headers, []string{"NS", "NAME", "STATUS"}) {
		t.Fatalf("headers = %v", headers)
	}
	if !reflect.DeepEqual(keys, []string{"default/p1", "default/p2"}) {
		t.Fatalf("keys = %v", keys)
	}
	if len(rows) != 2 || !reflect.DeepEqual(rows[0], []string{"default", "p1", "Running"}) {
		t.Fatalf("rows = %v", rows)
	}
	if s.Len() != 2 {
		t.Fatalf("Len = %d, want 2", s.Len())
	}
}

func TestResourceStore_SetFull_PreservesOrderAndDedups(t *testing.T) {
	s := NewResourceStore()
	// Duplicate key: second occurrence overwrites cells but must not duplicate entry.
	s.SetFull([]string{"H"}, []KeyedRow{
		{Key: "a", Row: []string{"1"}},
		{Key: "b", Row: []string{"2"}},
		{Key: "a", Row: []string{"1-updated"}},
	})

	if s.Len() != 2 {
		t.Fatalf("Len = %d, want 2 (dedup)", s.Len())
	}
	if got := s.RowByKey("a"); !reflect.DeepEqual(got, []string{"1-updated"}) {
		t.Fatalf("row a = %v, want overwritten value", got)
	}
	_, keys, _ := s.Snapshot()
	// Order is derived from first insertion of each key.
	if !reflect.DeepEqual(keys, []string{"a", "b"}) {
		t.Fatalf("keys order = %v", keys)
	}
}

func TestResourceStore_Upsert(t *testing.T) {
	s := NewResourceStore()

	if added := s.Upsert("ns/x", []string{"ns", "x", "v1"}); !added {
		t.Fatalf("first Upsert should report added=true")
	}
	if added := s.Upsert("ns/x", []string{"ns", "x", "v2"}); added {
		t.Fatalf("second Upsert should report added=false")
	}

	if got := s.RowByKey("ns/x"); !reflect.DeepEqual(got, []string{"ns", "x", "v2"}) {
		t.Fatalf("row = %v, want updated value", got)
	}
	if s.Len() != 1 {
		t.Fatalf("Len = %d, want 1", s.Len())
	}
}

func TestResourceStore_Delete(t *testing.T) {
	s := NewResourceStore()
	s.SetFull(nil, []KeyedRow{
		{Key: "a", Row: []string{"1"}},
		{Key: "b", Row: []string{"2"}},
		{Key: "c", Row: []string{"3"}},
	})

	if ok := s.Delete("b"); !ok {
		t.Fatalf("Delete existing key should return true")
	}
	if ok := s.Delete("missing"); ok {
		t.Fatalf("Delete missing key should return false")
	}

	_, keys, _ := s.Snapshot()
	if !reflect.DeepEqual(keys, []string{"a", "c"}) {
		t.Fatalf("keys after delete = %v, want [a c]", keys)
	}
	if s.Len() != 2 {
		t.Fatalf("Len = %d, want 2", s.Len())
	}
}

func TestResourceStore_IndexOf_KeyAt(t *testing.T) {
	s := NewResourceStore()
	s.SetFull(nil, []KeyedRow{
		{Key: "a", Row: nil},
		{Key: "b", Row: nil},
	})
	if s.IndexOf("a") != 0 || s.IndexOf("b") != 1 || s.IndexOf("missing") != -1 {
		t.Fatalf("IndexOf mismatch")
	}
	if s.KeyAt(0) != "a" || s.KeyAt(1) != "b" || s.KeyAt(5) != "" {
		t.Fatalf("KeyAt mismatch")
	}
}

func TestResourceStore_Snapshot_IsDefensiveCopy(t *testing.T) {
	s := NewResourceStore()
	s.SetFull([]string{"H"}, []KeyedRow{{Key: "a", Row: []string{"v"}}})

	_, _, rows := s.Snapshot()
	rows[0][0] = "MUTATED"

	// Mutating the snapshot must not affect store state.
	if got := s.RowByKey("a"); !reflect.DeepEqual(got, []string{"v"}) {
		t.Fatalf("store mutated via snapshot: %v", got)
	}
}

func TestResourceStore_Sort(t *testing.T) {
	s := NewResourceStore()
	s.SetFull(nil, []KeyedRow{
		{Key: "c", Row: []string{"3"}},
		{Key: "a", Row: []string{"1"}},
		{Key: "b", Row: []string{"2"}},
	})

	s.Sort(func(aKey, bKey string, aRow, bRow []string) bool {
		return aRow[0] < bRow[0]
	})

	_, keys, rows := s.Snapshot()
	if !reflect.DeepEqual(keys, []string{"a", "b", "c"}) {
		t.Fatalf("sorted keys = %v", keys)
	}
	if rows[0][0] != "1" {
		t.Fatalf("sorted rows[0] = %v", rows[0])
	}
}

func TestRowKey(t *testing.T) {
	cases := []struct {
		resource string
		row      []string
		want     string
	}{
		{"pods", []string{"default", "nginx", "Running"}, "default/nginx"},
		{"nodes", []string{"node-1", "Ready"}, "node-1"},
		{"namespaces", []string{"kube-system"}, "kube-system"},
		{"persistentvolumes", []string{"pv-1"}, "pv-1"},
		{"deployments", []string{"default", "web"}, "default/web"},
		{"customresourcedefinitions", []string{"crd-1"}, "crd-1"},
	}
	for _, c := range cases {
		if got := rowKey(c.resource, c.row); got != c.want {
			t.Errorf("rowKey(%q) = %q, want %q", c.resource, got, c.want)
		}
	}
}

func TestKeyedRows(t *testing.T) {
	rows := [][]string{
		{"default", "p1", "Running"},
		{"default", "p2", "Pending"},
	}
	got := keyedRows("pods", rows)
	want := []KeyedRow{
		{Key: "default/p1", Row: rows[0]},
		{Key: "default/p2", Row: rows[1]},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("keyedRows = %+v, want %+v", got, want)
	}
}
