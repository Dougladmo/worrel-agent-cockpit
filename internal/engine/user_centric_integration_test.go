package engine_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
	eng "github.com/eduardoworrel/worrel-agent-cockpit/internal/engine"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/engine/memory"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/engine/skill"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// routingLLM devolve intents quando o prompt é de extração de intenção, e um
// rascunho de skill quando é o distiller de skill. Assim um único fake serve aos
// dois caminhos que rodam na mesma execução (extração + destilação).
type routingLLM struct{ intents string }

func (f routingLLM) RunHeadless(_ context.Context, prompt string, _ adapter.HeadlessOpts) (string, error) {
	if strings.Contains(prompt, "Extraia a INTENÇÃO") {
		return f.intents, nil
	}
	// distiller de skill: um rascunho por janela
	return `[{"signature":"llm","title":"Fluxo de rota","skill_draft":{"name":"rota","content":"passos","structured":"{}"},"agente_draft":{"name":"rota","persona":"p"}}]`, nil
}

// Correção do usuário (heurística pura, sem LLM) vira sugestão de memória.
func TestUserCentric_CorrectionProducesMemory(t *testing.T) {
	st, _ := store.Open(filepath.Join(t.TempDir(), "t.db"))
	defer st.Close()
	p, _ := st.CreateProject("App", "")
	sess, _ := st.CreateSession(&store.Session{ProjectID: p.ID, Adapter: "claude-code", Mode: "wrapper"})
	_ = st.AppendTranscriptEvent(sess.ID, "user", "text", "cria uma rota GET /users", 0, 0)
	_ = st.AppendTranscriptEvent(sess.ID, "assistant", "text", "aqui está o código", 0, 0)
	_ = st.AppendTranscriptEvent(sess.ID, "user", "text", "não é isso, eu quero com validação", 0, 0)

	// heuristic_only: o LLM NÃO pode ser chamado (correção é heurística)
	_ = st.SetEngineConfig("memory", "detection_mode", "heuristic_only", "")
	_ = st.SetEngineConfig("memory", "user_signals", "on", "")

	m := memory.New(explodeLLM{t})
	r := eng.NewRegistry()
	r.Register(m)
	if err := r.Run(context.Background(), st, "memory", p.ID, sess.ID); err != nil {
		t.Fatal(err)
	}
	sgs, _ := st.ListSuggestions("", "")
	var found bool
	for _, s := range sgs {
		if s.Type == "add_memory_entry" && strings.Contains(s.Title, "corrigiu") {
			found = true
		}
	}
	if !found {
		t.Fatalf("esperava sugestão de memória a partir da correção; got %+v", sgs)
	}
}

// Assinatura por intenção acumula cross-session: a mesma intenção canônica em
// duas sessões distintas eleva occurrences e matura o candidato.
func TestUserCentric_IntentSignatureAccumulatesCrossSession(t *testing.T) {
	st, _ := store.Open(filepath.Join(t.TempDir(), "t.db"))
	defer st.Close()
	p, _ := st.CreateProject("App", "")

	_ = st.SetEngineConfig("skill", "detection_mode", "hybrid", "")
	_ = st.SetEngineConfig("skill", "signature_mode", "intent_summary", "")
	_ = st.SetEngineConfig("skill", "maturation_threshold", "2", "")

	// intents com a MESMA chave canônica (criacao|criar|rota) em ambas as sessões,
	// mas texto de summary diferente — prova que não é hash de texto livre.
	run := func(userMsg, intentJSON string) {
		sess, _ := st.CreateSession(&store.Session{ProjectID: p.ID, Adapter: "claude-code", Mode: "wrapper"})
		_ = st.AppendTranscriptEvent(sess.ID, "user", "text", "primeiro "+userMsg+" depois testa", 0, 0)
		_ = st.AppendTranscriptEventRich(sess.ID, "assistant", "tool_use", "Write", `[{"type":"tool_use","name":"Write"}]`, 0, 0)
		_ = st.AppendTranscriptEventRich(sess.ID, "assistant", "tool_use", "Bash", `[{"type":"tool_use","name":"Bash"}]`, 0, 0)
		s := skill.New(routingLLM{intents: intentJSON})
		r := eng.NewRegistry()
		r.Register(s)
		if err := r.Run(context.Background(), st, "skill", p.ID, sess.ID); err != nil {
			t.Fatal(err)
		}
	}
	// seq da mensagem de usuário é 1 em ambas as sessões
	run("cria a rota de usuários", `[{"seq":1,"summary":"criar rota de usuarios","category":"criacao","action":"criar","object":"rota"}]`)
	run("cria uma rota para listar usuários", `[{"seq":1,"summary":"criar endpoint que lista usuarios","category":"criacao","action":"criar","object":"rota"}]`)

	cands, _ := st.ListSkillCandidates(p.ID, "")
	var intentCand *store.SkillCandidate
	for _, c := range cands {
		if strings.HasPrefix(c.Signature, "intent:") {
			intentCand = c
		}
	}
	if intentCand == nil {
		t.Fatalf("esperava candidato com assinatura intent:; got %+v", cands)
	}
	if intentCand.Occurrences < 2 {
		t.Fatalf("esperava acúmulo cross-session (>=2 occurrences), got %d (sig=%s)", intentCand.Occurrences, intentCand.Signature)
	}
	if intentCand.Signature != "intent:criacao|criar|rota" {
		t.Fatalf("assinatura canônica inesperada: %s", intentCand.Signature)
	}
}

type explodeLLM struct{ t *testing.T }

func (e explodeLLM) RunHeadless(_ context.Context, _ string, _ adapter.HeadlessOpts) (string, error) {
	e.t.Fatal("LLM não deveria ser chamado no modo heuristic_only")
	return "", nil
}
