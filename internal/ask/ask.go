// Package ask arbitra pedidos de confirmação/escolha que vão para a UI (balões).
// Estado em memória, não persiste. Espelha internal/vault.Broker, mas a resposta
// é uma string (decisão "allow"/"deny" ou a escolha/texto do usuário).
package ask

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Request é o envelope publicado no bus (ask.requested) e listado em /api/asks/pending.
type Request struct {
	ID           string   `json:"request_id"`
	SessionID    string   `json:"session_id"`
	SessionLabel string   `json:"session_label"`
	Kind         string   `json:"kind"` // "permission" | "choice"
	Title        string   `json:"title"`
	Detail       string   `json:"detail,omitempty"`
	Options      []string `json:"options"`
}

type entry struct {
	req Request
	ch  chan string
}

// Broker guarda pedidos pendentes por request_id.
type Broker struct {
	mu      sync.Mutex
	pending map[string]*entry
}

func New() *Broker { return &Broker{pending: map[string]*entry{}} }

// Open registra um pedido (gera o ID) e devolve (request preenchido, canal de resposta).
func (b *Broker) Open(req Request) (Request, chan string) {
	req.ID = uuid.NewString()
	ch := make(chan string, 1)
	b.mu.Lock()
	b.pending[req.ID] = &entry{req: req, ch: ch}
	b.mu.Unlock()
	return req, ch
}

// Resolve responde um pedido pendente; false se o id não existir mais.
func (b *Broker) Resolve(id, answer string) bool {
	b.mu.Lock()
	e, ok := b.pending[id]
	if ok {
		delete(b.pending, id)
	}
	b.mu.Unlock()
	if !ok {
		return false
	}
	e.ch <- answer // canal bufferizado (cap 1): nunca bloqueia
	return true
}

// Wait bloqueia até Resolve (devolve answer, true) ou cancelamento do contexto
// (devolve "", false). Espera indefinidamente enquanto o ctx estiver vivo.
func (b *Broker) Wait(ctx context.Context, ch chan string) (string, bool) {
	select {
	case answer := <-ch:
		return answer, true
	case <-ctx.Done():
		return "", false
	}
}

// Remove descarta um pedido sem respondê-lo; devolve (req, true) se existia.
func (b *Broker) Remove(id string) (Request, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e, ok := b.pending[id]
	if !ok {
		return Request{}, false
	}
	delete(b.pending, id)
	return e.req, true
}

// Pending devolve um snapshot dos pedidos abertos (para re-hidratar a UI).
func (b *Broker) Pending() []Request {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Request, 0, len(b.pending))
	for _, e := range b.pending {
		out = append(out, e.req)
	}
	return out
}
