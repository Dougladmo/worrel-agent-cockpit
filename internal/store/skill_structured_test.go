package store

import "testing"

func TestSkillStructuredRoundTrip(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("App", "")
	sk, _ := s.CreateSkill(p.ID, "Deploy", "## Passos\n- build\n- deploy")
	const js = `{"inputs":["env"],"steps":["build","deploy"],"edge_cases":[],"completion":"ok","own_memory":""}`
	if err := s.SetSkillStructured(sk.ID, js); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSkill(sk.ID)
	if err != nil || got.Structured != js {
		t.Fatalf("structured=%q err=%v", got.Structured, err)
	}
}
