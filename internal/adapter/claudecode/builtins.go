package claudecode

import "github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"

// builtinSlashCommands lista os comandos "/" embutidos do Claude Code — o CLI não
// os enumera, então curamos à mão. Inclui só os que rodam no modo headless/stream-json
// do worrel; os de TUI interativa (/resume, /login, /model, /mcp…) respondem
// "isn't available in this environment" e ficam de fora. Validado contra o CLI v2.1.x.
func builtinSlashCommands() []adapter.SlashCommand {
	specs := [...][2]string{
		{"/clear", "Inicia uma nova conversa com contexto vazio"},
		{"/compact", "Libera contexto resumindo a conversa"},
		{"/config", "Define valores de config (key=value)"},
		{"/context", "Visualiza o uso atual de contexto"},
		{"/cost", "Mostra custo e uso da sessão"},
		{"/init", "Inicializa o projeto com um guia CLAUDE.md"},
		{"/review", "Revisa um pull request localmente"},
		{"/security-review", "Analisa mudanças em busca de vulnerabilidades"},
		{"/usage", "Mostra custo da sessão e uso do plano"},
	}
	out := make([]adapter.SlashCommand, 0, len(specs))
	for _, s := range specs {
		out = append(out, adapter.SlashCommand{Trigger: s[0], Description: s[1], Source: "builtin"})
	}
	return out
}
