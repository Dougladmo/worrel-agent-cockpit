package adapter

import (
	"os/exec"
	"time"
)

// headlessWaitDelay é a folga após o cancelamento do contexto em que Wait ainda
// espera pelo processo antes de FECHAR os pipes à força e retornar. Sem isso, um
// CLI headless (ex.: `opencode run`) que deixa um NETO herdando o stdout mantém
// o pipe aberto após o SIGKILL do processo direto — e cmd.Output()/Wait bloqueia
// PARA SEMPRE. Isso trava a goroutine do summarizer com o guard `inflight` preso,
// e a sessão nunca mais gera título/progresso (card fica "aguardando"). O
// WaitDelay garante que Output retorna (com erro) em ctx+delay em vez de nunca.
const headlessWaitDelay = 3 * time.Second

// HeadlessOutput roda um exec.Cmd JÁ montado (Dir/Env/Stdin/args do adapter) e
// devolve stdout, garantindo o WaitDelay antes do Wait. É o ponto único onde os
// adapters (opencode/claude/codex/antigravity) executam headless: antes cada um
// duplicava cmd.Output() SEM WaitDelay e podia travar indefinidamente quando o
// CLI deixava um neto segurando o pipe. Passe um cmd criado com
// exec.CommandContext para que o cancelamento de ctx continue matando o processo.
func HeadlessOutput(cmd *exec.Cmd) ([]byte, error) {
	cmd.WaitDelay = headlessWaitDelay
	return cmd.Output()
}
