package httpapi

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/agui"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/bus"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// Regressão: sessões do motor pararam de ganhar nome. O título "vivo" só era
// gerado no GET /interaction (ramo do motor vivo + History>=2); quando o front
// não consultava no instante certo, a sessão ficava "sem nome". O gatilho certo
// é o FIM DE TURNO (o agente parou de responder): deve gerar o título
// server-side, a partir do histórico, sem depender de poll do navegador.
func TestTitleOnTurnEnd_GeneratesAndStores(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	_ = st.SetEngineConfig("summary", "__enabled", "true", "")
	sess, err := st.CreateSession(&store.Session{Adapter: "engine", Mode: "wrapper"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	srv := New(Deps{
		Bus:        bus.New(),
		Store:      st,
		Summarizer: &fakeHeadless{out: `{"title":"Zipando skills","lines":["fez X"]}`},
	})

	history := []agui.HistoryLine{
		{Role: "you", Text: "zipa as skills pra mim"},
		{Role: "ai", Text: "pronto, zipei"},
	}
	srv.TitleOnTurnEnd(sess.ID, history)

	var title string
	for i := 0; i < 100; i++ {
		if s, _ := st.GetSession(sess.ID); s != nil {
			title = s.Title
		}
		if title != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if title != "Zipando skills" {
		t.Fatalf("título esperado 'Zipando skills', veio %q", title)
	}
}

// Custo: fim de turno pode notificar várias vezes com o MESMO histórico. Não
// deve regenerar o título (nem chamar o LLM) quando o histórico não avançou.
func TestTitleOnTurnEnd_SkipsRegenForSameHistory(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	_ = st.SetEngineConfig("summary", "__enabled", "true", "")
	sess, _ := st.CreateSession(&store.Session{Adapter: "engine", Mode: "wrapper"})

	fake := &fakeHeadless{out: `{"title":"Zipando skills","lines":["fez X"]}`}
	srv := New(Deps{Bus: bus.New(), Store: st, Summarizer: fake})

	history := []agui.HistoryLine{
		{Role: "you", Text: "zipa as skills pra mim"},
		{Role: "ai", Text: "pronto, zipei"},
	}
	srv.TitleOnTurnEnd(sess.ID, history)
	// espera a 1ª geração concluir (título gravado).
	for i := 0; i < 100; i++ {
		if s, _ := st.GetSession(sess.ID); s != nil && s.Title != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	srv.TitleOnTurnEnd(sess.ID, history) // mesmo histórico → não deve regerar
	time.Sleep(50 * time.Millisecond)

	if c := atomic.LoadInt32(&fake.calls); c != 1 {
		t.Fatalf("esperava 1 chamada ao LLM (histórico igual), veio %d", c)
	}
}
