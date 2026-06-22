import type { ReactNode } from 'react';
import { getInteractionStyle } from '../interactionStyle';

interface Props {
  onClose: () => void;
  children: ReactNode;
}

// ResponderShell apresenta a janela de resposta ao agente como MODAL
// (centralizado) ou DRAWER (lateral), conforme a preferência do onboarding.
export default function ResponderShell({ onClose, children }: Props) {
  const style = getInteractionStyle();
  if (style === 'drawer') {
    return (
      <div className="responder-overlay" onClick={onClose}>
        <div className="responder-drawer" onClick={(e) => e.stopPropagation()}>{children}</div>
      </div>
    );
  }
  return (
    <div className="responder-overlay responder-center" onClick={onClose}>
      <div className="responder-modal" onClick={(e) => e.stopPropagation()}>{children}</div>
    </div>
  );
}
