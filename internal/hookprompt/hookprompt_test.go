package hookprompt

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunReturnsAllowDecision(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/sessions/sess-1/permission-request") {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["tool"] != "Bash" {
			t.Errorf("tool = %v", body["tool"])
		}
		w.Write([]byte(`{"decision":"allow"}`))
	}))
	defer srv.Close()

	in := strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"ls"}}`)
	var out strings.Builder
	if err := Run(in, &out, srv.URL, "sess-1", "claude"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"permissionDecision":"allow"`) {
		t.Fatalf("out = %s", out.String())
	}
}

// Codex compartilha o schema do Claude: permissionDecision allow/deny/ask.
func TestRunCodexUsesPermissionDecision(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"decision":"deny"}`))
	}))
	defer srv.Close()

	in := strings.NewReader(`{"tool_name":"shell","tool_input":{}}`)
	var out strings.Builder
	if err := Run(in, &out, srv.URL, "sess-1", "codex"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"permissionDecision":"deny"`) {
		t.Fatalf("out = %s", out.String())
	}
}

// Gemini usa {"decision":"allow"|"deny"}, NÃO permissionDecision.
func TestRunGeminiUsesDecisionField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"decision":"deny"}`))
	}))
	defer srv.Close()

	in := strings.NewReader(`{"tool_name":"run_shell_command","tool_input":{}}`)
	var out strings.Builder
	if err := Run(in, &out, srv.URL, "sess-1", "gemini"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "permissionDecision") {
		t.Fatalf("gemini não deve usar permissionDecision: %s", got)
	}
	if !strings.Contains(got, `"decision":"deny"`) || !strings.Contains(got, `"reason"`) {
		t.Fatalf("out = %s", got)
	}
}

func TestRunFallsBackToAskWhenServerDown(t *testing.T) {
	in := strings.NewReader(`{"tool_name":"Bash","tool_input":{}}`)
	var out strings.Builder
	if err := Run(in, &out, "http://127.0.0.1:1", "sess-1", "claude"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"permissionDecision":"ask"`) {
		t.Fatalf("out = %s", out.String())
	}
}

// Gemini defere ao fluxo nativo emitindo {} (sem decisão) quando o worrel cai.
func TestRunGeminiDefersWhenServerDown(t *testing.T) {
	in := strings.NewReader(`{"tool_name":"run_shell_command","tool_input":{}}`)
	var out strings.Builder
	if err := Run(in, &out, "http://127.0.0.1:1", "sess-1", "gemini"); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(out.String())
	if got != "{}" {
		t.Fatalf("esperava {} (defer), got = %s", got)
	}
}

func TestRunBadStdinFallsBackToAsk(t *testing.T) {
	in := strings.NewReader(`not json`)
	var out strings.Builder
	if err := Run(in, &out, "http://127.0.0.1:1", "sess-1", "claude"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"permissionDecision":"ask"`) {
		t.Fatalf("out = %s", out.String())
	}
}
