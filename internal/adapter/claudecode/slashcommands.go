package claudecode

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
)

// ListSlashCommands descobre os comandos "/" nativos do Claude Code lendo os
// arquivos .md em ~/.claude/commands (nível de usuário) e, quando workingDir é
// informado, em <workingDir>/.claude/commands (nível de projeto). Comandos de
// projeto têm precedência sobre os de usuário com o mesmo trigger.
func (a *Adapter) ListSlashCommands(_ context.Context, workingDir string) ([]adapter.SlashCommand, error) {
	claudeDir := a.userClaudeDir
	if claudeDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			claudeDir = filepath.Join(home, ".claude")
		}
	}

	byTrigger := map[string]adapter.SlashCommand{}
	for _, c := range builtinSlashCommands() {
		byTrigger[c.Trigger] = c
	}
	for _, c := range scanSlashCommands(filepath.Join(claudeDir, "commands"), "user") {
		byTrigger[c.Trigger] = c // comando de usuário sobrescreve built-in homônimo
	}
	if workingDir != "" {
		for _, c := range scanSlashCommands(filepath.Join(workingDir, ".claude", "commands"), "project") {
			byTrigger[c.Trigger] = c // projeto sobrescreve usuário
		}
	}

	out := make([]adapter.SlashCommand, 0, len(byTrigger))
	for _, c := range byTrigger {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Trigger < out[j].Trigger })
	return out, nil
}

// scanSlashCommands varre recursivamente um diretório de comandos e devolve um
// SlashCommand por arquivo .md. O trigger deriva do caminho relativo: subpastas
// viram namespaces separados por ":" (commands/sc/analyze.md → /sc:analyze).
// Diretório ausente devolve lista vazia (degradação graciosa).
func scanSlashCommands(commandsDir, source string) []adapter.SlashCommand {
	var out []adapter.SlashCommand
	_ = filepath.WalkDir(commandsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(commandsDir, path)
		if relErr != nil {
			return nil
		}
		out = append(out, adapter.SlashCommand{
			Trigger:     "/" + triggerFromRelPath(rel),
			Description: readCommandDescription(path),
			Source:      source,
		})
		return nil
	})
	return out
}

// triggerFromRelPath converte "sc/analyze.md" → "sc:analyze" e "a/b/c.md" → "a:b:c".
func triggerFromRelPath(rel string) string {
	rel = strings.TrimSuffix(rel, ".md")
	return strings.ReplaceAll(filepath.ToSlash(rel), "/", ":")
}

// readCommandDescription extrai o campo `description` do frontmatter YAML do
// arquivo de comando, se houver. Devolve "" quando ausente. Leitura tolerante:
// não falha o scan inteiro por causa de um arquivo ilegível.
func readCommandDescription(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return "" // sem frontmatter
	}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "---" {
			break
		}
		if rest, ok := strings.CutPrefix(line, "description:"); ok {
			return strings.Trim(strings.TrimSpace(rest), `"'`)
		}
	}
	return ""
}
