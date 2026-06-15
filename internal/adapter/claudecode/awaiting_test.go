package claudecode

import (
	"os"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
)

func TestAwaitingInput(t *testing.T) {
	root, jsonlPath := writeFixture(t)
	a := &Adapter{ProjectsRoot: root}
	ref := adapter.SessionRef{Adapter: "claude-code", ExternalRef: "sess-abc"}

	// Fixture termina com um turno de assistente → é a vez do usuário.
	awaiting, ok := a.AwaitingInput(ref)
	if !ok {
		t.Fatal("ok = false, esperava sinal disponível")
	}
	if !awaiting {
		t.Fatal("awaiting = false, esperava true (último evento é assistant)")
	}

	// Acrescenta um turno de usuário → agente vai trabalhar, não-awaiting.
	more := "\n" + `{"type":"user","sessionId":"sess-abc","cwd":"/tmp/proj-x","timestamp":"2026-06-12T10:01:00Z","message":{"role":"user","content":"e o rollback?"}}`
	if err := appendFile(jsonlPath, more); err != nil {
		t.Fatal(err)
	}
	awaiting, ok = a.AwaitingInput(ref)
	if !ok {
		t.Fatal("ok = false após append")
	}
	if awaiting {
		t.Fatal("awaiting = true, esperava false (último evento é user)")
	}
}

func TestAwaitingInputNoTranscript(t *testing.T) {
	a := &Adapter{ProjectsRoot: t.TempDir()}
	_, ok := a.AwaitingInput(adapter.SessionRef{Adapter: "claude-code", ExternalRef: "inexistente"})
	if ok {
		t.Fatal("ok = true, esperava false sem transcrição")
	}
}

func appendFile(path, s string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s)
	return err
}
