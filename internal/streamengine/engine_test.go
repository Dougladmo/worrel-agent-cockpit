package streamengine

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/agui"
)

// newTestSession creates a Session wired to a pipe so write() doesn't block.
func newTestSession() *Session {
	r, w := io.Pipe()
	go io.Copy(io.Discard, r)
	return &Session{
		id:     "s1",
		state:  agui.StateAwaiting,
		message: "Oi! Como posso ajudar?",
		stdin:  json.NewEncoder(w),
		stdinW: w,
		history: []agui.HistoryLine{
			{Role: "you", Text: "ola"},
			{Role: "ai", Text: "Oi! Como posso ajudar?"},
		},
	}
}

func TestSendPromptClearsMessage(t *testing.T) {
	s := newTestSession()
	if err := s.SendPrompt("faça algo"); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if snap.Message != "" {
		t.Fatalf("SendPrompt deveria limpar Message, mas ficou %q", snap.Message)
	}
	if snap.State != agui.StateWorking {
		t.Fatalf("State = %q, quer working", snap.State)
	}
	// histórico preservou o user message
	if len(snap.History) != 3 || snap.History[2].Role != "you" || !strings.Contains(snap.History[2].Text, "faça algo") {
		t.Fatalf("history inesperado: %+v", snap.History)
	}
}

func TestToolOnlyTurnDoesNotLeakPreviousMessage(t *testing.T) {
	s := newTestSession()
	if err := s.SendPrompt("execute a tarefa"); err != nil {
		t.Fatal(err)
	}

	// assistant com tool_use apenas (sem texto)
	s.handle(map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"content": []any{
				map[string]any{"type": "tool_use", "name": "Read", "input": map[string]any{"file_path": "x.go"}},
				map[string]any{"type": "tool_use", "name": "Edit", "input": map[string]any{"file_path": "x.go", "old": "a", "new": "b"}},
			},
		},
	})

	snap := s.Snapshot()
	if snap.Message != "" {
		t.Fatalf("tool_use-only turn: Message deveria ser vazia, mas ficou %q (vazou do turno anterior)", snap.Message)
	}
	if len(snap.ToolCalls) != 2 {
		t.Fatalf("ToolCalls = %d, quer 2", len(snap.ToolCalls))
	}
	if snap.State != agui.StateWorking {
		t.Fatalf("State = %q, quer working (ainda nao veio result)", snap.State)
	}

	// result → awaiting
	s.handle(map[string]any{"type": "result"})
	snap = s.Snapshot()
	if snap.State != agui.StateAwaiting {
		t.Fatalf("State = %q, quer awaiting apos result", snap.State)
	}
	if snap.Message != "" {
		t.Fatalf("tool_use-only turn: Message deveria continuar vazia apos result, mas ficou %q", snap.Message)
	}
}

func TestMessageOverwrittenOnNewAssistantText(t *testing.T) {
	s := newTestSession()
	if err := s.SendPrompt("conte uma historia"); err != nil {
		t.Fatal(err)
	}

	// primeiro assistant com texto
	s.handle(map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"content": []any{
				map[string]any{"type": "text", "text": "Era uma vez…"},
			},
		},
	})
	snap := s.Snapshot()
	if snap.Message != "Era uma vez…" {
		t.Fatalf("Message = %q, quer 'Era uma vez…'", snap.Message)
	}

	// tool_use (nao altera message)
	s.handle(map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"content": []any{
				map[string]any{"type": "tool_use", "name": "Bash", "input": map[string]any{"command": "ls"}},
			},
		},
	})
	snap = s.Snapshot()
	if snap.Message != "Era uma vez…" {
		t.Fatalf("tool_use nao deve alterar Message: %q", snap.Message)
	}

	// segundo texto: deve SOBRESCREVER (nao append)
	s.handle(map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"content": []any{
				map[string]any{"type": "text", "text": "…e assim termina a historia."},
			},
		},
	})
	snap = s.Snapshot()
	if snap.Message != "…e assim termina a historia." {
		t.Fatalf("Message deveria ser o ultimo texto, mas ficou %q", snap.Message)
	}

	// result
	s.handle(map[string]any{"type": "result"})
	snap = s.Snapshot()
	if snap.Message != "…e assim termina a historia." {
		t.Fatalf("Message apos result = %q, quer o ultimo texto", snap.Message)
	}
}
