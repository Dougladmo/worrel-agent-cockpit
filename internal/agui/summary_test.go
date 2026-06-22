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

func TestParseProgress_JSON(t *testing.T) {
	out := "```json\n{\"title\":\"Atualizando o DB\",\"lines\":[\"agente está lendo o db\",\"encontrou a senha\"]}\n```"
	title, lines := ParseProgress(out)
	if title != "Atualizando o DB" {
		t.Fatalf("title = %q", title)
	}
	if len(lines) != 2 || lines[0] != "agente está lendo o db" {
		t.Fatalf("lines = %#v", lines)
	}
}

func TestParseProgress_Fallback(t *testing.T) {
	// sem JSON → cada linha vira progresso, sem título.
	title, lines := ParseProgress("- agente está lendo o db\n1. encontrou a senha")
	if title != "" {
		t.Fatalf("título deveria ser vazio no fallback, veio %q", title)
	}
	if len(lines) != 2 || lines[0] != "agente está lendo o db" {
		t.Fatalf("lines = %#v", lines)
	}
}

func TestParseProgress_Empty(t *testing.T) {
	if title, lines := ParseProgress("\n  \n"); len(lines) != 0 || title != "" {
		t.Fatalf("esperava vazio, veio title=%q lines=%#v", title, lines)
	}
}
