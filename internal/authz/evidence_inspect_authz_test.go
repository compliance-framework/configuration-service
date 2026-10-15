package authz

import (
	"slices"
	"testing"
)

func TestManifestDeclaresEvidenceInspect(t *testing.T) {
	m, err := DefaultManifest()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Resources[ResourceEvidence].Actions, ActionInspect) {
		t.Fatalf("evidence resource missing %q (have %v)", ActionInspect, m.Resources[ResourceEvidence].Actions)
	}
}

// Under Cedar only admin ("*": ["*"]) holds evidence:inspect; the roles that create or read
// evidence do not, and neither do agents.
func TestCedarEvidenceInspectMatrix(t *testing.T) {
	c := mustCedar(t, &RoleAssignments{
		Users: map[string]string{
			"admin@x":       "admin",
			"viewer@x":      "viewer",
			"auditor@x":     "auditor",
			"contributor@x": "contributor",
		},
	})
	user := func(id string) Subject { return Subject{Type: "user", ID: id} }
	cases := []struct {
		subj Subject
		want bool
	}{
		{user("admin@x"), true},
		{user("viewer@x"), false},
		{user("auditor@x"), false},
		{user("contributor@x"), false},
		{Subject{Type: "agent", ID: "agent-1"}, false},
		{Subject{Type: "anonymous"}, false},
	}
	for _, tc := range cases {
		if got := allows(t, c, tc.subj, ActionInspect, ResourceEvidence); got != tc.want {
			t.Errorf("%s %s: evidence:inspect allow = %v, want %v", tc.subj.Type, tc.subj.ID, got, tc.want)
		}
	}
}
