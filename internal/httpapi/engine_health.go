package httpapi

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/bus"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

// HealthEntry descreve um engine de IA headless que está falhando (provider fora,
// timeout, saída vazia). Alimenta o banner global "IA indisponível".
type HealthEntry struct {
	EngineID string `json:"engine_id"`
	Harness  string `json:"harness"` // harness resolvido no momento da falha ("" = default)
	Model    string `json:"model"`
	Kind     string `json:"kind"`  // timeout | empty | other
	LastErr  string `json:"last_error"`
	Since    int64  `json:"since"` // unix ms da 1ª falha da sequência atual
}

// engineHealth guarda, por engine, se ele está indisponível. fail/ok publicam o
// evento de mudança APENAS na borda (ok→falha e falha→ok), evitando ruído.
type engineHealth struct {
	mu    sync.Mutex
	state map[string]HealthEntry
}

func newEngineHealth() *engineHealth {
	return &engineHealth{state: map[string]HealthEntry{}}
}

// fail marca o engine como indisponível. Preserva Since se já estava falhando.
// Retorna true quando houve transição ok→falha (o chamador publica o evento).
func (h *engineHealth) fail(engineID, harness, model, kind, lastErr string, nowMs int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	prev, existed := h.state[engineID]
	since := nowMs
	if existed {
		since = prev.Since
	}
	h.state[engineID] = HealthEntry{
		EngineID: engineID, Harness: harness, Model: model,
		Kind: kind, LastErr: lastErr, Since: since,
	}
	return !existed
}

// ok limpa o engine. Retorna true quando houve transição falha→ok.
func (h *engineHealth) ok(engineID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, existed := h.state[engineID]; !existed {
		return false
	}
	delete(h.state, engineID)
	return true
}

// list devolve um snapshot das entradas atuais (engines falhando).
func (h *engineHealth) list() []HealthEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]HealthEntry, 0, len(h.state))
	for _, e := range h.state {
		out = append(out, e)
	}
	return out
}

// classifyLLMErr rotula o erro de RunHeadless para o health/banner. O opencode
// rate-limitado não emite 429 — ele trava e estoura o timeout do worrel; então
// DeadlineExceeded é o sinal real de "provider fora".
func classifyLLMErr(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "other"
}

// resolveHarnessModel resolve o harness/modelo configurados para um engine
// (sessão ⊕ global), no mesmo formato usado por summarizerFor. "" = default.
func (s *Server) resolveHarnessModel(engineID, sessionID string) (harness, model string) {
	get := func(key string) string {
		if s.deps.Store == nil {
			return ""
		}
		if sessionID != "" {
			if m, err := s.deps.Store.GetEngineConfig(engineID, "session:"+sessionID); err == nil {
				if v, ok := m[key]; ok && v != "" {
					return v
				}
			}
		}
		if m, err := s.deps.Store.GetEngineConfig(engineID, ""); err == nil {
			return m[key]
		}
		return ""
	}
	return get("harness"), get("model")
}

// noteEngineFail registra uma falha de geração de IA: audita (prompt + erro),
// marca o engine como indisponível e publica engine.health.changed na borda
// ok→falha. Uma linha por call site.
func (s *Server) noteEngineFail(engineID, sessionID, trigger, prompt, kind string, cause error) {
	if s.health == nil {
		return
	}
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	if s.deps.Store != nil {
		_ = s.deps.Store.LogEngineRun(&store.EngineLogEntry{
			EngineID: engineID, SessionID: sessionID, Trigger: trigger,
			Input: prompt, Output: "ERRO (" + kind + "): " + msg,
		})
	}
	harness, model := s.resolveHarnessModel(engineID, sessionID)
	if s.health.fail(engineID, harness, model, kind, msg, time.Now().UnixMilli()) && s.deps.Bus != nil {
		s.deps.Bus.Publish(bus.Event{Type: "engine.health.changed"})
	}
}

// noteEngineOk limpa o engine após uma geração bem-sucedida e publica o evento
// na borda falha→ok (o banner some).
func (s *Server) noteEngineOk(engineID string) {
	if s.health == nil {
		return
	}
	if s.health.ok(engineID) && s.deps.Bus != nil {
		s.deps.Bus.Publish(bus.Event{Type: "engine.health.changed"})
	}
}
