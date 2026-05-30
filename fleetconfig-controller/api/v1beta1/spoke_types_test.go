package v1beta1

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDerivedManagedClusterName(t *testing.T) {
	tests := []struct {
		name  string
		spoke *Spoke
		check func(t *testing.T, got string)
	}{
		{
			name: "short name + namespace",
			spoke: &Spoke{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns1"},
			},
			check: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, "foo-") {
					t.Errorf("want prefix foo-, got %q", got)
				}
				if len(got) != len("foo")+1+8 {
					t.Errorf("want len %d, got %d (%q)", len("foo")+9, len(got), got)
				}
			},
		},
		{
			name: "max-length name truncates to 54",
			spoke: &Spoke{
				ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 63), Namespace: "ns"},
			},
			check: func(t *testing.T, got string) {
				if len(got) != 63 {
					t.Errorf("want len 63, got %d (%q)", len(got), got)
				}
				if !strings.HasPrefix(got, strings.Repeat("a", 54)+"-") {
					t.Errorf("want truncated 54-a + '-' prefix, got %q", got)
				}
			},
		},
		{
			name: "hub-as-spoke gets a hashed name (no sentinel carve-out)",
			spoke: &Spoke{
				ObjectMeta: metav1.ObjectMeta{Name: ManagedClusterTypeHubAsSpoke, Namespace: "ns1"},
			},
			check: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, ManagedClusterTypeHubAsSpoke+"-") {
					t.Errorf("want prefix %q-, got %q", ManagedClusterTypeHubAsSpoke, got)
				}
				if got == ManagedClusterTypeHubAsSpoke {
					t.Errorf("expected hashed name, got bare sentinel %q", got)
				}
			},
		},
		{
			name: "two hub-as-spoke Spokes in different namespaces produce different names",
			spoke: &Spoke{
				ObjectMeta: metav1.ObjectMeta{Name: ManagedClusterTypeHubAsSpoke, Namespace: "ns2"},
			},
			check: func(t *testing.T, got string) {
				other := (&Spoke{ObjectMeta: metav1.ObjectMeta{Name: ManagedClusterTypeHubAsSpoke, Namespace: "ns1"}}).DerivedManagedClusterName()
				if got == other {
					t.Errorf("expected different derived names across namespaces, both got %q", got)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, tc.spoke.DerivedManagedClusterName())
		})
	}
}

// TestDerivedManagedClusterName_NamespaceDistinguishes confirms two Spokes with the same metadata.Name
// but different namespaces produce different derived names (the core collision-avoidance property).
func TestDerivedManagedClusterName_NamespaceDistinguishes(t *testing.T) {
	a := (&Spoke{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns1"}}).DerivedManagedClusterName()
	b := (&Spoke{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns2"}}).DerivedManagedClusterName()
	if a == b {
		t.Fatalf("derived names must differ across namespaces, both got %q", a)
	}
}

// TestDerivedManagedClusterName_Deterministic confirms the same input always produces the same output.
func TestDerivedManagedClusterName_Deterministic(t *testing.T) {
	s := &Spoke{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns1"}}
	first := s.DerivedManagedClusterName()
	for range 10 {
		if got := s.DerivedManagedClusterName(); got != first {
			t.Fatalf("non-deterministic: first=%q got=%q", first, got)
		}
	}
}
