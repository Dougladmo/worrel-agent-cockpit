package friction_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
	eng "github.com/eduardoworrel/worrel-agent-cockpit/internal/engine"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/engine/friction"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

type fakeLLM struct{ out string }

func (f fakeLLM) RunHeadless(_ context.Context, _ string, _ adapter.HeadlessOpts) (string, error) {
	return f.out, nil
}

func TestFrictionRoutesToMemory(t *testing.T) {
	st, _ := store.Open(filepath.Join(t.TempDir(), "t.db"))
	p, _ := st.CreateProject("App", "")
	sess, _ := st.CreateSession(&store.Session{ProjectID: p.ID, Adapter: "claude-code", Mode: "wrapper"})
	// transcript com erro→correção (sinal que memory.DetectFriction acha)
	_ = st.AppendTranscriptEventRich(sess.ID, "assistant", "tool_use", "Bash make build", `[{"type":"tool_use","name":"Bash"}]`, 0, 0)
	_ = st.AppendTranscriptEventRich(sess.ID, "user", "tool_result", "make: not found", `[{"type":"tool_result","output":"make: not found","is_error":true}]`, 0, 0)
	_ = st.AppendTranscriptEventRich(sess.ID, "assistant", "tool_use", "Bash go build", `[{"type":"tool_use","name":"Bash"}]`, 0, 0)

	llm := fakeLLM{out: `[{"destino":"memory","memory":{"content":"build é go build","category":"convencao"},"evidence":"e"}]`}
	m := friction.New(llm)
	r := eng.NewRegistry()
	r.Register(m)
	if err := r.Run(context.Background(), st, "friction", p.ID, sess.ID); err != nil {
		t.Fatal(err)
	}
	sgs, _ := st.ListSuggestions("", "")
	if len(sgs) != 1 || sgs[0].Type != "add_memory_entry" || sgs[0].Origin != "engine:friction" {
		t.Fatalf("sgs=%+v", sgs)
	}
}
