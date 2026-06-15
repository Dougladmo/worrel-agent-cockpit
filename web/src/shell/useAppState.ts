import { useCallback, useEffect, useState } from 'react';
import { listProjects, listSessions, listActiveSessions } from '../api';
import type { Project, Session } from '../api';

export interface AppState {
  loading: boolean;
  projects: Project[];
  // Sessões iniciadas no app (Mode "wrapper"); base para grouping e isEmpty.
  wrapperSessions: Session[];
  // IDs das sessões com processo vivo (bar "Ativas"). A sidebar mostra só estas.
  liveIds: Set<string>;
  isEmpty: boolean;
  reload: () => void;
}

// useAppState carrega projetos, sessões e as sessões vivas; decide o estado
// macro do shell. isEmpty = nenhum projeto E nenhuma sessão wrapper → onboarding.
export function useAppState(): AppState {
  const [loading, setLoading] = useState(true);
  const [projects, setProjects] = useState<Project[]>([]);
  const [wrapperSessions, setWrapperSessions] = useState<Session[]>([]);
  const [liveIds, setLiveIds] = useState<Set<string>>(new Set());

  const reload = useCallback(() => {
    setLoading(true);
    Promise.all([listProjects(), listSessions(), listActiveSessions()])
      .then(([projs, sessions, active]) => {
        setProjects(projs);
        setWrapperSessions(sessions.filter((s) => s.mode === 'wrapper'));
        setLiveIds(new Set(active.map((s) => s.id)));
      })
      .catch(() => {
        setProjects([]);
        setWrapperSessions([]);
        setLiveIds(new Set());
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  const isEmpty = projects.length === 0 && wrapperSessions.length === 0;
  return { loading, projects, wrapperSessions, liveIds, isEmpty, reload };
}
