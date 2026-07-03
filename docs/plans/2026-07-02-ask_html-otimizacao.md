# Plano (v2, revisado pós-review do fable) — ask_html instantâneo + deferral polishing

> ⚠️ Plano de implementação temporário. Delete-o após concluir.
>
> **Mudança de rumo (v2):** a v1 propunha um `BuildAskHTML` estático em Go. O review
> apontou que isso (a) não compilava (`agui.Session` não existe, roles reais são
> `you/ai/tool/system`), (b) **suprimia os botões de opção clicáveis** — o valor real
> do ask_html eram os `data-choice`, e sua presença faz o `InteractionPanel` esconder
> os botões de choice interpretados (`attachInterpretation`) — e (c) inflava o payload
> duplicando o histórico. **Solução correta e mais simples: remover o ask_html por
> completo.** Os fallbacks já existentes (markdown de `expects` + botões de choice do
> `attachInterpretation` + input livre) entregam a resposta instantânea, com menos
> código e melhor UX. Também descartada a lógica de "defer batch move para o fim da
> fila" (Task 9 da v1): criava loop de modais. Batch = enfileira todos; defer sai da
> fila mas continua como bolinha (não se perde).

**Goal:** (1) Tornar o `ask_html` instantâneo removendo a geração via LLM. (2) Corrigir
sessões deferidas: sumir ao encerrar, agrupar bolinhas por projeto, backdrop-click adia
(não descarta), clique num grupo abre todas em fila.

**Tech Stack:** Go (stdlib). React 19/TS, Vite. `go build/test`, `npx tsc -b`.

## Global Constraints
- Comentários e copy em **português**.
- Manter os campos `AskHTML`/`AskHTMLPending`/`ResponseWidget` do `Snapshot` (contrato
  JSON) e o render de `ask_html` no `InteractionPanel` — viram código morto inofensivo
  (o backend nunca mais seta, então os branches nunca disparam).

---

### Task 1: Remover geração de ask_html (backend)

**Files:**
- Delete: `internal/agui/ask_html.go`, `internal/agui/ask_html_test.go`
- Delete: `internal/httpapi/interaction_ask_html.go`
- Modify: `internal/httpapi/interaction.go` — remover as 2 chamadas `s.attachAskHTML(&snap)` (linhas 121, 139)
- Modify: `internal/httpapi/server.go` — remover campo `askHTML` (45) e init `askHTML: newAskHTMLCache()` (52)

**Verificação:** `agui.AskHTML`/`AskHTMLPrompt`/`ParseAskHTML`/`expectsOf`/`askHTMLTimeout`
só são referenciados por esses arquivos. `ResponseWidget` (tipo) e os campos do Snapshot
ficam.

- [ ] Deletar os 3 arquivos.
- [ ] Remover as 2 chamadas em `interaction.go`.
- [ ] Remover campo + init em `server.go`.
- [ ] `go build ./... && go test ./internal/agui/ ./internal/httpapi/` → PASS.
- [ ] Commit: `refactor(ask_html): remove geracao via LLM (fallback markdown ja e instantaneo)`

---

### Task 2: Limpar `deferred_at` ao encerrar sessão

**Files:** `internal/store/sessions.go`

- [ ] `EndSession` (114): `SET status='ended', ended_at=?, deferred_at=NULL`
- [ ] `EndSessionWithReason` (134-137): acrescentar `, deferred_at=NULL`
- [ ] `EndOrphanedWrapperSessions` (343-344): acrescentar `, deferred_at=NULL`
- [ ] `go build ./...` → PASS.
- [ ] Commit: `fix(deferred): limpar deferred_at ao encerrar sessao`

---

### Task 3: Filtrar deferidas por `status='active'`

**Files:** `internal/store/deferred.go` (51)

- [ ] Trocar `status != 'archived'` por `status = 'active'` na query de `ListDeferredSessions`.
- [ ] `go build ./...` → PASS.
- [ ] Commit: `fix(deferred): listar so status=active`

---

### Task 4: Agrupar bolinhas por projeto + abrir em fila

**Files:** `web/src/shell/SuggestionsDrawer.tsx`, `web/src/App.tsx`

- [ ] `App.tsx`: novo `handleOpenDeferred(ids: string[])` — enfileira todos (dedup) em
  `modalQueue`; o efeito de auto-open existente abre um a um. Passar como `onOpen` ao drawer
  (linha 330). (A bolinha deferida é pedido explícito → não passa por `enqueueAutoOpen`.)
- [ ] `SuggestionsDrawer.tsx`: `Props.onOpen` vira `(sessionIds: string[]) => void`. Agrupar
  `deferred` por `project_id`; renderizar uma bolinha por projeto com badge de contagem se
  >1. Preservar `nameOf`/`initials`/`data-kind`/`title`. `onClick` → `onOpen(sessions.map(s => s.session_id))`.
- [ ] `cd web && npx tsc -b` → PASS.
- [ ] Commit: `feat(deferred): agrupar baloes por projeto + abrir em fila`

---

### Task 5: Backdrop-click adia (não descarta)

**Files:** `web/src/components/ResponderShell.tsx`, `web/src/components/GlobalInteractionModal.tsx`

- [ ] `ResponderShell`: prop opcional `onBackdropClick?`. No overlay, `onClick={onBackdropClick ?? onClose}`.
- [ ] `GlobalInteractionModal`: `handleBackdropDefer` = `try { await deferSession(sessionId) } catch {} ; onClose()`.
  Passar `onBackdropClick={handleBackdropDefer}`. Importar `deferSession`.
- [ ] `cd web && npx tsc -b && npm run build` → PASS.
- [ ] Commit: `fix(modal): backdrop adia em vez de descartar`

---

### Task 6: CSS do badge de contagem + grupo

**Files:** `web/src/styles.css`

- [ ] `.deferred-dot-count` (badge circular) e ajuste em `.deferred-dot` para posicionar.
- [ ] `cd web && npm run build` → PASS.
- [ ] Commit: `style: badge de contagem para grupo deferido`
