package authz

import (
	"context"
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

// Builtin: evidence:inspect is an admin capability (users through the admin check, agents
// and anonymous denied) while the other evidence actions stay open to any authenticated
// caller. A user without an id is denied before any DB access, so a nil DB proves the admin
// path is taken.
func TestBuiltinEvidenceInspect(t *testing.T) {
	b := NewBuiltin(nil, nil, nil)
	ctx := context.Background()
	evaluate := func(s Subject, action string) bool {
		t.Helper()
		dec, err := b.Evaluate(ctx, s, action, Resource{Type: ResourceEvidence}, nil)
		if err != nil {
			t.Fatalf("%s %s: %v", s.Type, action, err)
		}
		return dec.Allow
	}

	agent := Subject{Type: "agent", ID: "agent-1"}
	if evaluate(agent, ActionInspect) {
		t.Error("agent evidence:inspect allowed, want deny")
	}
	if evaluate(Subject{Type: "anonymous"}, ActionInspect) {
		t.Error("anonymous evidence:inspect allowed, want deny")
	}
	if evaluate(Subject{Type: "user", ID: ""}, ActionInspect) {
		t.Error("user without id evidence:inspect allowed, want deny via the admin check")
	}
	for _, action := range []string{ActionRead, ActionCreate} {
		if !evaluate(agent, action) {
			t.Errorf("agent evidence:%s denied, want the unchanged default allow", action)
		}
	}
}
