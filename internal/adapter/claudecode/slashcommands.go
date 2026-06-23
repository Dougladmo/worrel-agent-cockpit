package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
)

// ListSlashCommands reúne todos os comandos "/" disponíveis para a sessão: os
// built-ins curados, comandos e skills de plugins, e comandos/skills de usuário e
// de projeto. Triggers em conflito seguem a precedência da ordem de mescla abaixo.
func (a *Adapter) ListSlashCommands(_ context.Context, workingDir string) ([]adapter.SlashCommand, error) {
	claudeDir := a.resolveClaudeDir()

	byTrigger := map[string]adapter.SlashCommand{}
	merge := func(cmds []adapter.SlashCommand) {
		for _, c := range cmds {
			byTrigger[c.Trigger] = c
		}
	}

	// Ordem crescente de precedência: a última fonte a definir um trigger vence.
	merge(builtinSlashCommands())
	merge(scanPluginSlashCommands(claudeDir, workingDir))
	merge(scanSlashCommands(filepath.Join(claudeDir, "commands"), "user"))
	merge(scanClaudeSkills(filepath.Join(claudeDir, "skills"), "", "user"))
	if workingDir != "" {
		projectClaudeDir := filepath.Join(workingDir, ".claude")
		merge(scanSlashCommands(filepath.Join(projectClaudeDir, "commands"), "project"))
		merge(scanClaudeSkills(filepath.Join(projectClaudeDir, "skills"), "", "project"))
	}

	return sortedByTrigger(byTrigger), nil
}

func (a *Adapter) resolveClaudeDir() string {
	if a.userClaudeDir != "" {
		return a.userClaudeDir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".claude")
	}
	return ""
}

func sortedByTrigger(byTrigger map[string]adapter.SlashCommand) []adapter.SlashCommand {
	out := make([]adapter.SlashCommand, 0, len(byTrigger))
	for _, c := range byTrigger {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Trigger < out[j].Trigger })
	return out
}

// scanSlashCommands devolve um SlashCommand por arquivo .md sob commandsDir.
// Subpastas viram namespaces: commands/sc/analyze.md → /sc:analyze. Diretório
// ausente devolve lista vazia.
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
			Description: readFrontmatter(path)["description"],
			Source:      source,
		})
		return nil
	})
	return out
}

func triggerFromRelPath(rel string) string {
	rel = strings.TrimSuffix(rel, ".md")
	return strings.ReplaceAll(filepath.ToSlash(rel), "/", ":")
}

// scanClaudeSkills expõe skills do Claude Code (pastas com SKILL.md) como comandos
// "/": o CLI invoca uma skill pelo nome e isso funciona no headless. Skills de
// plugin recebem o prefixo /<namespace>:, salvo quando o nome já vem namespaced
// (ex.: pinecone → "pinecone:assistant"), pois duplicar viraria "Unknown command".
func scanClaudeSkills(skillsDir, namespace, source string) []adapter.SlashCommand {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	var out []adapter.SlashCommand
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		front := readFrontmatter(filepath.Join(skillsDir, e.Name(), "SKILL.md"))
		if front == nil {
			continue
		}
		name := front["name"]
		if name == "" {
			name = e.Name()
		}
		trigger := "/" + name
		if namespace != "" && !strings.HasPrefix(name, namespace+":") {
			trigger = "/" + namespace + ":" + name
		}
		out = append(out, adapter.SlashCommand{Trigger: trigger, Description: front["description"], Source: source})
	}
	return out
}

// installedPlugins e userSettings espelham os arquivos de ~/.claude que decidem,
// juntos, quais plugins valem na sessão.
type installedPlugins struct {
	Plugins map[string][]struct {
		Scope       string `json:"scope"`       // "user" (global) | "local" (preso a um projeto)
		ProjectPath string `json:"projectPath"` // raiz do projeto, quando scope=="local"
		InstallPath string `json:"installPath"`
	} `json:"plugins"`
}

type userSettings struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins"` // chave "<nome>@<marketplace>"
}

// scanPluginSlashCommands reúne os comandos e skills dos plugins disponíveis na
// sessão, namespaced por plugin (/<plugin>:<x>). Um plugin está disponível quando
// habilitado globalmente nos settings ou, sendo local, quando a sessão roda dentro
// do seu projeto.
func scanPluginSlashCommands(claudeDir, workingDir string) []adapter.SlashCommand {
	if claudeDir == "" {
		return nil
	}
	installed, ok := decodeJSONFile[installedPlugins](filepath.Join(claudeDir, "plugins", "installed_plugins.json"))
	if !ok {
		return nil
	}
	settings, _ := decodeJSONFile[userSettings](filepath.Join(claudeDir, "settings.json"))
	enabledGlobally := settings.EnabledPlugins

	var out []adapter.SlashCommand
	for key, instances := range installed.Plugins {
		pluginName, _, _ := strings.Cut(key, "@")
		for _, inst := range instances {
			isLocalToSession := inst.Scope == "local" && isPathWithin(workingDir, inst.ProjectPath)
			if (!enabledGlobally[key] && !isLocalToSession) || inst.InstallPath == "" {
				continue
			}
			for _, c := range scanSlashCommands(filepath.Join(inst.InstallPath, "commands"), "plugin") {
				c.Trigger = "/" + pluginName + ":" + strings.TrimPrefix(c.Trigger, "/")
				out = append(out, c)
			}
			out = append(out, scanClaudeSkills(filepath.Join(inst.InstallPath, "skills"), pluginName, "plugin")...)
		}
	}
	return out
}

func decodeJSONFile[T any](path string) (T, bool) {
	var v T
	b, err := os.ReadFile(path)
	if err != nil {
		return v, false
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return v, false
	}
	return v, true
}

func isPathWithin(child, parent string) bool {
	if child == "" || parent == "" {
		return false
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// readFrontmatter devolve os campos escalares do frontmatter YAML de um .md (entre
// as cercas "---"). nil quando o arquivo não abre ou não começa com frontmatter —
// o que distingue "sem frontmatter" de um frontmatter presente porém sem o campo.
func readFrontmatter(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return nil
	}
	fields := map[string]string{}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "---" {
			break
		}
		if key, val, ok := strings.Cut(line, ":"); ok {
			fields[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(val), `"'`)
		}
	}
	return fields
}
