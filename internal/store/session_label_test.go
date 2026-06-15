package store

import (
	"strings"
	"testing"
)

func TestSessionLabel(t *testing.T) {
	s, err := Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	p, _ := s.CreateProject("Cockpit", "")
	sess, _ := s.CreateSession(&Session{ProjectID: p.ID, Adapter: "claude-code", Mode: "wrapper"})
	if got := s.SessionLabel(sess.ID); got != "Cockpit" {
		t.Fatalf("label with project = %q, want Cockpit", got)
	}

	free, _ := s.CreateSession(&Session{Adapter: "claude-code", Mode: "wrapper"})
	if got := s.SessionLabel(free.ID); !strings.HasPrefix(got, "sessão ") {
		t.Fatalf("label without project = %q, want 'sessão ...'", got)
	}

	if got := s.SessionLabel("does-not-exist"); got != "sessão" {
		t.Fatalf("label unknown = %q, want 'sessão'", got)
	}
}
