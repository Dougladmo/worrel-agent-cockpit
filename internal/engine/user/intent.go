// Package user implementa a detecção de sinais centrados no USUÁRIO (e não no
// output do agente): repetição de pedidos, correções explícitas, frustração e
// cancelamento. Também extrai a INTENÇÃO de cada mensagem `user/text` via LLM e
// a reduz a uma CHAVE CANÔNICA (categoria|ação|objeto) — estável entre sessões,
// sem hashear texto livre — que os motores usam para casar recorrência.
//
// Import: este pacote depende de `store` (tipo TranscriptEvent), mas `store`
// NÃO importa `user`; e a janela de saída (Window) vive aqui, não é reusada de
// `memory`. Assim não há ciclo de import.
package user

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
)

// Headless é o executor headless do LLM (mesmo contrato dos demais motores).
type Headless interface {
	RunHeadless(ctx context.Context, prompt string, opts adapter.HeadlessOpts) (string, error)
}

// Intent é a intenção destilada de uma mensagem do usuário.
type Intent struct {
	Seq      int64  `json:"seq"`
	Summary  string `json:"summary"`  // uma frase do que o usuário quer
	Category string `json:"category"` // enum (ver categorias válidas)
	Action   string `json:"action"`   // verbo normalizado (slot p/ a chave canônica)
	Object   string `json:"object"`   // objeto normalizado (slot p/ a chave canônica)

	// keyOverride é a chave canônica já materializada (lida do cache do store),
	// usada quando os slots Action/Object não estão disponíveis. Não serializa.
	keyOverride string
}

// WithKey devolve um Intent cuja Key() é fixada em k (para reconstruir a partir
// do cache do store, onde só a intent_key foi persistida).
func WithKey(seq int64, summary, category, k string) *Intent {
	return &Intent{Seq: seq, Summary: summary, Category: category, keyOverride: k}
}

// Categorias válidas para a intenção (o LLM é instruído a escolher uma).
var validCategories = map[string]bool{
	"criacao": true, "alteracao": true, "debug": true, "configuracao": true,
	"duvida": true, "exploracao": true, "refactor": true, "deploy": true, "outro": true,
}

// Key é a chave canônica da intenção: categoria|acao|objeto, tudo normalizado
// (minúsculas, sem acentos, sem pontuação). Duas mensagens diferentes que pedem
// a MESMA tarefa colapsam na mesma chave — é isso que faz a recorrência
// cross-session funcionar sem depender do texto exato do summary.
func (it *Intent) Key() string {
	if it == nil {
		return ""
	}
	if it.keyOverride != "" {
		return it.keyOverride
	}
	cat := norm(it.Category)
	if cat == "" || !validCategories[cat] {
		cat = "outro"
	}
	return cat + "|" + norm(it.Action) + "|" + norm(it.Object)
}

// Message é uma mensagem de usuário candidata à extração (par seq+texto).
type Message struct {
	Seq  int64
	Text string
}

// ExtractIntents extrai, em UMA ÚNICA chamada LLM (batch por sessão), a intenção
// de cada mensagem informada. Chamadores devem passar só os seqs ainda não
// presentes no cache (ver store.ListSessionIntents). Nunca chamada no modo
// heuristic_only (extração é exclusivamente LLM).
func ExtractIntents(ctx context.Context, h Headless, model string, msgs []Message) ([]*Intent, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	var b strings.Builder
	b.WriteString(`Extraia a INTENÇÃO do usuário em cada mensagem abaixo. Para cada uma, devolva:
- seq: o número da mensagem (repita o valor dado).
- summary: uma frase curta do que o usuário quer.
- category: uma de {criacao, alteracao, debug, configuracao, duvida, exploracao, refactor, deploy, outro}.
- action: o VERBO no infinitivo que resume a ação (ex.: criar, corrigir, listar, configurar). Uma palavra.
- object: o OBJETO principal da ação, no singular e genérico (ex.: rota, teste, deploy, componente). Uma ou duas palavras.
Normalize action/object para formas canônicas: "criar uma rota para listar usuários" e "criar rota GET /users" devem dar action=criar, object=rota.
Output APENAS um array JSON de objetos {seq, summary, category, action, object}.`)
	b.WriteString("\n\n## Mensagens\n")
	for _, m := range msgs {
		b.WriteString("### seq " + itoa(m.Seq) + "\n")
		b.WriteString(m.Text + "\n\n")
	}
	raw, err := h.RunHeadless(ctx, b.String(), adapter.HeadlessOpts{Model: model})
	if err != nil {
		return nil, err
	}
	return parseIntents(raw), nil
}

func parseIntents(raw string) []*Intent {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start < 0 || end <= start {
		return nil
	}
	var out []*Intent
	if json.Unmarshal([]byte(raw[start:end+1]), &out) != nil {
		return nil
	}
	// descarta entradas vazias
	clean := out[:0]
	for _, it := range out {
		if it != nil && (it.Summary != "" || it.Action != "" || it.Object != "") {
			clean = append(clean, it)
		}
	}
	return clean
}

// SimilarHeuristic reporta se duas intenções são a mesma tarefa, sem LLM. Base:
// mesma chave canônica (categoria|ação|objeto). É determinística e não depende
// de sobreposição lexical do summary — por isso "criar rota GET /users" e
// "criar uma rota para listar usuários" batem (mesma ação/objeto canônicos).
func SimilarHeuristic(a, b *Intent) bool {
	if a == nil || b == nil {
		return false
	}
	ka, kb := a.Key(), b.Key()
	if ka == "" || kb == "" {
		return false
	}
	return ka == kb
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
