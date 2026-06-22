package wrapper

import (
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/workspace"
)

// BuildSpawnOpts monta as opções de spawn a partir do estado persistido:
// gera/persiste o token MCP, monta primer (memória + skill opcional) e a URL.
// O cwd é resolvido pelo workspace gerenciado: sess.WorkspaceDir se preenchido,
// senão o workspace do projeto (via wm.SyncProject), senão um scratch por sessão.
func BuildSpawnOpts(st *store.Store, wm *workspace.Manager, sessionID string, port int, skillContent, agentPersona string) (adapter.SpawnOpts, error) {
	sess, err := st.GetSession(sessionID)
	if err != nil {
		return adapter.SpawnOpts{}, err
	}

	workdir := sess.WorkspaceDir
	primer := ""
	if sess.ProjectID != "" {
		proj, err := st.GetProject(sess.ProjectID)
		if err != nil {
			return adapter.SpawnOpts{}, err
		}
		// cwd = workspace gerenciado do escopo, com symlinks p/ as pastas reais
		ws, err := wm.SyncProject(proj.Slug, proj.Dirs)
		if err != nil {
			return adapter.SpawnOpts{}, err
		}
		workdir = ws
		delivery, err := st.ResolveEngineConfig("memory", sess.ProjectID,
			map[string]string{"delivery": "always_inject"})
		if err != nil {
			return adapter.SpawnOpts{}, err
		}
		if delivery["delivery"] != "on_demand" {
			rendered, err := st.RenderMemory(sess.ProjectID)
			if err != nil {
				return adapter.SpawnOpts{}, err
			}
			primer = rendered
		}
		// Modo "o agente decide": injeta a regra no início da sessão para o próprio
		// agente registrar verdades anti-erro via MCP (suggest_memory) quando perceber.
		mode, err := st.ResolveEngineConfig("memory", sess.ProjectID,
			map[string]string{"__enabled": "false", "__trigger": "project_open_close"})
		if err != nil {
			return adapter.SpawnOpts{}, err
		}
		if mode["__enabled"] == "true" && mode["__trigger"] == "agent_self" {
			rule := "## Memória anti-erro (auto-registro)\n" +
				"Quando você perceber uma verdade anti-erro — um erro seguido de correção, " +
				"ou uma regra que evita repetir um engano — registre-a já, durante a sessão, " +
				"chamando a ferramenta MCP `suggest_memory` com um resumo curto (1-2 frases) e a categoria."
			if primer != "" {
				primer += "\n\n"
			}
			primer += rule
		}
	} else if workdir == "" {
		// não-classificada sem workspace definido: cria scratch
		ws, err := wm.ScratchWorkspace(sessionID)
		if err != nil {
			return adapter.SpawnOpts{}, err
		}
		workdir = ws
	}
	if skillContent != "" {
		if primer != "" {
			primer += "\n\n"
		}
		primer += "## Skill a executar\n" + skillContent +
			"\n\nSiga a skill. Pergunte ao usuário os inputs declarados antes de começar."
	}

	token := uuid.NewString()
	if err := st.SetSessionMCPToken(sessionID, token); err != nil {
		return adapter.SpawnOpts{}, err
	}

	systemAppend := ""
	if agentPersona != "" {
		systemAppend = "# Persona desta sessão\n" + agentPersona
	}

	selfExe, _ := os.Executable()
	return adapter.SpawnOpts{
		SessionID:    sessionID,
		WorkingDir:   workdir,
		Primer:       primer,
		SystemAppend: systemAppend,
		MCPURL:       fmt.Sprintf("http://127.0.0.1:%d/mcp?s=%s", port, token),
		SelfExe:      selfExe,
		Port:         port,
	}, nil
}
