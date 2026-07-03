package httpapi

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestEngineHealthTransitions(t *testing.T) {
	h := newEngineHealth()

	// 1ª falha = borda ok→falha.
	if !h.fail("summary", "opencode", "m", "timeout", "boom", 1000) {
		t.Fatal("1ª falha deveria sinalizar transição")
	}
	// 2ª falha do mesmo engine NÃO é borda e preserva Since.
	if h.fail("summary", "opencode", "m", "timeout", "boom2", 2000) {
		t.Fatal("2ª falha consecutiva não deveria sinalizar transição")
	}
	got := h.list()
	if len(got) != 1 {
		t.Fatalf("esperava 1 engine falhando, veio %d", len(got))
	}
	if got[0].Since != 1000 {
		t.Fatalf("Since deveria ser preservado (1000), veio %d", got[0].Since)
	}
	if got[0].LastErr != "boom2" {
		t.Fatalf("LastErr deveria atualizar, veio %q", got[0].LastErr)
	}

	// ok = borda falha→ok e limpa a entrada.
	if !h.ok("summary") {
		t.Fatal("ok deveria sinalizar transição falha→ok")
	}
	if len(h.list()) != 0 {
		t.Fatal("lista deveria ficar vazia após ok")
	}
	// ok num engine já saudável = sem transição.
	if h.ok("summary") {
		t.Fatal("ok num engine saudável não deveria sinalizar transição")
	}
}

func TestClassifyLLMErr(t *testing.T) {
	if got := classifyLLMErr(context.DeadlineExceeded); got != "timeout" {
		t.Fatalf("DeadlineExceeded deveria virar timeout, veio %q", got)
	}
	if got := classifyLLMErr(fmt.Errorf("wrap: %w", context.DeadlineExceeded)); got != "timeout" {
		t.Fatalf("erro que embrulha DeadlineExceeded deveria virar timeout, veio %q", got)
	}
	if got := classifyLLMErr(errors.New("parse boom")); got != "other" {
		t.Fatalf("erro genérico deveria virar other, veio %q", got)
	}
}
