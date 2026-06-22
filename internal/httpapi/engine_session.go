package httpapi

import (
	"context"
	"net/http"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/bus"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

func (s *Server) routesEngineSessions() {
	s.mux.HandleFunc("POST /api/sessions/engine", s.handleCreateEngineSession)
}

// handleCreateEngineSession cria uma sessão dirigida pelo MOTOR (stream-json):
// sem PTY, sem hook, sem ask. A Home a vê como uma sessão viva e interage por
// ela via o canal AG-UI (snapshot/respond/prompt).
func (s *Server) handleCreateEngineSession(w http.ResponseWriter, r *http.Request) {
	if s.deps.Engine == nil {
		writeErr(w, 503, "motor indisponível")
		return
	}
	in, _ := decode[struct {
		ProjectID string `json:"project_id"`
		Mode      string `json:"mode"` // "" = acceptEdits (auto-mode)
	}](r)

	sess, err := s.deps.Store.CreateSession(&store.Session{
		ProjectID: in.ProjectID,
		Adapter:   "engine", // marca: dirigida pelo motor stream-json
		Mode:      "wrapper", // entra na faixa de sessões vivas da Home
	})
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	// cwd: workspace efêmero do worrel (headless → sem trust prompt).
	cwd, err := s.deps.Workspace.ScratchWorkspace(sess.ID)
	if err != nil {
		_ = s.deps.Store.EndSession(sess.ID)
		writeErr(w, 500, err.Error())
		return
	}
	_ = s.deps.Store.SetSessionWorkspaceDir(sess.ID, cwd)

	if err := s.deps.Engine.Start(context.Background(), sess.ID, cwd, in.Mode); err != nil {
		_ = s.deps.Store.EndSession(sess.ID)
		writeErr(w, 500, err.Error())
		return
	}
	s.deps.Bus.Publish(bus.Event{Type: "session.started", Payload: map[string]any{"id": sess.ID, "project_id": sess.ProjectID}})
	fresh, _ := s.deps.Store.GetSession(sess.ID)
	writeJSON(w, 201, fresh)
}
