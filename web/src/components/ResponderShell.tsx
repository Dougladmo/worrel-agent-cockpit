import type { ReactNode } from 'react';
import { getInteractionStyle } from '../interactionStyle';

interface Props {
  onClose: () => void;
  // Clique no backdrop (fora do modal). Se ausente, cai em onClose. A Home usa
  // para ADIAR a sessão (não perder a pergunta) em vez de só fechar.
  onBackdropClick?: () => void;
  children: ReactNode;
}

// ResponderShell apresenta a janela de resposta ao agente como MODAL
// (centralizado) ou DRAWER (lateral), conforme a preferência do onboarding.
export default function ResponderShell({ onClose, onBackdropClick, children }: Props) {
  const style = getInteractionStyle();
  const onBackdrop = onBackdropClick ?? onClose;
  if (style === 'drawer') {
    return (
      <div className="responder-overlay" onClick={onBackdrop}>
        <div className="responder-drawer" onClick={(e) => e.stopPropagation()}>{children}</div>
      </div>
    );
  }
  return (
    <div className="responder-overlay responder-center" onClick={onBackdrop}>
      <div className="responder-modal" onClick={(e) => e.stopPropagation()}>{children}</div>
    </div>
  );
}
