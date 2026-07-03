package adapter

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// TestHeadlessOutputDoesNotHangOnLeakedGrandchild reproduz o bug real: um CLI
// headless (aqui um `sh` simulando o `opencode run`) deixa um NETO herdando o
// stdout e não termina. Sem WaitDelay, cmd.Output() bloquearia até o neto morrer
// (na prática, nunca) — travando a goroutine do summarizer e congelando o card.
// HeadlessOutput deve retornar (com erro) logo após ctx+WaitDelay, não travar.
func TestHeadlessOutputDoesNotHangOnLeakedGrandchild(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// `sleep 60 &` herda o stdout e sobrevive ao fim do sh: é o neto que segura o pipe.
	cmd := exec.CommandContext(ctx, "sh", "-c", "sleep 60 & echo oi; wait $!")
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, err := HeadlessOutput(cmd)
		done <- err
	}()

	// Deve retornar dentro de ctx(0.5s) + WaitDelay(3s) + folga. Se travar, o
	// select estoura e o teste falha — exatamente o sintoma do usuário.
	select {
	case <-done:
		if elapsed := time.Since(start); elapsed > headlessWaitDelay+3*time.Second {
			t.Fatalf("HeadlessOutput demorou %v — WaitDelay não surtiu efeito", elapsed)
		}
	case <-time.After(headlessWaitDelay + 5*time.Second):
		t.Fatal("HeadlessOutput TRAVOU num neto segurando o pipe (bug do summarizer)")
	}
}
