import { useState } from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { FanMark } from '../components/Fan';
import { ProviderBadge } from '../session';
import { killSession, archiveSession, postHandoff } from '../api';
import type { Project, Session } from '../api';

interface Props {
  projects: Project[];
  // Todas as sessões wrapper (vivas + encerradas) e o conjunto das vivas.
  sessions: Session[];
  liveIds: Set<string>;
  awaitingIds: Set<string>;
  // onChanged recarrega o estado da app após uma ação (encerrar/arquivar) para
  // que a sidebar reflita o novo estado da sessão.
  onChanged: () => void;
}

// AppNav é a navegação principal. O item "terminals" não navega: expande, no
// próprio sidebar, a lista de terminais ATIVOS (agrupados por projeto, incluindo
// "sem projeto") e um HISTÓRICO das sessões encerradas. Cada item ganha as ações
// por estado: vivas → Encerrar; encerradas → Recomeçar / Arquivar.
export default function AppNav({ projects, sessions, liveIds, awaitingIds, onChanged }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [open, setOpen] = useState(true);
  const [busy, setBusy] = useState(false);
  // Alvos dos modais de confirmação (a ação destrutiva só acontece após confirmar).
  const [killTarget, setKillTarget] = useState<Session | null>(null);
  const [archiveTarget, setArchiveTarget] = useState<Session | null>(null);

  const live = sessions.filter((s) => liveIds.has(s.id));
  const ended = sessions.filter((s) => !liveIds.has(s.id));
  const byProject = (pid: string) => live.filter((s) => s.project_id === pid);
  const orphans = live.filter((s) => !s.project_id);

  // run serializa as ações (evita duplo clique) e recarrega o estado ao final.
  async function run(fn: () => Promise<unknown>) {
    if (busy) return;
    setBusy(true);
    try { await fn(); onChanged(); } catch { /* preserva o último estado bom */ } finally { setBusy(false); }
  }

  // Recomeçar: cria uma nova sessão encadeada (handoff) herdando o contexto da
  // encerrada e abre o terminal dela.
  function handleResume(sessionId: string) {
    return run(async () => {
      const r = await postHandoff(sessionId);
      navigate(`/sessions/${r.new_id}`);
    });
  }

  function handleKill(sessionId: string) {
    return run(async () => {
      await killSession(sessionId);
      setKillTarget(null);
    });
  }

  function handleArchive(sessionId: string) {
    return run(async () => {
      await archiveSession(sessionId);
      setArchiveTarget(null);
    });
  }

  function item(s: Session, isLive: boolean) {
    const name = s.title?.trim() || t('terminals.untitled', 'Sessão');
    const time = s.started_at
      ? new Date(s.started_at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
      : '';
    return (
      <div key={s.id} className="appnav-term-wrap">
        <NavLink to={`/sessions/${s.id}`}
          className={`appnav-term${awaitingIds.has(s.id) ? ' needs-attention' : ''}${isLive ? ' live' : ''}`}>
          <span className="appnav-term-top">
            <ProviderBadge adapter={s.adapter} />
            <span className="appnav-term-time">{time}</span>
          </span>
          <span className="appnav-term-name">
            {isLive && <span className="appnav-live-dot" aria-hidden="true" />}
            <span className="appnav-term-label">{name}</span>
          </span>
        </NavLink>
        <span className="appnav-term-actions">
          {isLive ? (
            <button
              className="appnav-term-act"
              disabled={busy}
              title={t('sessions.endHint', 'Encerra o processo do agente desta sessão') as string}
              aria-label={t('sessions.end', 'Encerrar') as string}
              onClick={() => setKillTarget(s)}
            >⨯</button>
          ) : (
            <>
              <button
                className="appnav-term-act"
                disabled={busy}
                title={t('sessions.resumeHint') as string}
                aria-label={t('sessions.resume') as string}
                onClick={() => handleResume(s.id)}
              >↻</button>
              <button
                className="appnav-term-act"
                disabled={busy}
                title={t('sessions.archive') as string}
                aria-label={t('sessions.archive') as string}
                onClick={() => setArchiveTarget(s)}
              >🗄</button>
            </>
          )}
        </span>
      </div>
    );
  }

  function group(name: string, list: Session[]) {
    if (list.length === 0) return null;
    return (
      <div className="appnav-term-group" key={name}>
        <div className="appnav-term-proj">{name}</div>
        {list.map((s) => item(s, true))}
      </div>
    );
  }

  return (
    <aside className="appnav">
      <div className="appnav-brand">
        <FanMark size={22} />
        Worrel
      </div>
      <nav className="appnav-links">
        <NavLink to="/" end className="appnav-link">{t('home.nav.home')}</NavLink>
        <NavLink to="/projects" className="appnav-link">{t('home.nav.projects')}</NavLink>
        <span className="appnav-link disabled">{t('home.nav.metrics')}</span>
        <span className="appnav-link disabled">{t('home.nav.joystick')}</span>
        <span className="appnav-link disabled">{t('home.nav.lab')}</span>

        <button className={`appnav-link strong appnav-toggle${open ? ' open' : ''}`}
          onClick={() => setOpen((v) => !v)} aria-expanded={open}>
          {t('home.nav.terminals')}
          {live.length > 0 && <span className="appnav-term-count">{live.length}</span>}
        </button>

        {open && (
          <div className="appnav-terms">
            {live.length === 0 && <div className="appnav-term-empty">{t('terminals.empty')}</div>}
            {group(t('home.wizard.noProject'), orphans)}
            {projects.map((p) => group(p.name, byProject(p.id)))}

            {ended.length > 0 && (
              <div className="appnav-term-group">
                <div className="appnav-term-proj appnav-term-history">{t('terminals.history')}</div>
                {ended.slice(0, 8).map((s) => item(s, false))}
              </div>
            )}
          </div>
        )}
      </nav>
      <div className="appnav-foot">
        <NavLink to="/settings" className="appnav-link">{t('nav.settings')}</NavLink>
      </div>

      {killTarget && (
        <div className="modal-overlay" onClick={() => !busy && setKillTarget(null)}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="appnav-kill-title"
            onClick={(e) => e.stopPropagation()}>
            <h3 id="appnav-kill-title" style={{ marginTop: 0 }}>{t('sessions.endConfirmTitle', 'Encerrar sessão em andamento?')}</h3>
            <p>{t('sessions.endConfirmMsg', 'O processo do agente será finalizado. A sessão fica no histórico e pode ser recomeçada depois.')}</p>
            <div style={{ display: 'flex', gap: '1rem', marginTop: '1.5rem' }}>
              <button className="btn btn-secondary" style={{ flex: 1 }} disabled={busy} onClick={() => setKillTarget(null)}>
                {t('common.cancel')}
              </button>
              <button className="btn btn-primary" style={{ flex: 1 }} disabled={busy} onClick={() => handleKill(killTarget.id)}>
                {t('sessions.end', 'Encerrar')}
              </button>
            </div>
          </div>
        </div>
      )}

      {archiveTarget && (
        <div className="modal-overlay" onClick={() => !busy && setArchiveTarget(null)}>
          <div className="modal" role="dialog" aria-modal="true" aria-labelledby="appnav-archive-title"
            onClick={(e) => e.stopPropagation()}>
            <h3 id="appnav-archive-title" style={{ marginTop: 0 }}>{t('sessions.archiveConfirmTitle')}</h3>
            <p>{t('sessions.archiveConfirmMsg')}</p>
            <div style={{ display: 'flex', gap: '1rem', marginTop: '1.5rem' }}>
              <button className="btn btn-secondary" style={{ flex: 1 }} disabled={busy} onClick={() => setArchiveTarget(null)}>
                {t('common.cancel')}
              </button>
              <button className="btn btn-primary" style={{ flex: 1 }} disabled={busy} onClick={() => handleArchive(archiveTarget.id)}>
                {t('sessions.archive')}
              </button>
            </div>
          </div>
        </div>
      )}
    </aside>
  );
}
