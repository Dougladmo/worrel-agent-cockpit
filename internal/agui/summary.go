package agui

import (
	"encoding/json"
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
	b.WriteString("Você observa a sessão de um agente de programação. Responda APENAS em " +
		"JSON, sem texto extra:\n" +
		"{\"title\":\"<2 a 4 palavras: o foco atual da sessão, vivo>\"," +
		"\"lines\":[\"<frase curta do que está acontecendo>\"]}\n" +
		"- title: rótulo curtíssimo (ex.: \"Ajustando o login\", \"Lendo o banco\"). Sem ponto final.\n" +
		"- lines: ATÉ 3 frases simples do que o agente fez/está fazendo, em português.\n" +
		"Não invente — use só o transcript.\n\n" +
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

// ParseProgress extrai o título "vivo" e as linhas de progresso do JSON do LLM
// (tolerante a cercas/lixo ao redor). Em falha, cai num parse linha-a-linha
// (sem título), preservando robustez.
func ParseProgress(out string) (title string, lines []string) {
	if start := strings.IndexByte(out, '{'); start >= 0 {
		if end := strings.LastIndexByte(out, '}'); end > start {
			var raw struct {
				Title string   `json:"title"`
				Lines []string `json:"lines"`
			}
			if json.Unmarshal([]byte(out[start:end+1]), &raw) == nil && (raw.Title != "" || len(raw.Lines) > 0) {
				return cleanTitle(raw.Title), cleanLines(raw.Lines)
			}
		}
	}
	// fallback: trata cada linha como progresso (sem título).
	return "", cleanLines(strings.Split(out, "\n"))
}

func cleanTitle(t string) string {
	t = strings.Trim(strings.TrimSpace(t), "\"'`.")
	if len(t) > 48 {
		t = t[:47] + "…"
	}
	return t
}

func cleanLines(in []string) []string {
	var lines []string
	for _, raw := range in {
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
