package user

import (
	"regexp"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// Window é um trecho de eventos que carrega um sinal centrado no usuário. É o
// tipo próprio do pacote (não reusa memory.FrictionWindow — isso criaria ciclo
// e acoplaria semânticas distintas).
type Window struct {
	Signal    string
	Events    []*store.TranscriptEvent
	IntentKey string // preenchido em user_repetition (a chave que se repetiu)
}

// Marcadores de CORREÇÃO: o usuário rejeita/reorienta a resposta anterior.
// Casados por substring sobre o texto FOLDED (sem acentos) — evita a fragilidade
// de \b com caracteres acentuados. Ex.: "não é isso" -> "nao e isso".
var correctionMarkers = []string{
	"nao e isso", "nao e iss", "nao era isso", "nao era iss", "nao e bem",
	"nao quero", "nao foi isso", "nao foi bem", "nao ajudou", "nao funcionou",
	"na verdade", "ta errado", "esta errado", "de novo nao", "refaz",
	"nao deu certo", "quero diferente", "nao pedi isso",
}

// Marcadores de FRUSTRAÇÃO: carga emocional inequívoca. Deliberadamente NÃO
// inclui "não funciona"/"deu erro" — em cockpit de código isso quase sempre é
// descrição factual do bug a corrigir, não frustração.
var frustrationMarkers = []string{
	"que droga", "pessimo", "odeio", "inutil", "nao presta", "que raiva",
	"perda de tempo", "cansei disso", "de saco cheio", "que porcaria",
}

// Marcadores de CANCELAMENTO no texto do usuário. "para" nu foi removido
// (preposição onipresente em PT-BR); usamos formas inequívocas com \b.
var cancelTextRe = regexp.MustCompile(`\b(cancela|cancelar|aborta|abortar|chega|parar|deixa pra la|esquece isso|stop)\b`)

// Marcadores de CANCELAMENTO no end_reason da sessão. Os valores reais vêm de
// wrapper.exitReason(): texto livre como "CLI morto por sinal interrupt" ou
// "CLI saiu com código 130 — …". Casamos por substring sobre o valor folded.
var cancelEndReasonMarkers = []string{
	"morto por sinal", "codigo 130", "codigo 143", "codigo 137", "codigo 129",
}

func isUserText(e *store.TranscriptEvent) bool {
	return e.Role == "user" && e.Kind == "text"
}
func isAssistantText(e *store.TranscriptEvent) bool {
	return e.Role == "assistant" && e.Kind == "text"
}

// DetectRepetition acha intenções repetidas na mesma sessão: ≥2 mensagens de
// usuário com a mesma chave canônica. Requer intents (extração LLM) — no modo
// heuristic_only recebe nil e não emite nada (preserva o custo-zero do modo).
// A janela contém as mensagens de usuário que repetiram a intenção.
func DetectRepetition(events []*store.TranscriptEvent, intents []*Intent) []Window {
	if len(intents) < 2 {
		return nil
	}
	bySeq := map[int64]*store.TranscriptEvent{}
	for _, e := range events {
		if isUserText(e) {
			bySeq[e.Seq] = e
		}
	}
	groups := map[string][]*Intent{}
	var order []string
	for _, it := range intents {
		k := it.Key()
		if k == "" {
			continue
		}
		if _, seen := groups[k]; !seen {
			order = append(order, k)
		}
		groups[k] = append(groups[k], it)
	}
	var out []Window
	for _, k := range order {
		g := groups[k]
		if len(g) < 2 {
			continue
		}
		w := Window{Signal: "user_repetition", IntentKey: k}
		for _, it := range g {
			if ev := bySeq[it.Seq]; ev != nil {
				w.Events = append(w.Events, ev)
			}
		}
		if len(w.Events) >= 2 {
			out = append(out, w)
		}
	}
	return out
}

// DetectCorrection acha `user/text` que corrige/rejeita a resposta anterior do
// assistente. A janela inclui o contexto do assistente (quando há) + a mensagem
// de correção. Puramente heurística (sem LLM) — funciona em qualquer modo.
func DetectCorrection(events []*store.TranscriptEvent) []Window {
	return detectMarkerAfterAssistant(events, "user_correction", correctionMarkers)
}

// DetectFrustration acha `user/text` com carga emocional negativa explícita.
func DetectFrustration(events []*store.TranscriptEvent) []Window {
	return detectMarkerAfterAssistant(events, "user_frustration", frustrationMarkers)
}

func detectMarkerAfterAssistant(events []*store.TranscriptEvent, signal string, markers []string) []Window {
	var out []Window
	for i, e := range events {
		if !isUserText(e) {
			continue
		}
		if !containsAny(fold(e.Content), markers) {
			continue
		}
		win := []*store.TranscriptEvent{}
		// contexto: a última fala do assistente antes desta mensagem
		for j := i - 1; j >= 0; j-- {
			if isAssistantText(events[j]) {
				win = append(win, events[j])
				break
			}
		}
		win = append(win, e)
		out = append(out, Window{Signal: signal, Events: win})
	}
	return out
}

// DetectCancellation acha sessões abortadas pelo usuário: ou o end_reason indica
// morte por sinal / exit code de interrupção, ou alguma mensagem `user/text`
// carrega um marcador de cancelamento. Emite no máximo UMA janela por sessão.
func DetectCancellation(endReason string, events []*store.TranscriptEvent) []Window {
	if containsAny(fold(endReason), cancelEndReasonMarkers) {
		return []Window{{Signal: "user_cancellation", Events: lastUserText(events)}}
	}
	for _, e := range events {
		if isUserText(e) && cancelTextRe.MatchString(fold(e.Content)) {
			return []Window{{Signal: "user_cancellation", Events: []*store.TranscriptEvent{e}}}
		}
	}
	return nil
}

func lastUserText(events []*store.TranscriptEvent) []*store.TranscriptEvent {
	for i := len(events) - 1; i >= 0; i-- {
		if isUserText(events[i]) {
			return []*store.TranscriptEvent{events[i]}
		}
	}
	return nil
}

// UserMessages devolve os pares (seq, texto) das mensagens `user/text`, base
// para a extração de intenção. Excludes vazias.
func UserMessages(events []*store.TranscriptEvent) []Message {
	var out []Message
	for _, e := range events {
		if isUserText(e) && e.Content != "" {
			out = append(out, Message{Seq: e.Seq, Text: e.Content})
		}
	}
	return out
}
