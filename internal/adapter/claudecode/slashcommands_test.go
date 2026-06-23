package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeCmd cria um arquivo de comando .md com frontmatter opcional.
func writeCmd(t *testing.T, path, description string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "corpo do comando"
	content := body
	if description != "" {
		content = "---\nname: ignored\ndescription: \"" + description + "\"\n---\n" + body
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanSlashCommands_NamespaceFromSubdir(t *testing.T) {
	userHome := t.TempDir()
	writeCmd(t, filepath.Join(userHome, ".claude", "commands", "sc", "analyze.md"), "analisa o código")
	writeCmd(t, filepath.Join(userHome, ".claude", "commands", "deploy.md"), "")

	cmds := scanSlashCommands(filepath.Join(userHome, ".claude", "commands"), "user")

	got := map[string]adapterSlash{}
	for _, c := range cmds {
		got[c.Trigger] = adapterSlash{c.Description, c.Source}
	}
	if v, ok := got["/sc:analyze"]; !ok || v.desc != "analisa o código" || v.source != "user" {
		t.Fatalf("esperava /sc:analyze com descrição e source=user, got %+v", got)
	}
	if v, ok := got["/deploy"]; !ok || v.source != "user" {
		t.Fatalf("esperava /deploy (sem namespace), got %+v", got)
	}
}

func TestScanSlashCommands_DeepNamespace(t *testing.T) {
	root := t.TempDir()
	writeCmd(t, filepath.Join(root, "a", "b", "c.md"), "")
	cmds := scanSlashCommands(root, "project")
	if len(cmds) != 1 || cmds[0].Trigger != "/a:b:c" {
		t.Fatalf("esperava /a:b:c, got %+v", cmds)
	}
}

func TestScanSlashCommands_MissingDir(t *testing.T) {
	cmds := scanSlashCommands(filepath.Join(t.TempDir(), "inexistente"), "user")
	if len(cmds) != 0 {
		t.Fatalf("dir ausente deve devolver vazio, got %+v", cmds)
	}
}

func TestListSlashCommands_MergesUserAndProject(t *testing.T) {
	userHome := t.TempDir()
	writeCmd(t, filepath.Join(userHome, ".claude", "commands", "sc", "load.md"), "carrega contexto")
	projectDir := t.TempDir()
	writeCmd(t, filepath.Join(projectDir, ".claude", "commands", "ship.md"), "")

	a := &Adapter{userClaudeDir: filepath.Join(userHome, ".claude")}
	cmds, err := a.ListSlashCommands(context.Background(), projectDir)
	if err != nil {
		t.Fatal(err)
	}
	triggers := map[string]string{}
	for _, c := range cmds {
		triggers[c.Trigger] = c.Source
	}
	if triggers["/sc:load"] != "user" {
		t.Fatalf("esperava /sc:load de user, got %+v", triggers)
	}
	if triggers["/ship"] != "project" {
		t.Fatalf("esperava /ship de project, got %+v", triggers)
	}
	// Built-ins do CLI (hardcoded) devem aparecer junto com os de arquivo.
	if triggers["/compact"] != "builtin" || triggers["/init"] != "builtin" {
		t.Fatalf("esperava built-ins /compact e /init, got %+v", triggers)
	}
}

func TestListSlashCommands_UserOverridesBuiltin(t *testing.T) {
	userHome := t.TempDir()
	// Um comando de usuário homônimo a um built-in deve sobrescrever a origem.
	writeCmd(t, filepath.Join(userHome, ".claude", "commands", "review.md"), "minha review")
	a := &Adapter{userClaudeDir: filepath.Join(userHome, ".claude")}
	cmds, err := a.ListSlashCommands(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cmds {
		if c.Trigger == "/review" {
			if c.Source != "user" || c.Description != "minha review" {
				t.Fatalf("esperava /review sobrescrito por user, got %+v", c)
			}
			return
		}
	}
	t.Fatal("não encontrou /review")
}

type adapterSlash struct {
	desc   string
	source string
}

// setupPlugin cria a árvore de um plugin instalado: o arquivo de comando e a
// entrada em installed_plugins.json. Devolve o installPath.
func setupPlugin(t *testing.T, claudeDir, name, cmdFile string) string {
	t.Helper()
	installPath := filepath.Join(claudeDir, "plugins", "cache", name)
	writeCmd(t, filepath.Join(installPath, "commands", cmdFile), "comando do plugin")
	return installPath
}

func writeInstalledPlugins(t *testing.T, claudeDir, body string) {
	t.Helper()
	path := filepath.Join(claudeDir, "plugins", "installed_plugins.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPluginSlashCommands_UserEnabledAndLocalScope(t *testing.T) {
	claudeDir := t.TempDir()
	projectDir := t.TempDir()

	userPath := setupPlugin(t, claudeDir, "speckit", "speckit-specify.md")
	localPath := setupPlugin(t, claudeDir, "pinecone", "join-discord.md")
	disabledPath := setupPlugin(t, claudeDir, "dormant", "nope.md")

	writeInstalledPlugins(t, claudeDir, `{"plugins":{
		"speckit@mkt":[{"scope":"user","installPath":"`+userPath+`"}],
		"pinecone@mkt":[{"scope":"local","projectPath":"`+projectDir+`","installPath":"`+localPath+`"}],
		"dormant@mkt":[{"scope":"user","installPath":"`+disabledPath+`"}]
	}}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"enabledPlugins":{"speckit@mkt":true,"dormant@mkt":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &Adapter{userClaudeDir: claudeDir}
	cmds, err := a.ListSlashCommands(context.Background(), projectDir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range cmds {
		got[c.Trigger] = c.Source
	}
	if got["/speckit:speckit-specify"] != "plugin" {
		t.Fatalf("esperava /speckit:speckit-specify (user habilitado), got %+v", got)
	}
	if got["/pinecone:join-discord"] != "plugin" {
		t.Fatalf("esperava /pinecone:join-discord (local no projeto), got %+v", got)
	}
	if _, ok := got["/dormant:nope"]; ok {
		t.Fatalf("plugin desabilitado não deveria aparecer, got %+v", got)
	}
}

// writeSkill cria <skillsDir>/<dir>/SKILL.md com frontmatter name/description.
func writeSkill(t *testing.T, skillsDir, dir, name, description string) {
	t.Helper()
	path := filepath.Join(skillsDir, dir, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: \"" + name + "\"\ndescription: \"" + description + "\"\n---\ncorpo"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListSlashCommands_DiscoversProjectAndPluginSkills(t *testing.T) {
	claudeDir := t.TempDir()
	projectDir := t.TempDir()

	// Skill de projeto (cenário spec-kit): /speckit-specify, sem namespace.
	writeSkill(t, filepath.Join(projectDir, ".claude", "skills"), "speckit-specify",
		"speckit-specify", "Cria a especificação da feature")

	// Skill de plugin habilitado (user): vira /<plugin>:<nome>.
	pluginPath := filepath.Join(claudeDir, "plugins", "cache", "superpowers")
	writeSkill(t, filepath.Join(pluginPath, "skills"), "tdd", "test-driven-development", "TDD")
	writeInstalledPlugins(t, claudeDir, `{"plugins":{
		"superpowers@mkt":[{"scope":"user","installPath":"`+pluginPath+`"}]
	}}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"enabledPlugins":{"superpowers@mkt":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &Adapter{userClaudeDir: claudeDir}
	cmds, err := a.ListSlashCommands(context.Background(), projectDir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range cmds {
		got[c.Trigger] = c.Source
	}
	if got["/speckit-specify"] != "project" {
		t.Fatalf("esperava /speckit-specify (skill de projeto), got %+v", got)
	}
	if got["/superpowers:test-driven-development"] != "plugin" {
		t.Fatalf("esperava /superpowers:test-driven-development (skill de plugin), got %+v", got)
	}
}

func TestPluginSkills_PreNamespacedNameNotDuplicated(t *testing.T) {
	claudeDir := t.TempDir()
	pluginPath := filepath.Join(claudeDir, "plugins", "cache", "pinecone")
	// Skill cujo name já vem namespaced (ex.: pinecone) não deve virar
	// /pinecone:pinecone:assistant — o CLI invoca /pinecone:assistant.
	writeSkill(t, filepath.Join(pluginPath, "skills"), "assistant", "pinecone:assistant", "Assistant")
	writeInstalledPlugins(t, claudeDir, `{"plugins":{
		"pinecone@mkt":[{"scope":"user","installPath":"`+pluginPath+`"}]
	}}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"enabledPlugins":{"pinecone@mkt":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &Adapter{userClaudeDir: claudeDir}
	cmds, err := a.ListSlashCommands(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range cmds {
		got[c.Trigger] = true
	}
	if !got["/pinecone:assistant"] {
		t.Fatalf("esperava /pinecone:assistant, got %+v", got)
	}
	if got["/pinecone:pinecone:assistant"] {
		t.Fatalf("não deveria duplicar o namespace (/pinecone:pinecone:assistant)")
	}
}

func TestScanClaudeSkills_SkipsDirsWithoutSkillFile(t *testing.T) {
	skillsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(skillsDir, "vazio"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, skillsDir, "real", "real-skill", "ok")
	got := scanClaudeSkills(skillsDir, "", "user")
	if len(got) != 1 || got[0].Trigger != "/real-skill" {
		t.Fatalf("esperava só /real-skill, got %+v", got)
	}
}

func TestPluginSlashCommands_LocalScopeOutsideProjectHidden(t *testing.T) {
	claudeDir := t.TempDir()
	projectDir := t.TempDir()
	otherDir := t.TempDir()

	localPath := setupPlugin(t, claudeDir, "pinecone", "join-discord.md")
	writeInstalledPlugins(t, claudeDir, `{"plugins":{
		"pinecone@mkt":[{"scope":"local","projectPath":"`+projectDir+`","installPath":"`+localPath+`"}]
	}}`)

	a := &Adapter{userClaudeDir: claudeDir}
	// Sessão rodando em OUTRO diretório: o plugin local não deve aparecer.
	cmds, err := a.ListSlashCommands(context.Background(), otherDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cmds {
		if c.Trigger == "/pinecone:join-discord" {
			t.Fatalf("plugin local não deveria aparecer fora do seu projeto")
		}
	}
}
