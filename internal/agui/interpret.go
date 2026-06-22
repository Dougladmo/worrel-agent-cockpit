package agui

import (
	"encoding/json"
	"strings"
)

// InterpretPrompt monta o prompt que pede ao LLM para classificar uma fala do
// agente (turno encerrado SEM ferramenta) em: é uma escolha (com opções) ou
// espera resposta livre? Isso é o que o auto-mode produz quando o agente fala
// em vez de invocar uma tool — e precisa virar UI acionável.
func InterpretPrompt(message string, context []HistoryLine) string {
	var b strings.Builder
	b.WriteString("Um agente de programação terminou o turno FALANDO (sem usar ferramenta). " +
		"Decida o que o usuário precisa fazer para responder. Responda APENAS em JSON, sem texto extra:\n" +
		"{\"kind\":\"choice\"|\"text\",\"prompt\":\"<a pergunta/decisão em 1 frase curta>\",\"options\":[\"...\"]}\n" +
		"- kind=\"choice\" se o agente ofereceu opções ou pediu uma escolha; preencha options com as opções (curtas).\n" +
		"- kind=\"text\" se espera uma resposta livre; options vazio.\n\n")
	if len(context) > 0 {
		b.WriteString("## Contexto recente\n")
		tail := context
		if len(tail) > 8 {
			tail = tail[len(tail)-8:]
		}
		for _, h := range tail {
			b.WriteString("[" + h.Role + "] " + h.Text + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Última fala do agente\n" + message + "\n")
	return b.String()
}

// Interpretation é o resultado estruturado da classificação.
type Interpretation struct {
	Kind    InterruptKind `json:"kind"`
	Prompt  string        `json:"prompt"`
	Options []string      `json:"options"`
}

// ParseInterpretation extrai o JSON da saída do LLM (tolerante a cercas/lixo ao
// redor). Em falha, devolve kind=text (campo livre) — o fallback seguro.
func ParseInterpretation(out string) Interpretation {
	def := Interpretation{Kind: KindText}
	start := strings.IndexByte(out, '{')
	end := strings.LastIndexByte(out, '}')
	if start < 0 || end <= start {
		return def
	}
	var raw struct {
		Kind    string   `json:"kind"`
		Prompt  string   `json:"prompt"`
		Options []string `json:"options"`
	}
	if json.Unmarshal([]byte(out[start:end+1]), &raw) != nil {
		return def
	}
	kind := KindText
	if raw.Kind == "choice" && len(raw.Options) > 0 {
		kind = KindChoice
	}
	// limpa opções vazias
	opts := make([]string, 0, len(raw.Options))
	for _, o := range raw.Options {
		if o = strings.TrimSpace(o); o != "" {
			opts = append(opts, o)
		}
	}
	if kind == KindChoice && len(opts) == 0 {
		kind = KindText
	}
	return Interpretation{Kind: kind, Prompt: strings.TrimSpace(raw.Prompt), Options: opts}
}
