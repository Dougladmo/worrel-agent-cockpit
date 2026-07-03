package skill

import (
	"context"

	eng "github.com/eduardoworrel/worrel-agent-cockpit/internal/engine"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/engine/user"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// signatureModeField declara o modo de assinatura do workflow. O default
// preserva o comportamento atual (hash das ferramentas executadas).
func signatureModeField() eng.ConfigField {
	return eng.ConfigField{Key: "signature_mode", Label: "Modo de assinatura", Type: "select", Default: "tool_use_hash", Options: []eng.ConfigOption{
		{Value: "tool_use_hash", Label: "Hash de ferramentas", Description: "Assina o workflow pelo que o agente executou (comportamento atual)."},
		{Value: "intent_summary", Label: "Intenção do usuário", Description: "Assina pela intenção canônica do que o usuário pediu (categoria|ação|objeto). Melhor recorrência cross-session. Usa LLM fora do modo só-heurística."},
	}}
}

// windowIntentKeys devolve, alinhado por índice às janelas, a CHAVE CANÔNICA da
// intenção da mensagem de usuário que LIDERA cada janela (o primeiro evento, que
// DetectWorkflows garante ser um user/text). Vazio quando não há intent.
//
// A chave é estável e canônica (categoria|ação|objeto) — nunca hash de texto
// livre —, então a mesma tarefa em sessões diferentes colapsa na mesma
// assinatura "intent:<chave>" e o candidato ACUMULA de verdade cross-session.
func (e *Engine) windowIntentKeys(ctx context.Context, rc eng.RunContext, windows []WorkflowWindow, events []*store.TranscriptEvent) []string {
	keys := make([]string, len(windows))
	if rc.Config["detection_mode"] == "heuristic_only" {
		return keys // sem LLM não há extração de intenção
	}
	hl, model := e.llm(rc.Config)
	intents, _ := user.ResolveIntents(ctx, hl, model, rc.Store, rc.ProjectID, rc.SessionID, events)
	keyBySeq := map[int64]string{}
	for _, it := range intents {
		keyBySeq[it.Seq] = it.Key()
	}
	for i, w := range windows {
		if len(w.Events) == 0 {
			continue
		}
		lead := w.Events[0]
		if lead.Role == "user" && lead.Kind == "text" {
			keys[i] = keyBySeq[lead.Seq]
		}
	}
	return keys
}
