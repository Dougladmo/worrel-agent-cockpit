import { useTranslation } from 'react-i18next';
import { NavLink } from 'react-router-dom';
import { FanMark } from '../components/Fan';
import type { Project, Session } from '../api';

interface Props {
  projects: Project[];
  wrapperSessions: Session[];
  onNewSession: (projectId: string) => void;
  onAnalyzeHistory: () => void;
}

// ProjectSidebar lista projetos; sob cada um, suas sessões wrapper navegáveis.
// Cada projeto tem a ação ＋ (nova sessão já vinculada).
export default function ProjectSidebar({ projects, wrapperSessions, onNewSession, onAnalyzeHistory }: Props) {
  const { t } = useTranslation();
  const byProject = (pid: string) => wrapperSessions.filter((s) => s.project_id === pid);

  return (
    <aside className="sidebar">
      <div className="sidebar-title">
        <FanMark size={22} />
        Worrel
      </div>
      <div className="sidebar-section">{t('sidebar.projects')}</div>

      <nav className="sidebar-projects">
        {projects.map((p) => (
          <div key={p.id} className="sidebar-project">
            <div className="sidebar-project-head">
              <NavLink to={`/projects/${p.id}`} className="sidebar-project-name">{p.name}</NavLink>
              <button
                className="sidebar-new-btn"
                aria-label={t('sidebar.newSessionIn', { name: p.name })}
                onClick={() => onNewSession(p.id)}
              >＋</button>
            </div>
            <div className="sidebar-sessions">
              {byProject(p.id).map((s) => (
                <NavLink key={s.id} to={`/sessions/${s.id}`} className="sidebar-session">
                  {s.title || s.id.slice(0, 8)}
                </NavLink>
              ))}
            </div>
          </div>
        ))}
      </nav>

      <div className="sidebar-foot">
        <button className="btn btn-secondary" style={{ width: '100%' }} onClick={onAnalyzeHistory}>
          {t('onboarding.analyzeHistory')}
        </button>
      </div>
    </aside>
  );
}
