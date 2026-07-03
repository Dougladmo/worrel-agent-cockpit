package user

import (
	"context"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// ResolveIntents devolve as intenções de todas as mensagens `user/text` da
// sessão, usando o store como CACHE: só chama o LLM para os seqs ainda não
// extraídos, e persiste os novos. Idempotente entre execuções do motor sobre a
// mesma sessão (o gargalo de custo do LLM-as-extractor).
//
// Deve ser chamada apenas fora do modo heuristic_only (extração é LLM). Erros de
// extração não são fatais: devolve o que já havia em cache.
func ResolveIntents(ctx context.Context, h Headless, model string, st *store.Store, projectID, sessionID string, events []*store.TranscriptEvent) ([]*Intent, error) {
	cached, err := st.ListSessionIntents(sessionID)
	if err != nil {
		return nil, err
	}
	haveSeq := map[int64]bool{}
	var out []*Intent
	for _, c := range cached {
		haveSeq[c.Seq] = true
		// Do cache só temos a chave canônica (não os slots); reconstruímos um
		// Intent cuja Key() reproduz a chave persistida.
		out = append(out, WithKey(c.Seq, c.Summary, c.Category, c.IntentKey))
	}

	var missing []Message
	for _, m := range UserMessages(events) {
		if !haveSeq[m.Seq] {
			missing = append(missing, m)
		}
	}
	if len(missing) == 0 {
		return out, nil
	}

	extracted, err := ExtractIntents(ctx, h, model, missing)
	if err != nil {
		return out, nil // degrada para o cache; não falha o motor
	}
	for _, it := range extracted {
		if it == nil {
			continue
		}
		_ = st.SaveUserIntent(&store.UserIntent{
			ProjectID: projectID, SessionID: sessionID, Seq: it.Seq,
			Summary: it.Summary, Category: it.Category, IntentKey: it.Key(),
		})
		out = append(out, it)
	}
	return out, nil
}
