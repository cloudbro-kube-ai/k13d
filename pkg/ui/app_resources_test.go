package ui

import (
	"reflect"
	"testing"
)

func TestReorderNamespaceListByRecent_PrioritizesRecentAfterAll(t *testing.T) {
	allNamespaces := []string{"", "default", "kube-system", "monitoring", "staging"}
	recent := []string{"monitoring", "default"}

	got := reorderNamespaceListByRecent(allNamespaces, recent)
	want := []string{"", "monitoring", "default", "kube-system", "staging"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reorderNamespaceListByRecent() = %v, want %v", got, want)
	}
}

func TestAddRecentNamespace_TracksRecentWithoutMutatingBaseOrder(t *testing.T) {
	app := CreateMinimalTestApp()
	app.namespaces = []string{"", "default", "kube-system", "monitoring"}

	app.mx.Lock()
	app.addRecentNamespace("monitoring")
	gotNamespaces := append([]string(nil), app.namespaces...)
	gotRecent := append([]string(nil), app.recentNamespaces...)
	gotQuickSelect := reorderNamespaceListByRecent(gotNamespaces, gotRecent)
	app.mx.Unlock()

	wantNamespaces := []string{"", "default", "kube-system", "monitoring"}
	wantRecent := []string{"monitoring"}
	wantQuickSelect := []string{"", "monitoring", "default", "kube-system"}

	if !reflect.DeepEqual(gotNamespaces, wantNamespaces) {
		t.Fatalf("base namespaces after addRecentNamespace() = %v, want %v", gotNamespaces, wantNamespaces)
	}
	if !reflect.DeepEqual(gotRecent, wantRecent) {
		t.Fatalf("recentNamespaces after addRecentNamespace() = %v, want %v", gotRecent, wantRecent)
	}
	if !reflect.DeepEqual(gotQuickSelect, wantQuickSelect) {
		t.Fatalf("quick-select namespaces after addRecentNamespace() = %v, want %v", gotQuickSelect, wantQuickSelect)
	}
}

func TestAddRecentNamespace_PromotesExistingRecentNamespaceWithoutDuplicates(t *testing.T) {
	app := CreateMinimalTestApp()
	app.namespaces = []string{"", "default", "kube-system", "monitoring"}
	app.recentNamespaces = []string{"default", "monitoring"}

	app.mx.Lock()
	app.addRecentNamespace("monitoring")
	gotNamespaces := append([]string(nil), app.namespaces...)
	gotRecent := append([]string(nil), app.recentNamespaces...)
	gotQuickSelect := reorderNamespaceListByRecent(gotNamespaces, gotRecent)
	app.mx.Unlock()

	wantNamespaces := []string{"", "default", "kube-system", "monitoring"}
	wantRecent := []string{"monitoring", "default"}
	wantQuickSelect := []string{"", "monitoring", "default", "kube-system"}

	if !reflect.DeepEqual(gotNamespaces, wantNamespaces) {
		t.Fatalf("base namespaces after promoting recent namespace = %v, want %v", gotNamespaces, wantNamespaces)
	}
	if !reflect.DeepEqual(gotRecent, wantRecent) {
		t.Fatalf("recentNamespaces after promoting recent namespace = %v, want %v", gotRecent, wantRecent)
	}
	if !reflect.DeepEqual(gotQuickSelect, wantQuickSelect) {
		t.Fatalf("quick-select namespaces after promoting recent namespace = %v, want %v", gotQuickSelect, wantQuickSelect)
	}
}
