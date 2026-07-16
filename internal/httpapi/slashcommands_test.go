package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// slashFakeAdapter adiciona SlashCommandLister ao baseFakeAdapter (models_test.go).
type slashFakeAdapter struct {
	baseFakeAdapter
	cmds []adapter.SlashCommand
}

func (f slashFakeAdapter) ListSlashCommands(context.Context, string) ([]adapter.SlashCommand, error) {
	return f.cmds, nil
}

func getSlash(t *testing.T, ts *httptest.Server, id string) (int, []adapter.SlashCommand) {
	t.Helper()
	resp, err := ts.Client().Get(ts.URL + "/api/adapters/" + id + "/slash-commands")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body slashCommandsResponse
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Commands
}

func TestSlashCommandsListerReturnsList(t *testing.T) {
	ts := newModelsServer(t, slashFakeAdapter{
		baseFakeAdapter: baseFakeAdapter{id: "fake"},
		cmds:            []adapter.SlashCommand{{Trigger: "/sc:load", Description: "carrega", Source: "user"}},
	})
	code, got := getSlash(t, ts, "fake")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if len(got) != 1 || got[0].Trigger != "/sc:load" {
		t.Fatalf("commands = %v, want [/sc:load]", got)
	}
}

func TestSlashCommandsWithoutListerReturnsEmpty(t *testing.T) {
	ts := newModelsServer(t, baseFakeAdapter{id: "plain"})
	code, got := getSlash(t, ts, "plain")
	if code != http.StatusOK || len(got) != 0 {
		t.Fatalf("status=%d commands=%v, want 200 []", code, got)
	}
}

func TestSlashCommandsUnknownAdapter404(t *testing.T) {
	ts := newModelsServer(t, baseFakeAdapter{id: "plain"})
	code, _ := getSlash(t, ts, "nope")
	if code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", code)
	}
}

// dirCapturingAdapter registra o dir recebido em ListSlashCommands, para provar
// que o endpoint só repassa dirs de workspaces conhecidos.
type dirCapturingAdapter struct {
	baseFakeAdapter
	gotDir *string
}

func (f dirCapturingAdapter) ListSlashCommands(_ context.Context, dir string) ([]adapter.SlashCommand, error) {
	*f.gotDir = dir
	return nil, nil
}

func getSlashDir(t *testing.T, ts *httptest.Server, id, dir string) {
	t.Helper()
	resp, err := ts.Client().Get(ts.URL + "/api/adapters/" + id + "/slash-commands?dir=" + url.QueryEscape(dir))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

// TestSlashCommandsRestrictsDirToKnownWorkspace garante que ?dir= só é aceito
// quando aponta para o workspace de uma sessão real; um caminho arbitrário
// degrada para vazio (nível de usuário), fechando a leitura de <dir>/.claude
// de qualquer lugar do disco.
func TestSlashCommandsRestrictsDirToKnownWorkspace(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	sess, _ := st.CreateSession(&store.Session{Adapter: "engine:claude-code", Mode: "wrapper"})
	known := t.TempDir()
	if err := st.SetSessionWorkspaceDir(sess.ID, known); err != nil {
		t.Fatal(err)
	}

	var gotDir string
	reg := adapter.NewRegistry()
	reg.Register(dirCapturingAdapter{baseFakeAdapter{id: "claude-code"}, &gotDir})
	srv := New(Deps{Adapters: reg, Store: st})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	getSlashDir(t, ts, "claude-code", known)
	if gotDir != known {
		t.Fatalf("dir conhecido: adapter recebeu %q, quer %q", gotDir, known)
	}

	getSlashDir(t, ts, "claude-code", "/etc/nao-autorizado")
	if gotDir != "" {
		t.Fatalf("dir arbitrário deveria degradar p/ vazio, mas veio %q", gotDir)
	}
}
