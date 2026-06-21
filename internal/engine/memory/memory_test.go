package memory_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
	eng "github.com/eduardoworrel/worrel-agent-cockpit/internal/engine"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/engine/memory"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

type fakeLLM struct{ out string }

func (f fakeLLM) RunHeadless(_ context.Context, _ string, _ adapter.HeadlessOpts) (string, error) {
	return f.out, nil
}

func seedFriction(t *testing.T, st *store.Store, sessID string) {
	_ = st.AppendTranscriptEventRich(sessID, "assistant", "tool_use", "Bash make build", `[{"type":"tool_use","name":"Bash"}]`, 0, 0)
	_ = st.AppendTranscriptEventRich(sessID, "user", "tool_result", "make: not found", `[{"type":"tool_result","output":"make: not found","is_error":true}]`, 0, 0)
	_ = st.AppendTranscriptEventRich(sessID, "assistant", "tool_use", "Bash go build ./...", `[{"type":"tool_use","name":"Bash"}]`, 0, 0)
}

func TestMemoryEngineHybridCreatesSuggestion(t *testing.T) {
	st, _ := store.Open(filepath.Join(t.TempDir(), "t.db"))
	p, _ := st.CreateProject("App", "")
	sess, _ := st.CreateSession(&store.Session{ProjectID: p.ID, Adapter: "claude-code", Mode: "wrapper"})
	seedFriction(t, st, sess.ID)

	llm := fakeLLM{out: `[{"content":"build é go build ./...","category":"convencao","evidence":"s1"}]`}
	m := memory.New(llm)

	// modo default hybrid (config vazia → Defaults resolve)
	r := eng.NewRegistry()
	r.Register(m)
	if err := r.Run(context.Background(), st, "memory", p.ID, sess.ID); err != nil {
		t.Fatal(err)
	}
	sgs, _ := st.ListSuggestions("", "")
	if len(sgs) != 1 {
		t.Fatalf("esperava 1 sugestão, got %d", len(sgs))
	}
	if sgs[0].Type != "add_memory_entry" || sgs[0].Origin != "engine:memory" {
		t.Fatalf("type=%q origin=%q", sgs[0].Type, sgs[0].Origin)
	}
	var pl memory.GoldenTruth
	_ = json.Unmarshal([]byte(sgs[0].Payload), &pl)
	if pl.Category != "convencao" {
		t.Fatalf("payload=%+v", pl)
	}
}

func TestMemoryEngineHeuristicOnlyNoLLM(t *testing.T) {
	st, _ := store.Open(filepath.Join(t.TempDir(), "t.db"))
	p, _ := st.CreateProject("App", "")
	sess, _ := st.CreateSession(&store.Session{ProjectID: p.ID, Adapter: "claude-code", Mode: "wrapper"})
	seedFriction(t, st, sess.ID)

	// LLM que explode se for chamado — heuristic_only não deve chamar
	m := memory.New(panicLLM{})
	_ = st.SetEngineConfig("memory", "detection_mode", "heuristic_only", "")

	r := eng.NewRegistry()
	r.Register(m)
	if err := r.Run(context.Background(), st, "memory", p.ID, sess.ID); err != nil {
		t.Fatal(err)
	}
	sgs, _ := st.ListSuggestions("", "")
	if len(sgs) == 0 {
		t.Fatal("heuristic_only deveria gerar ao menos 1 sugestão crua")
	}
}

type panicLLM struct{}

func (panicLLM) RunHeadless(_ context.Context, _ string, _ adapter.HeadlessOpts) (string, error) {
	panic("LLM não deveria ser chamado em heuristic_only")
}
