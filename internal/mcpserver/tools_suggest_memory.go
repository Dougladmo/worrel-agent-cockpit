package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

type suggestMemoryArg struct {
	Content  string `json:"content" jsonschema:"a verdade anti-erro a lembrar (1-2 frases)"`
	Category string `json:"category,omitempty" jsonschema:"opcional: convencao|arquitetura|gotcha|never_do|decisao"`
	Evidence string `json:"evidence,omitempty" jsonschema:"opcional: trecho/contexto que originou a regra"`
}

// addSuggestMemoryTools expõe a tool de auto-reporte usada no modo "o agente
// decide": o próprio agente registra uma verdade anti-erro quando a percebe; ela
// vira uma sugestão pendente de memória para o usuário revisar.
func (svc *Service) addSuggestMemoryTools(srv *mcp.Server, a *attribution) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "suggest_memory",
		Description: "Registra uma verdade anti-erro (golden truth) percebida nesta sessão como sugestão de memória, para o usuário revisar. Use quando notar um erro→correção ou regra que vale lembrar.",
	},
		func(ctx context.Context, req *mcp.CallToolRequest, in suggestMemoryArg) (*mcp.CallToolResult, any, error) {
			sessID, projID := a.sessionProject()
			if projID == "" {
				return errResult("sessão sem projeto vinculado"), nil, nil
			}
			if in.Content == "" {
				return errResult("content obrigatório"), nil, nil
			}
			cat := in.Category
			if cat == "" {
				cat = "gotcha"
			}
			payload, _ := json.Marshal(map[string]string{"content": in.Content, "category": cat, "evidence": in.Evidence})
			title := in.Content
			if r := []rune(title); len(r) > 80 {
				title = string(r[:80])
			}
			sid := sessID
			if _, err := svc.store.CreateSuggestion(&store.Suggestion{
				ProjectID: projID, SessionID: &sid, Type: "add_memory_entry",
				Title: title, Payload: string(payload), Origin: "agent:memory",
			}); err != nil {
				return errResult(err.Error()), nil, nil
			}
			return textResult("memória sugerida (aguarda revisão do usuário)"), nil, nil
		})
}
