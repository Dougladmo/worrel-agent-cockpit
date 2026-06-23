package streamengine

import "testing"

func assistantEvent(model, text string) map[string]any {
	return map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model":   model,
			"content": []any{map[string]any{"type": "text", "text": text}},
		},
	}
}

// TestSyntheticOutputBecomesCommand prova que a saída de um slash command local
// (assistant de model "<synthetic>") vira uma linha de role "command" e NÃO é
// tomada como última fala do agente — é o que faz a UI enquadrá-la num card.
func TestSyntheticOutputBecomesCommand(t *testing.T) {
	var persisted []string
	s := &Session{persist: func(role, _ string) { persisted = append(persisted, role) }}

	s.handle(assistantEvent("<synthetic>", "Current session: 43% used"))

	snap := s.Snapshot()
	if len(snap.History) != 1 || snap.History[0].Role != "command" {
		t.Fatalf("history = %+v, quero uma linha de role command", snap.History)
	}
	if snap.Message != "" {
		t.Fatalf("Message = %q, saída de comando não deve virar última fala", snap.Message)
	}
	if len(persisted) != 1 || persisted[0] != "command" {
		t.Fatalf("persisted roles = %v, quero [command]", persisted)
	}
}

// TestRealAssistantTextStaysAI garante que texto de model real continua sendo
// fala do agente (role "ai" e última fala), sem regressão.
func TestRealAssistantTextStaysAI(t *testing.T) {
	s := &Session{}

	s.handle(assistantEvent("claude-opus-4-8", "claro, já faço isso"))

	snap := s.Snapshot()
	if len(snap.History) != 1 || snap.History[0].Role != "ai" {
		t.Fatalf("history = %+v, quero uma linha de role ai", snap.History)
	}
	if snap.Message != "claro, já faço isso" {
		t.Fatalf("Message = %q, fala real do agente deveria virar última fala", snap.Message)
	}
}
