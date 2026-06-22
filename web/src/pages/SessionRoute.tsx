import { useParams } from 'react-router-dom';
import type { Session } from '../api';
import Terminal from './Terminal';
import SessionStream from './SessionStream';

// SessionRoute decide a "visão de sessão" pelo tipo: sessões do MOTOR
// (adapter "engine") não têm PTY → mostram a conversa estruturada
// (SessionStream); as demais abrem o terminal xterm (Terminal).
export default function SessionRoute({ sessions }: { sessions: Session[] }) {
  const { id } = useParams<{ id: string }>();
  const sess = sessions.find((s) => s.id === id);
  if (sess?.adapter === 'engine') return <SessionStream />;
  return <Terminal />;
}
