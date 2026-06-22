package agui

import (
	"strings"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// progressTailEvents é quantos eventos do FIM do transcript entram no prompt de
// resumo — contexto recente o bastante sem inflar o custo do LLM.
const progressTailEvents = 24

// ProgressPrompt monta o prompt que pede ao LLM um resumo narrado e curto do que
// a sessão está fazendo AGORA — as linhas que aparecem na timeline do card.
// Determinístico (puro) para ser testável; a chamada ao LLM fica na borda.
func ProgressPrompt(events []*store.TranscriptEvent) string {
	tail := events
	if len(tail) > progressTailEvents {
		tail = tail[len(tail)-progressTailEvents:]
	}
	var b strings.Builder
	b.WriteString("Você observa a sessão de um agente de programação. " +
		"Resuma em ATÉ 3 linhas curtas, em português, o que está acontecendo agora: " +
		"o que o agente fez e o que está fazendo. Uma frase simples por linha, " +
		"sem markdown, sem numeração, sem aspas. Não invente — use só o transcript.\n\n" +
		"## Transcript (mais recente no fim)\n")
	for _, e := range tail {
		c := strings.TrimSpace(e.Content)
		if c == "" {
			continue
		}
		if len(c) > 280 {
			c = c[:279] + "…"
		}
		b.WriteString("[" + e.Role + "/" + e.Kind + "] " + c + "\n")
	}
	return b.String()
}

// ParseProgress reduz a saída do LLM às linhas de progresso: tira marcação,
// descarta vazios e limita a 3 linhas de até 120 chars.
func ParseProgress(out string) []string {
	var lines []string
	for _, raw := range strings.Split(out, "\n") {
		l := strings.TrimSpace(raw)
		l = strings.TrimLeft(l, "-*•0123456789. ")
		l = strings.Trim(l, "\"'`")
		l = strings.TrimSpace(l)
		if len(l) < 3 {
			continue
		}
		if len(l) > 120 {
			l = l[:119] + "…"
		}
		lines = append(lines, l)
		if len(lines) == 3 {
			break
		}
	}
	return lines
}
