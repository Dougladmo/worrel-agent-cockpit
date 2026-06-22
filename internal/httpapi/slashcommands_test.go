package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
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
