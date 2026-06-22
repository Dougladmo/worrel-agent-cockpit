package mcpserver

import (
	"strings"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

func TestRecallMemoryFilters(t *testing.T) {
	entries := []*store.MemoryEntry{
		{Category: "convencao", Content: "build é go build ./..."},
		{Category: "arquitetura", Content: "config fica em internal/x"},
		{Category: "gotcha", Content: "make não existe; use go"},
	}
	if got := recallMemory(entries, "", ""); len(got) != 3 {
		t.Fatalf("sem filtro: %d", len(got))
	}
	if got := recallMemory(entries, "arquitetura", ""); len(got) != 1 || got[0]["content"] != "config fica em internal/x" {
		t.Fatalf("por categoria: %+v", got)
	}
	if got := recallMemory(entries, "", "make"); len(got) != 1 || !strings.Contains(got[0]["content"], "make") {
		t.Fatalf("por query: %+v", got)
	}
	if got := recallMemory(entries, "", "inexistente-xyz"); len(got) != 0 {
		t.Fatalf("query sem match: %+v", got)
	}
}

func TestGetMemoryToolReturnsEntries(t *testing.T) {
	svc, s, _ := setup(t)
	p, _ := s.CreateProject("App", "")
	_, _ = s.CreateMemoryEntry(&store.MemoryEntry{ProjectID: p.ID, Content: "use go build", Category: "convencao"})
	tok := "tok-recall"
	_, _ = s.CreateSession(&store.Session{ProjectID: p.ID, Adapter: "claude-code", Mode: "wrapper", MCPToken: &tok})

	cs := connect(t, svc, tok)
	out := callText(t, cs, "get_memory", map[string]any{})
	if !strings.Contains(out, "use go build") || !strings.Contains(out, "convencao") {
		t.Fatalf("get_memory: %s", out)
	}
}

func TestLoadAgentTool(t *testing.T) {
	svc, s, _ := setup(t)
	p, _ := s.CreateProject("App", "")
	ag, _ := s.CreateAgent(p.ID, "Revisor", "Você é um revisor Go rigoroso.", "")
	tok := "tok-agent"
	_, _ = s.CreateSession(&store.Session{ProjectID: p.ID, Adapter: "claude-code", Mode: "wrapper", MCPToken: &tok})

	cs := connect(t, svc, tok)
	out := callText(t, cs, "load_agent", map[string]any{"agent_id": ag.ID})
	if !strings.Contains(out, "revisor Go") {
		t.Fatalf("load_agent: %s", out)
	}
}
