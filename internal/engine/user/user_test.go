package user

import (
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

func TestSimilarHeuristic_SameCanonicalKey(t *testing.T) {
	a := &Intent{Summary: "criar rota GET /users", Category: "criacao", Action: "criar", Object: "rota"}
	b := &Intent{Summary: "criar uma rota para listar usuários", Category: "criacao", Action: "criar", Object: "rota"}
	if !SimilarHeuristic(a, b) {
		t.Fatalf("deveriam ser similares (mesma chave): %q vs %q", a.Key(), b.Key())
	}
}

func TestSimilarHeuristic_DiffCategory(t *testing.T) {
	a := &Intent{Category: "debug", Action: "corrigir", Object: "erro"}
	b := &Intent{Category: "criacao", Action: "criar", Object: "rota"}
	if SimilarHeuristic(a, b) {
		t.Fatal("chaves diferentes não são similares")
	}
}

func TestKeyIsCanonical(t *testing.T) {
	a := &Intent{Category: "criacao", Action: "Criar", Object: "Rota GET /users"}
	if got := a.Key(); got != "criacao|criar|rota get users" {
		t.Fatalf("chave canônica inesperada: %q", got)
	}
}

func TestCorrectionMarkerMatchesRealMessage(t *testing.T) {
	// mensagem exata do fixture do plano — o regex antigo falhava aqui
	evs := []*store.TranscriptEvent{
		{Role: "assistant", Kind: "text", Content: "aqui está o código", Seq: 1},
		{Role: "user", Kind: "text", Content: "não é isso, eu quero com validação", Seq: 2},
	}
	got := DetectCorrection(evs)
	if len(got) != 1 || got[0].Signal != "user_correction" {
		t.Fatalf("esperava 1 correção, obtive %+v", got)
	}
}

func TestCancellationDoesNotFalsePositiveOnPara(t *testing.T) {
	evs := []*store.TranscriptEvent{
		{Role: "user", Kind: "text", Content: "cria uma rota para listar usuários", Seq: 1},
	}
	if got := DetectCancellation("", evs); len(got) != 0 {
		t.Fatalf("'para' preposição não deve gerar cancelamento: %+v", got)
	}
}

func TestCancellationFromEndReasonSignal(t *testing.T) {
	// valor real produzido por wrapper.exitReason quando morto por sinal
	if got := DetectCancellation("CLI morto por sinal interrupt", nil); len(got) != 1 {
		t.Fatalf("esperava cancelamento por end_reason, obtive %+v", got)
	}
	if got := DetectCancellation("CLI saiu com código 130 — ^C", nil); len(got) != 1 {
		t.Fatalf("esperava cancelamento por exit 130, obtive %+v", got)
	}
}

func TestCancellationFromUserText(t *testing.T) {
	evs := []*store.TranscriptEvent{{Role: "user", Kind: "text", Content: "cancela isso, chega", Seq: 1}}
	if got := DetectCancellation("", evs); len(got) != 1 {
		t.Fatalf("esperava cancelamento por texto, obtive %+v", got)
	}
}

func TestFrustrationNotTriggeredByFactualError(t *testing.T) {
	evs := []*store.TranscriptEvent{
		{Role: "user", Kind: "text", Content: "o teste não funciona, deu erro 500", Seq: 1},
	}
	if got := DetectFrustration(evs); len(got) != 0 {
		t.Fatalf("relato factual de bug não é frustração: %+v", got)
	}
	evs2 := []*store.TranscriptEvent{
		{Role: "user", Kind: "text", Content: "que droga, isso é péssimo", Seq: 1},
	}
	if got := DetectFrustration(evs2); len(got) != 1 {
		t.Fatalf("esperava frustração explícita, obtive %+v", got)
	}
}

func TestRepetitionGroupsBySameKey(t *testing.T) {
	evs := []*store.TranscriptEvent{
		{Role: "user", Kind: "text", Content: "cria a rota de usuários", Seq: 1},
		{Role: "assistant", Kind: "text", Content: "feito", Seq: 2},
		{Role: "user", Kind: "text", Content: "cria a rota para listar usuários", Seq: 3},
	}
	intents := []*Intent{
		{Seq: 1, Category: "criacao", Action: "criar", Object: "rota"},
		{Seq: 3, Category: "criacao", Action: "criar", Object: "rota"},
	}
	got := DetectRepetition(evs, intents)
	if len(got) != 1 || got[0].Signal != "user_repetition" || len(got[0].Events) != 2 {
		t.Fatalf("esperava 1 janela de repetição com 2 eventos, obtive %+v", got)
	}
}

func TestRepetitionNoIntentsNoWindows(t *testing.T) {
	// modo heuristic_only: sem intents (extração é LLM) -> nada
	if got := DetectRepetition(nil, nil); len(got) != 0 {
		t.Fatalf("sem intents não deve emitir repetição: %+v", got)
	}
}
