package friction

import (
	"context"

	eng "github.com/eduardoworrel/worrel-agent-cockpit/internal/engine"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/engine/user"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// collectUserSignals coleta os sinais centrados no usuário como Signal do
// roteador. Repetição depende de intents (LLM); nos demais (correção,
// frustração, cancelamento) é puramente heurístico. TargetSkill de uma correção
// vem da única skill usada na sessão (heurística de correlação).
func (e *Engine) collectUserSignals(ctx context.Context, rc eng.RunContext, events []*store.TranscriptEvent) []Signal {
	var intents []*user.Intent
	if rc.Config["detection_mode"] != "heuristic_only" {
		hl, model := e.llm(rc.Config)
		intents, _ = user.ResolveIntents(ctx, hl, model, rc.Store, rc.ProjectID, rc.SessionID, events)
	}

	var endReason string
	if sess, err := rc.Store.GetSession(rc.SessionID); err == nil && sess != nil {
		endReason = sess.EndReason
	}

	// TargetSkill: se a sessão usou exatamente uma skill, correções provavelmente
	// se referem a ela.
	targetSkill := ""
	if used, err := rc.Store.SkillsUsedInSession(rc.SessionID); err == nil && len(used) == 1 {
		targetSkill = used[0]
	}

	var windows []user.Window
	windows = append(windows, user.DetectRepetition(events, intents)...)
	windows = append(windows, user.DetectCorrection(events)...)
	windows = append(windows, user.DetectFrustration(events)...)
	windows = append(windows, user.DetectCancellation(endReason, events)...)

	var out []Signal
	for _, w := range windows {
		s := Signal{Kind: w.Signal, Text: windowText(w.Events), IntentKey: w.IntentKey}
		if w.Signal == "user_correction" {
			s.TargetSkill = targetSkill
		}
		out = append(out, s)
	}
	return out
}
