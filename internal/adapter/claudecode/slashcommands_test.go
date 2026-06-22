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
	if triggers["/compact"] != "builtin" || triggers["/mcp"] != "builtin" {
		t.Fatalf("esperava built-ins /compact e /mcp, got %+v", triggers)
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
