package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readTrust(t *testing.T, home, dir string) any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatal(err)
	}
	projects, _ := root["projects"].(map[string]any)
	entry, _ := projects[dir].(map[string]any)
	if entry == nil {
		return nil
	}
	return entry["hasTrustDialogAccepted"]
}

func TestEnsureFolderTrusted_CreatesAndPreserves(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// config pré-existente com outro projeto + chave de topo a preservar.
	seed := `{"numStartups":7,"projects":{"/outro":{"hasTrustDialogAccepted":true,"keep":42}}}`
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(home, ".worrel", "workspaces", "_scratch-abc")
	ensureFolderTrusted(dir)

	if readTrust(t, home, dir) != true {
		t.Fatal("workspace novo deveria ficar trust=true")
	}
	// preserva o resto
	b, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	var root map[string]any
	_ = json.Unmarshal(b, &root)
	if root["numStartups"].(float64) != 7 {
		t.Fatal("chave de topo perdida")
	}
	other := root["projects"].(map[string]any)["/outro"].(map[string]any)
	if other["keep"].(float64) != 42 {
		t.Fatal("entrada de outro projeto perdida")
	}
}

func TestEnsureFolderTrusted_NoConfigYet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".worrel", "workspaces", "_scratch-xyz")
	ensureFolderTrusted(dir) // sem ~/.claude.json ainda
	if readTrust(t, home, dir) != true {
		t.Fatal("deveria criar o config e confiar a pasta")
	}
}

// Pastas FORA de ~/.worrel (repos reais do usuário) não devem ser tocadas.
func TestEnsureFolderTrusted_SkipsNonWorrel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "Documents", "repos", "meu-projeto")
	ensureFolderTrusted(dir)
	if _, err := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(err) {
		t.Fatal("repo do usuário fora de ~/.worrel não deveria ser tocado")
	}
}

func TestEnsureFolderTrusted_IgnoresEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ensureFolderTrusted("")
	if _, err := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(err) {
		t.Fatal("dir vazio não deveria escrever nada")
	}
}
