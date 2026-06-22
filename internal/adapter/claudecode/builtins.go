package claudecode

import "github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"

// builtinSlashCommands devolve os comandos "/" EMBUTIDOS do Claude Code. Eles são
// hardcoded no CLI (não existem como arquivos em ~/.claude/commands e o `claude`
// não os enumera programaticamente), então mantemos uma lista CURADA — mesma
// estratégia de curatedClaudeModels() para modelos.
//
// É um snapshot dos comandos PADRÃO/estáveis da família atual (v2.1.x). Inclui de
// propósito apenas os de uso geral; comandos puramente cosméticos/novelty ou
// presos a plataforma/plano (ex.: /theme, /stickers, /radio, /desktop,
// /setup-bedrock) ficam de fora para não poluir o menu. Comandos removidos do CLI
// (ex.: /vim, /pr-comments) também não entram. A lista pode divergir por versão;
// um comando de usuário homônimo em ~/.claude/commands sobrescreve o built-in.
func builtinSlashCommands() []adapter.SlashCommand {
	specs := [...][2]string{
		{"/add-dir", "Adiciona um diretório de trabalho ao acesso de arquivos"},
		{"/agents", "Gerencia configurações de agentes"},
		{"/clear", "Inicia uma nova conversa com contexto vazio"},
		{"/compact", "Libera contexto resumindo a conversa"},
		{"/config", "Abre as configurações ou define valores de config"},
		{"/context", "Visualiza o uso atual de contexto"},
		{"/cost", "Mostra custo e uso da sessão"},
		{"/doctor", "Diagnostica a instalação do Claude Code"},
		{"/exit", "Sai do CLI"},
		{"/export", "Exporta a conversa como texto"},
		{"/help", "Mostra ajuda e comandos disponíveis"},
		{"/hooks", "Vê as configurações de hooks"},
		{"/ide", "Gerencia integrações com IDEs"},
		{"/init", "Inicializa o projeto com um guia CLAUDE.md"},
		{"/login", "Entra na conta Anthropic"},
		{"/logout", "Sai da conta Anthropic"},
		{"/mcp", "Gerencia conexões de servidores MCP"},
		{"/memory", "Edita o CLAUDE.md e gerencia a memória"},
		{"/model", "Troca o modelo de IA"},
		{"/permissions", "Gerencia as regras de permissão de ferramentas"},
		{"/plan", "Entra no modo de planejamento"},
		{"/plugin", "Gerencia plugins do Claude Code"},
		{"/privacy-settings", "Vê e atualiza as configurações de privacidade"},
		{"/release-notes", "Vê o changelog"},
		{"/rename", "Renomeia a sessão atual"},
		{"/resume", "Retoma uma conversa por ID ou nome"},
		{"/review", "Revisa um pull request localmente"},
		{"/rewind", "Volta a conversa/código a um ponto anterior"},
		{"/security-review", "Analisa mudanças em busca de vulnerabilidades"},
		{"/skills", "Lista as skills disponíveis"},
		{"/status", "Mostra o status do sistema e configurações"},
		{"/statusline", "Configura a status line do Claude Code"},
		{"/tasks", "Vê e gerencia tarefas em segundo plano"},
		{"/terminal-setup", "Configura atalhos do terminal"},
		{"/usage", "Mostra custo da sessão e uso do plano"},
	}
	out := make([]adapter.SlashCommand, 0, len(specs))
	for _, s := range specs {
		out = append(out, adapter.SlashCommand{Trigger: s[0], Description: s[1], Source: "builtin"})
	}
	return out
}
