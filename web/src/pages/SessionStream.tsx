import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
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

// A saída de comandos locais (/usage) usa quebras de linha simples, que o markdown
// colapsa num parágrafo só. Convertê-las em hard breaks preserva as linhas sem
// afetar tabelas (ex.: /context já vem como tabela markdown).
const withHardBreaks = (text: string) => text.replace(/\n/g, '  \n');

// ChatLine renderiza uma linha do histórico como uma bolha de chat:
//   you → bolha do usuário (direita); ai → markdown renderizado (esquerda);
//   tool → linha discreta (mono); system → nota central;
//   command → painel com a saída formatada de um slash command (/usage, /context…).
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
  if (line.role === 'command') {
    return <div className="chat-command"><CommandOutput text={line.text} /></div>;
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

// Comandos como /context já voltam em markdown (tabelas, headings); os de dados
// crus (/usage, /config) são texto plano e ganham estrutura própria.
const looksLikeMarkdown = (text: string) =>
  /\|\s*:?-{3,}/.test(text) || /^#{1,6}\s/m.test(text) || /\*\*[^*\n]+\*\*/.test(text);

type CmdRow =
  | { kind: 'gap' }
  | { kind: 'lead'; text: string }
  | { kind: 'section'; text: string }
  | { kind: 'meter'; label: string; pct: number; meta?: string }
  | { kind: 'kv'; key: string; value: string }
  | { kind: 'stat'; pct: string; text: string }
  | { kind: 'text'; text: string };

// classifyCommandOutput estrutura a saída de dados linha a linha por padrão (não
// por comando): "N% used" vira medidor, "key=value" vira par, e o resto cai em
// texto — então comandos desconhecidos degradam para linhas limpas, sem quebrar.
function classifyCommandOutput(text: string): CmdRow[] {
  const rows: CmdRow[] = [];
  text.split('\n').forEach((raw, i) => {
    const line = raw.trim();
    if (!line) {
      if (rows.length && rows[rows.length - 1].kind !== 'gap') rows.push({ kind: 'gap' });
      return;
    }
    const meter = line.match(/^(.*?):\s*(\d+)%\s+used(?:\s*·\s*(.*))?$/);
    if (meter) { rows.push({ kind: 'meter', label: meter[1].trim(), pct: +meter[2], meta: meter[3]?.trim() }); return; }
    const kv = line.match(/^([A-Za-z0-9_]+)=(.+)$/);
    if (kv) { rows.push({ kind: 'kv', key: kv[1], value: kv[2] }); return; }
    if (i === 0) { rows.push({ kind: 'lead', text: line }); return; }
    if (line.endsWith('?') || /^Last \d/.test(line)) { rows.push({ kind: 'section', text: line.replace(/\?$/, '') }); return; }
    const stat = line.match(/^(\d+)%\s+(.+)$/);
    if (stat) { rows.push({ kind: 'stat', pct: stat[1], text: stat[2] }); return; }
    rows.push({ kind: 'text', text: line });
  });
  return rows;
}

const spacedOptions = (value: string) => value.replace(/\|/g, ' | ');

function CommandOutput({ text }: { text: string }) {
  if (looksLikeMarkdown(text)) {
    return (
      <div className="chat-command-body chat-md">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{withHardBreaks(text)}</ReactMarkdown>
      </div>
    );
  }
  const rows = classifyCommandOutput(text);
  const blocks: ReactNode[] = [];
  for (let i = 0; i < rows.length; ) {
    // Agrupa key=value consecutivos num grid único para alinhar as chaves.
    if (rows[i].kind === 'kv') {
      const pairs: { key: string; value: string }[] = [];
      while (i < rows.length && rows[i].kind === 'kv') { pairs.push(rows[i] as { key: string; value: string }); i++; }
      blocks.push(
        <div className="cmd-kv-grid" key={`kv${i}`}>
          {pairs.flatMap((p, k) => [
            <span key={`k${k}`} className="cmd-kv-key">{p.key}</span>,
            <span key={`v${k}`} className="cmd-kv-val">{spacedOptions(p.value)}</span>,
          ])}
        </div>,
      );
      continue;
    }
    const r = rows[i];
    const key = `r${i}`;
    if (r.kind === 'gap') blocks.push(<div className="cmd-gap" key={key} />);
    else if (r.kind === 'lead') blocks.push(<p className="cmd-lead" key={key}>{r.text}</p>);
    else if (r.kind === 'section') blocks.push(<p className="cmd-section" key={key}>{r.text}</p>);
    else if (r.kind === 'stat') blocks.push(<p className="cmd-stat" key={key}><b>{r.pct}%</b> {r.text}</p>);
    else if (r.kind === 'meter') blocks.push(
      <div className="cmd-meter" key={key}>
        <div className="cmd-meter-top"><span className="cmd-meter-label">{r.label}</span><span className="cmd-meter-pct">{r.pct}% used</span></div>
        <div className="cmd-meter-track"><div className="cmd-meter-fill" style={{ width: `${Math.min(r.pct, 100)}%` }} /></div>
        {r.meta && <div className="cmd-meter-meta">{r.meta}</div>}
      </div>,
    );
    else if (r.kind === 'text') blocks.push(<p className="cmd-text" key={key}>{r.text}</p>);
    i++;
  }
  return <div className="chat-command-body cmd-report">{blocks}</div>;
}
