package natsx

import "testing"

func TestDomainStreamsCoverArchitectureSubjects(t *testing.T) {
	want := map[string]string{
		"AUTH":      "auth.>",
		"CATALOG":   "catalog.>",
		"GEO":       "geo.>",
		"ORDERS":    "order.>",
		"INVENTORY": "inventory.>",
		"BILLING":   "billing.>",
	}
	got := DomainStreams()
	if len(got) != len(want) {
		t.Fatalf("DomainStreams len=%d want %d", len(got), len(want))
	}
	for _, s := range got {
		subj, ok := want[s.Name]
		if !ok {
			t.Fatalf("unexpected stream %q", s.Name)
		}
		if len(s.Subjects) != 1 || s.Subjects[0] != subj {
			t.Fatalf("stream %s subjects=%v want [%s]", s.Name, s.Subjects, subj)
		}
	}
}
