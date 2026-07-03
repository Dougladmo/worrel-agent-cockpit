package memory

import (
	"context"
	"strings"

	eng "github.com/eduardoworrel/worrel-agent-cockpit/internal/engine"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/engine/user"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// userSignalsField é o toggle (compartilhado por copy com o motor de fricção)
// que liga a detecção centrada no usuário.
func userSignalsField() eng.ConfigField {
	return eng.ConfigField{Key: "user_signals", Label: "Sinais de usuário", Type: "select", Default: "off", Options: []eng.ConfigOption{
		{Value: "off", Label: "Desligado", Description: "Apenas detecção clássica (erro→retry)."},
		{Value: "on", Label: "Ligado", Description: "Detecta repetição, correção, frustração e cancelamento do usuário (usa LLM fora do modo só-heurística)."},
	}}
}

// userCentricTruths coleta sinais centrados no usuário e os converte em golden
// truths — um construtor POR TIPO de sinal (não reusa heuristicTruth, que
// pressupõe tool_use). Repetição depende de intents (LLM); no modo
// heuristic_only a extração é pulada e só correção/frustração/cancelamento
// (puramente heurísticos) rodam.
func (e *Engine) userCentricTruths(ctx context.Context, rc eng.RunContext, events []*store.TranscriptEvent) []GoldenTruth {
	var intents []*user.Intent
	if rc.Config["detection_mode"] != "heuristic_only" {
		hl, model := e.llm(rc.Config)
		intents, _ = user.ResolveIntents(ctx, hl, model, rc.Store, rc.ProjectID, rc.SessionID, events)
	}

	var windows []user.Window
	windows = append(windows, user.DetectRepetition(events, intents)...)
	windows = append(windows, user.DetectCorrection(events)...)
	windows = append(windows, user.DetectFrustration(events)...)
	var endReason string
	if sess, err := rc.Store.GetSession(rc.SessionID); err == nil && sess != nil {
		endReason = sess.EndReason
	}
	windows = append(windows, user.DetectCancellation(endReason, events)...)

	var out []GoldenTruth
	for _, w := range windows {
		if gt := truthFromUserWindow(w); gt.Content != "" {
			out = append(out, gt)
		}
	}
	return out
}

// truthFromUserWindow monta o golden truth conforme o tipo de sinal do usuário.
func truthFromUserWindow(w user.Window) GoldenTruth {
	userText := lastUserContent(w.Events)
	switch w.Signal {
	case "user_repetition":
		return GoldenTruth{
			Content:  "O usuário pediu a mesma tarefa mais de uma vez (" + firstNonEmpty(userText, w.IntentKey) + "). Antecipe o que faltou na primeira resposta.",
			Category: "gotcha", Evidence: "user_repetition:" + w.IntentKey,
		}
	case "user_correction":
		return GoldenTruth{
			Content:  "O usuário corrigiu a resposta do agente: " + userText,
			Category: "never_do", Evidence: "user_correction",
		}
	case "user_frustration":
		return GoldenTruth{
			Content:  "Ponto de fricção que frustrou o usuário: " + userText,
			Category: "gotcha", Evidence: "user_frustration",
		}
	case "user_cancellation":
		return GoldenTruth{
			Content:  "O usuário abortou o agente" + optionalSuffix(userText),
			Category: "gotcha", Evidence: "user_cancellation",
		}
	}
	return GoldenTruth{}
}

func lastUserContent(events []*store.TranscriptEvent) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Role == "user" && events[i].Kind == "text" {
			return strings.TrimSpace(events[i].Content)
		}
	}
	return ""
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func optionalSuffix(s string) string {
	if strings.TrimSpace(s) == "" {
		return "."
	}
	return ": " + s
}
