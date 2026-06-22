package agui

import (
	"strings"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

func TestProgressPrompt_UsesTailAndContent(t *testing.T) {
	var events []*store.TranscriptEvent
	for i := 0; i < 40; i++ {
		events = append(events, ev("assistant", "text", "linha antiga"))
	}
	events = append(events, ev("user", "text", "faça o deploy"))
	p := ProgressPrompt(events)

	if !strings.Contains(p, "faça o deploy") {
		t.Fatal("prompt deve conter o evento mais recente")
	}
	// só a cauda entra (progressTailEvents), não os 41 eventos.
	if strings.Count(p, "linha antiga") >= 40 {
		t.Fatalf("prompt deve truncar para a cauda, contou %d", strings.Count(p, "linha antiga"))
	}
}

func TestParseProgress(t *testing.T) {
	out := "- agente está lendo o db\n1. encontrou a senha\n\n\"e está fazendo o update\"\nlinha extra demais"
	got := ParseProgress(out)
	if len(got) != 3 {
		t.Fatalf("esperava 3 linhas, veio %d: %#v", len(got), got)
	}
	if got[0] != "agente está lendo o db" || got[1] != "encontrou a senha" || got[2] != "e está fazendo o update" {
		t.Fatalf("limpeza errada: %#v", got)
	}
}

func TestParseProgress_Empty(t *testing.T) {
	if got := ParseProgress("\n  \n"); len(got) != 0 {
		t.Fatalf("esperava vazio, veio %#v", got)
	}
}
