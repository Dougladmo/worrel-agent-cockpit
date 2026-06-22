package httpapi

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/apply"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/bus"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/mirror"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

func TestAcceptAsAgenteEndpoint(t *testing.T) {
	st, _ := store.Open(filepath.Join(t.TempDir(), "t.db"))
	p, _ := st.CreateProject("App", "")
	const pl = `{"title":"D","signature":"s","skill_draft":{"name":"D","content":"c","structured":"{}"},"agente_draft":{"name":"Dr","persona":"p"}}`
	sg, _ := st.CreateSuggestion(&store.Suggestion{ProjectID: p.ID, Type: "skill_or_agente_candidate", Title: "D", Payload: pl})
	srv := New(Deps{Store: st, Applier: apply.New(st, mirror.New(t.TempDir()), bus.New())})

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/suggestions/"+sg.ID+"/accept?as=agente", nil))
	if rec.Code != 200 {
		t.Fatalf("accept: %d %s", rec.Code, rec.Body.String())
	}
	if ags, _ := st.ListAgents(p.ID); len(ags) != 1 {
		t.Fatalf("agents=%d", len(ags))
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/projects/"+p.ID+"/agents", nil))
	if rec.Code != 200 {
		t.Fatalf("list agents: %d", rec.Code)
	}
}
