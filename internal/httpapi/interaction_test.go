package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/agui"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// TestInteractionSnapshot cobre a borda HTTP do canal AG-UI: cria uma sessão com
// transcript, lê o snapshot e confere o contexto traduzido.
func TestInteractionSnapshot(t *testing.T) {
	ts, s, _ := newSessionsServer(t)
	sess, err := s.CreateSession(&store.Session{Mode: "wrapper"})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.AppendTranscriptEventRich(sess.ID, "user", "text", "rode os testes", "", 0, 0)
	_ = s.AppendTranscriptEventRich(sess.ID, "assistant", "tool_use", "Bash {\"command\":\"go test\"}", "", 0, 0)
	_ = s.AppendTranscriptEventRich(sess.ID, "assistant", "text", "testes passaram, sigo?", "", 0, 0)

	resp, err := ts.Client().Get(ts.URL + "/api/sessions/" + sess.ID + "/interaction")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("GET interaction: err=%v status=%d", err, resp.StatusCode)
	}
	var snap agui.Snapshot
	_ = json.NewDecoder(resp.Body).Decode(&snap)

	if snap.Message != "testes passaram, sigo?" {
		t.Fatalf("message = %q", snap.Message)
	}
	if snap.UserMessage != "rode os testes" {
		t.Fatalf("user_message = %q", snap.UserMessage)
	}
	if len(snap.ToolCalls) != 1 || snap.ToolCalls[0].Name != "Bash" {
		t.Fatalf("tool_calls = %+v", snap.ToolCalls)
	}
	// PTY não está rodando neste teste → sessão considerada encerrada.
	if snap.State != agui.StateEnded {
		t.Fatalf("state = %q, want ended (sem PTY vivo)", snap.State)
	}
}

// TestInteractionPromptNotRunning: injetar prompt numa sessão sem PTY → 409.
func TestInteractionPromptNotRunning(t *testing.T) {
	ts, s, _ := newSessionsServer(t)
	sess, _ := s.CreateSession(&store.Session{Mode: "wrapper"})
	resp, err := ts.Client().Post(
		ts.URL+"/api/sessions/"+sess.ID+"/interaction/prompt",
		"application/json", strings.NewReader(`{"text":"oi"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

// TestInteractionPromptEmpty: corpo sem texto → 400.
func TestInteractionPromptEmpty(t *testing.T) {
	ts, s, _ := newSessionsServer(t)
	sess, _ := s.CreateSession(&store.Session{Mode: "wrapper"})
	resp, _ := ts.Client().Post(
		ts.URL+"/api/sessions/"+sess.ID+"/interaction/prompt",
		"application/json", strings.NewReader(`{"text":"  "}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
