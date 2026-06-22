import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  listProjects,
  listSuggestions,
  listSessions,
} from '../api';
import type { Project, Session } from '../api';
import { sessionName, ProviderBadge } from '../session';
import { FanHero } from '../components/Fan';
import NewProjectModal from '../components/NewProjectModal';

interface Props {
  onPendingCount: (n: number) => void;
}

export default function Dashboard({ onPendingCount }: Props) {
  const { t } = useTranslation();
  const [projects, setProjects] = useState<Project[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [pendingMap, setPendingMap] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [error, setError] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const [ps, ss, sugs] = await Promise.all([
          listProjects(),
          listSessions(),
          listSuggestions(undefined, 'pending'),
        ]);
        if (cancelled) return;
        setProjects(ps);
        setSessions(ss.filter((s) => s.status === 'active'));
        const map: Record<string, number> = {};
        sugs.forEach((sg) => {
          map[sg.project_id] = (map[sg.project_id] ?? 0) + 1;
        });
        setPendingMap(map);
        onPendingCount(sugs.length);
      } catch {
        if (!cancelled) setError(true);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [reloadKey, onPendingCount]);

  if (loading) return <div className="main"><p>{t('common.loading')}</p></div>;

  return (
    <div className="main">
      <div className="page-head">
        <div>
          <h1>{t('dashboard.title')}</h1>
          <p className="sub">{t('dashboard.subtitle')}</p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setShowModal(true)}>
            {t('dashboard.newProject')}
          </button>
        </div>
      </div>

      {error && <p className="error-banner">{t('common.actionFailed')}</p>}

      {projects.length === 0 ? (
        <div className="empty">
          <FanHero width={132} height={68} />
          <h2>{t('dashboard.noProjects')}</h2>
          <p>{t('dashboard.noProjectsHint')}</p>
          <button className="btn btn-accent" style={{ marginTop: 18 }} onClick={() => setShowModal(true)}>
            {t('dashboard.newProject')}
          </button>
        </div>
      ) : (
        <div className="grid">
          {projects.map((p) => (
            <Link key={p.id} to={`/projects/${p.id}`} className="card clickable">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 8 }}>
                <h3 style={{ margin: 0, color: 'var(--ink)' }}>{p.name}</h3>
                {(pendingMap[p.id] ?? 0) > 0 && (
                  <span className="badge">{pendingMap[p.id]}</span>
                )}
              </div>
              <p className="muted" style={{ margin: '8px 0 0', fontSize: '0.875rem' }}>
                {p.description || '—'}
              </p>
            </Link>
          ))}
        </div>
      )}

      {sessions.length > 0 && (
        <>
          <h2 style={{ marginTop: 36 }}>{t('dashboard.activeSessions')}</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {sessions.map((s) => (
              <Link key={s.id} to={`/sessions/${s.id}`} className="card clickable" style={{ padding: '12px 16px' }}>
                <strong style={{ color: 'var(--ink)' }}>{sessionName(s)}</strong>
                <span style={{ marginLeft: 12, display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                  <ProviderBadge adapter={s.adapter} />
                  <span className="mono muted" style={{ fontSize: '0.8rem' }}>{s.mode}</span>
                </span>
              </Link>
            ))}
          </div>
        </>
      )}

      {showModal && (
        <NewProjectModal
          onClose={() => setShowModal(false)}
          onCreated={() => {
            setShowModal(false);
            setReloadKey((k) => k + 1);
          }}
        />
      )}
    </div>
  );
}
