package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// trustMu serializa as escritas concorrentes ao ~/.claude.json (várias sessões
// podem spawnar ao mesmo tempo). Não coordena com o próprio claude, mas a
// escrita atômica (tmp + rename) evita leituras parciais.
var trustMu sync.Mutex

// ensureFolderTrusted marca dir como pasta confiável no ~/.claude.json para que
// o claude-code NÃO pare no "trust this folder?" ao abrir uma sessão num
// workspace novo (ex.: _scratch-<id>). Sem isso a sessão fica presa no diálogo
// e nada que o usuário digita/injeta chega ao agente.
//
// É best-effort: qualquer falha é silenciosa (a sessão ainda sobe, só mostrando
// o prompt). Preserva todo o resto do config; escreve de forma atômica.
func ensureFolderTrusted(dir string) {
	if dir == "" {
		return
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	// Só auto-confia pastas DO PRÓPRIO worrel (~/.worrel/...): os workspaces
	// efêmeros que o worrel cria. Repositórios reais do usuário ficam de fora —
	// a confiança deles é decisão do usuário, não nossa.
	worrelRoot := filepath.Join(home, ".worrel") + string(filepath.Separator)
	if !strings.HasPrefix(abs+string(filepath.Separator), worrelRoot) {
		return
	}
	path := filepath.Join(home, ".claude.json")

	trustMu.Lock()
	defer trustMu.Unlock()

	root := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(b, &root) != nil {
			return // config ilegível → não arrisca sobrescrever
		}
	}

	projects, _ := root["projects"].(map[string]any)
	if projects == nil {
		projects = map[string]any{}
		root["projects"] = projects
	}
	entry, _ := projects[abs].(map[string]any)
	if entry == nil {
		entry = map[string]any{}
		projects[abs] = entry
	}
	if entry["hasTrustDialogAccepted"] == true {
		return // já confiado → nada a fazer
	}
	entry["hasTrustDialogAccepted"] = true

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".worrel.tmp"
	if os.WriteFile(tmp, out, 0o600) != nil {
		return
	}
	_ = os.Rename(tmp, path) // atômico no mesmo filesystem
}
