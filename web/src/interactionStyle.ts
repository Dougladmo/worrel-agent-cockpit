// Preferência de como a janela de resposta ao agente aparece: modal (centralizado)
// ou drawer (lateral). Escolhida no onboarding, salva localmente. Padrão: modal.
export type InteractionStyle = 'modal' | 'drawer';

const KEY = 'worrel.interactionStyle';

export function getInteractionStyle(): InteractionStyle {
  return (localStorage.getItem(KEY) as InteractionStyle) || 'modal';
}

export function setInteractionStyle(s: InteractionStyle): void {
  localStorage.setItem(KEY, s);
}

// hasChosenInteractionStyle indica se o usuário já escolheu (para o onboarding).
export function hasChosenInteractionStyle(): boolean {
  return localStorage.getItem(KEY) != null;
}
