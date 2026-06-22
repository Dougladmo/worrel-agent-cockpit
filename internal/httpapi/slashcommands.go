package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
)

// slashCommandsResponse é o shape do endpoint GET /api/adapters/{id}/slash-commands.
type slashCommandsResponse struct {
	Commands []adapter.SlashCommand `json:"commands"`
}

// routesSlashCommands registra GET /api/adapters/{id}/slash-commands.
//
// Contrato (espelha routesModels):
//   - 200 {"commands":[...]} quando o adapter existe. A lista vem de
//     ListSlashCommands() se o adapter implementa adapter.SlashCommandLister;
//     caso contrário, {"commands":[]}.
//   - 404 quando o id não existe no registry.
//
// Query opcional ?dir=<workingDir> inclui também os comandos de projeto
// (<dir>/.claude/commands). Sem dir, só os comandos de usuário (~/.claude).
func (s *Server) routesSlashCommands() {
	s.mux.HandleFunc("GET /api/adapters/{id}/slash-commands", func(w http.ResponseWriter, r *http.Request) {
		empty := slashCommandsResponse{Commands: []adapter.SlashCommand{}}
		if s.deps.Adapters == nil {
			writeJSON(w, 200, empty)
			return
		}
		a, ok := s.deps.Adapters.Get(r.PathValue("id"))
		if !ok {
			writeErr(w, 404, "adapter não encontrado")
			return
		}
		lister, ok := a.(adapter.SlashCommandLister)
		if !ok {
			writeJSON(w, 200, empty)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		cmds, err := lister.ListSlashCommands(ctx, r.URL.Query().Get("dir"))
		if cmds == nil {
			cmds = []adapter.SlashCommand{}
		}
		if err != nil {
			writeJSON(w, 502, slashCommandsResponse{Commands: cmds})
			return
		}
		writeJSON(w, 200, slashCommandsResponse{Commands: cmds})
	})
}
