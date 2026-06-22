import { useTranslation } from 'react-i18next';
import { setInteractionStyle } from '../interactionStyle';
import type { InteractionStyle } from '../interactionStyle';

interface Props {
  onChosen: () => void;
}

// InteractionStyleOnboarding pergunta UMA vez como o usuário quer responder ao
// agente: janela centralizada (modal) ou lateral (drawer).
export default function InteractionStyleOnboarding({ onChosen }: Props) {
  const { t } = useTranslation();
  const choose = (s: InteractionStyle) => { setInteractionStyle(s); onChosen(); };
  return (
    <div className="responder-overlay responder-center">
      <div className="onb-style">
        <h2>{t('onboardingStyle.title')}</h2>
        <p>{t('onboardingStyle.subtitle')}</p>
        <div className="onb-style-options">
          <button className="onb-style-opt" onClick={() => choose('modal')}>
            <span className="onb-style-art onb-style-modal"><i /></span>
            <b>{t('onboardingStyle.modal')}</b>
            <span>{t('onboardingStyle.modalDesc')}</span>
          </button>
          <button className="onb-style-opt" onClick={() => choose('drawer')}>
            <span className="onb-style-art onb-style-drawer"><i /></span>
            <b>{t('onboardingStyle.drawer')}</b>
            <span>{t('onboardingStyle.drawerDesc')}</span>
          </button>
        </div>
      </div>
    </div>
  );
}
