import { useCallback, useEffect, useState } from 'react';
import { listProjects, listSessions } from '../api';
import type { Project, Session } from '../api';

export interface AppState {
  loading: boolean;
  projects: Project[];
  // Sessões iniciadas no app (Mode "wrapper"); são as navegáveis/duráveis.
  wrapperSessions: Session[];
  isEmpty: boolean;
  reload: () => void;
}

// useAppState carrega projetos e sessões e decide o estado macro do shell.
// isEmpty = nenhum projeto E nenhuma sessão wrapper → onboarding de primeiro uso.
export function useAppState(): AppState {
  const [loading, setLoading] = useState(true);
  const [projects, setProjects] = useState<Project[]>([]);
  const [wrapperSessions, setWrapperSessions] = useState<Session[]>([]);

  const reload = useCallback(() => {
    setLoading(true);
    Promise.all([listProjects(), listSessions()])
      .then(([projs, sessions]) => {
        setProjects(projs);
        setWrapperSessions(sessions.filter((s) => s.mode === 'wrapper'));
      })
      .catch(() => {
        setProjects([]);
        setWrapperSessions([]);
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  const isEmpty = projects.length === 0 && wrapperSessions.length === 0;
  return { loading, projects, wrapperSessions, isEmpty, reload };
}
