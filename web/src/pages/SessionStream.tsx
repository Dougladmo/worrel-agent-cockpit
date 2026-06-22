import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import {
  getInteraction, sendPrompt, respondInteraction, killSession,
  listSlashCommands, listSkills, listAgents, listActiveSessions,
} from '../api';
import type { InteractionSnapshot, HistoryLine } from '../api';
import { useEvents } from '../useEvents';
import { useDraft } from '../useDraft';
import { providerLabel } from '../session';
import SlashCommandMenu from '../components/SlashCommandMenu';
import { filterSlashItems } from '../components/slashCommands';
import type { SlashItem } from '../components/slashCommands';

// slashQueryOf devolve o texto digitado após "/" quando o prompt INTEIRO é um
// comando em construção (começa com "/" e ainda não tem espaço); senão null
// (menu fechado). Ex.: "/sc" → "sc"; "/sc:load x" → null; "oi" → null.
function slashQueryOf(text: string): string | null {
  const m = /^\/(\S*)$/.exec(text);
  return m ? m[1] : null;
}

// SessionStream é a "interface de terminal" de uma sessão dirigida pelo MOTOR
// (stream-json): não há PTY/xterm — mostramos o HISTÓRICO da conversa (você, IA,
// ferramentas, decisões), a permissão pendente e um campo para mandar prompts.
export default function SessionStream() {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [snap, setSnap] = useState<InteractionSnapshot | null>(null);
  // Rascunho persistido por sessão: navegar entre terminais não perde o texto.
  const [text, setText, clearDraft] = useDraft(id);
  const [busy, setBusy] = useState(false);
  const bodyRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Menu de comandos "/": fontes unificadas (comandos nativos do CLI + skills +
  // agents do worrel). menuQuery=null = fechado; senão é o texto após a "/".
  const [slashItems, setSlashItems] = useState<SlashItem[]>([]);
  const [menuQuery, setMenuQuery] = useState<string | null>(null);
  const [activeIndex, setActiveIndex] = useState(0);
  const filtered = useMemo(
    () => (menuQuery === null ? [] : filterSlashItems(slashItems, menuQuery)),
    [slashItems, menuQuery],
  );
  const menuOpen = menuQuery !== null && filtered.length > 0;

  // Carrega as fontes uma vez por sessão. Resolve projeto/working dir via
  // listActiveSessions; degrada para nível de usuário quando a sessão não é ativa.
  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    (async () => {
      let projectId = '';
      let dir = '';
      try {
        const s = (await listActiveSessions()).find((x) => x.id === id);
        if (s) { projectId = s.project_id || ''; dir = s.workspace_dir || ''; }
      } catch { /* sessão encerrada/sem contexto: segue só com nível de usuário */ }
      const [cmds, skills, agents] = await Promise.all([
        listSlashCommands('claude-code', dir).catch(() => []),
        listSkills(projectId || undefined).catch(() => []),
        projectId ? listAgents(projectId).catch(() => []) : Promise.resolve([]),
      ]);
      if (cancelled) return;
      setSlashItems([
        ...cmds.map((c): SlashItem => ({
          label: c.trigger, description: c.description,
          insertText: `${c.trigger} `, kind: 'command',
        })),
        ...skills.map((s): SlashItem => ({
          label: `/skill:${s.slug}`, description: s.name,
          insertText: `Aplique a skill "${s.name}" (${s.slug}). `, kind: 'skill',
        })),
        ...agents.map((a): SlashItem => ({
          label: `/agent:${a.slug}`, description: a.name,
          insertText: `Aja como o agente "${a.name}" (${a.slug}). `, kind: 'agent',
        })),
      ]);
    })();
    return () => { cancelled = true; };
  }, [id]);

  const load = useCallback(() => {
    if (!id) return;
    getInteraction(id).then(setSnap).catch(() => { /* ignore */ });
  }, [id]);

  useEffect(() => { load(); }, [load]);
  useEvents(useCallback((ev) => {
    const p = ev.payload as { session_id?: string; id?: string };
    if ((p?.session_id === id || p?.id === id) &&
      ['interaction.changed', 'session.awaiting', 'session.busy', 'session.ended'].includes(ev.type)) {
      load();
    }
  }, [id, load]));

  // rola para o fim quando o histórico cresce.
  useEffect(() => {
    bodyRef.current?.scrollTo({ top: bodyRef.current.scrollHeight });
  }, [snap?.history?.length, snap?.interrupt]);

  async function act(fn: () => Promise<unknown>) {
    if (busy) return;
    setBusy(true);
    try { await fn(); load(); } catch { /* noop */ } finally { setBusy(false); }
  }

  function submit() {
    if (!id || !text.trim()) return;
    const t = text.trim();
    clearDraft();
    act(() => sendPrompt(id, t));
  }

  function onChangeText(v: string) {
    setText(v);
    setMenuQuery(slashQueryOf(v));
    setActiveIndex(0);
  }

  function selectSlashItem(it: SlashItem) {
    setText(it.insertText);
    setMenuQuery(null);
    setActiveIndex(0);
    requestAnimationFrame(() => {
      const el = inputRef.current;
      if (el) { el.focus(); el.setSelectionRange(el.value.length, el.value.length); }
    });
  }

  // Teclado do composer: com o menu aberto, setas/Enter/Tab/Esc controlam a lista
  // (Enter NÃO envia); fechado, mantém o comportamento padrão (Enter envia).
  function onInputKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (menuOpen) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setActiveIndex((i) => (i + 1) % filtered.length); return; }
      if (e.key === 'ArrowUp') { e.preventDefault(); setActiveIndex((i) => (i - 1 + filtered.length) % filtered.length); return; }
      if (e.key === 'Enter' || e.key === 'Tab') { e.preventDefault(); selectSlashItem(filtered[activeIndex]); return; }
      if (e.key === 'Escape') { e.preventDefault(); setMenuQuery(null); return; }
    }
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(); }
  }

  const interrupt = snap?.interrupt;
  const isPermission = !!interrupt?.request_id;
  const state = snap?.state ?? 'working';
  const history = snap?.history ?? [];

  const reply = (value: string) => { if (id) act(() => sendPrompt(id, value)); };

  return (
    <div className="sstream">
      <header className="sstream-head">
        <button className="btn btn-secondary btn-sm" onClick={() => navigate('/')}>← {t('home.nav.home')}</button>
        <span className="ixp-state" data-state={state}>{t(`home.ix.state.${state}`)}</span>
        <span className="sstream-engine">{providerLabel('engine')}</span>
        <button className="btn btn-danger btn-sm" style={{ marginLeft: 'auto' }}
          onClick={() => id && act(() => killSession(id).then(() => navigate('/')))}>
          {t('terminal.kill')}
        </button>
      </header>

      <div className="sstream-body" ref={bodyRef}>
        {history.length === 0 && <div className="sstream-empty">{t('home.ix.working')}</div>}
        {history.map((h, i) => <ChatLine key={i} line={h} />)}
        {state === 'working' && history.length > 0 && (
          <div className="chat-thinking">{t('home.ix.working')}<span className="dots"><i /><i /><i /></span></div>
        )}
      </div>

      {isPermission ? (
        <div className="sstream-foot sstream-permission">
          <div className="sstream-perm-q">{interrupt!.prompt}</div>
          <div className="ixp-actions">
            <button className="btn btn-primary btn-sm" disabled={busy}
              onClick={() => id && act(() => respondInteraction(id, interrupt!.request_id, 'allow'))}>{t('ask.allow')}</button>
            <button className="btn btn-danger btn-sm" disabled={busy}
              onClick={() => id && act(() => respondInteraction(id, interrupt!.request_id, 'deny'))}>{t('ask.deny')}</button>
          </div>
        </div>
      ) : interrupt?.kind === 'choice' && interrupt.options?.length ? (
        <div className="sstream-foot sstream-permission">
          {interrupt.prompt && <div className="sstream-perm-q">{interrupt.prompt}</div>}
          <div className="ixp-actions ixp-options">
            {interrupt.options.map((opt) => (
              <button key={opt} className="btn btn-secondary btn-sm" disabled={busy} onClick={() => reply(opt)}>{opt}</button>
            ))}
          </div>
          <div className="sstream-foot" style={{ padding: 0, border: 'none', background: 'none' }}>
            <span className="nsw-prompt-glyph" aria-hidden="true">›</span>
            <textarea className="sstream-input" value={text} onChange={(e) => setText(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(); } }}
              placeholder={t('home.ix.promptPlaceholder')} rows={2} />
            <button className="btn btn-primary btn-sm" disabled={busy || !text.trim()} onClick={submit}>{t('home.ix.send')}</button>
          </div>
        </div>
      ) : (
        <div className="sstream-foot">
          {menuOpen && (
            <SlashCommandMenu
              items={filtered}
              activeIndex={activeIndex}
              onSelect={selectSlashItem}
              onHover={setActiveIndex}
            />
          )}
          <span className="nsw-prompt-glyph" aria-hidden="true">›</span>
          <textarea
            ref={inputRef}
            className="sstream-input"
            value={text}
            onChange={(e) => onChangeText(e.target.value)}
            onKeyDown={onInputKeyDown}
            placeholder={t('home.ix.promptPlaceholder')}
            rows={2}
            autoFocus
          />
          <button className="btn btn-primary btn-sm" disabled={busy || !text.trim()} onClick={submit}>{t('home.ix.send')}</button>
        </div>
      )}
    </div>
  );
}

// ChatLine renderiza uma linha do histórico como uma bolha de chat:
//   you → bolha do usuário (direita); ai → markdown renderizado (esquerda);
//   tool → linha discreta (mono); system → nota central.
function ChatLine({ line }: { line: HistoryLine }) {
  if (line.role === 'tool') {
    return <div className="chat-tool"><code>{line.text}</code></div>;
  }
  if (line.role === 'system') {
    return <div className="chat-system">{line.text}</div>;
  }
  if (line.role === 'you') {
    return <div className="chat-row chat-you"><div className="chat-bubble">{line.text}</div></div>;
  }
  // ai → markdown
  return (
    <div className="chat-row chat-ai">
      <div className="chat-bubble chat-md">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{line.text}</ReactMarkdown>
      </div>
    </div>
  );
}
