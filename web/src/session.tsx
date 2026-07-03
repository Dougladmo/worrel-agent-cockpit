import type { Session } from './api';

// ============================================================
// Nome e provider da sessão
//
// O provider (adapter) é informação fixa e categórica → vira BADGE.
// O nome deve refletir O QUE ESTÁ ACONTECENDO, não o instante de criação:
//   1. title explícito (sessões observadas herdam do 1º recado);
//   2. 1ª linha útil do summary (preenchido pela análise/handoff);
//   3. fallback discreto pela hora de início (sem repetir o provider,
//      que já aparece na badge ao lado).
// ============================================================

function startTime(s: Session): string {
  return s.started_at
    ? new Date(s.started_at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
    : '';
}

// primeira linha significativa do summary, sem marcação markdown, truncada.
function summaryLine(summary: string): string | null {
  for (const raw of summary.split('\n')) {
    const line = raw.replace(/^#+\s*/, '').replace(/^[-*]\s*/, '').replace(/[*_`]/g, '').trim();
    if (line.length >= 4) {
      return line.length > 64 ? line.slice(0, 63).trimEnd() + '…' : line;
    }
  }
  return null;
}

// sessionName devolve o melhor rótulo legível para a sessão.
export function sessionName(s: Session): string {
  if (s.title?.trim()) return s.title.trim();
  if (s.summary?.trim()) {
    const line = summaryLine(s.summary);
    if (line) return line;
  }
  const time = startTime(s);
  return time ? `Sessão · ${time}` : s.id.slice(0, 8);
}

// cor estável por provider, derivada do nome (sem depender de uma lista fechada).
const PROVIDER_CLASS: Record<string, string> = {
  'claude-code': 'sky',
  claude: 'sky',
  opencode: 'amber',
  antigravity: 'green',
  gemini: 'green', // mantido p/ render de sessões antigas (legado, sem custo)
  codex: 'pink',
  pidev: 'amber',
};

// Rótulo amigável por provider.
const PROVIDER_LABEL: Record<string, string> = {
  'claude-code': 'Claude Code',
};

// isEngine indica que a sessão é dirigida pelo motor stream-json. O adapter é
// um marcador INTERNO de roteamento: "engine" (legado) ou "engine:<provider>".
export function isEngine(adapter: string): boolean {
  return adapter === 'engine' || adapter.startsWith('engine:');
}

// resolveProvider extrai o provider REAL de um adapter. Para sessões de motor,
// vem no sufixo ("engine:opencode"); "engine" cru (legado) assume claude-code —
// que era o único driver quando essas sessões foram criadas.
function resolveProvider(adapter: string): string {
  if (adapter === 'engine') return 'claude-code';
  if (adapter.startsWith('engine:')) return adapter.slice('engine:'.length) || 'claude-code';
  return adapter;
}

// providerLabel devolve o nome legível do provider (cai no próprio provider
// quando não há mapeamento específico).
export function providerLabel(adapter: string): string {
  if (!adapter) return '';
  const p = resolveProvider(adapter);
  return PROVIDER_LABEL[p.toLowerCase()] ?? p;
}

// ProviderBadge exibe o provider como badge categórica (com nome amigável).
export function ProviderBadge({ adapter }: { adapter: string }) {
  if (!adapter) return null;
  const cls = PROVIDER_CLASS[resolveProvider(adapter).toLowerCase()] ?? '';
  return <span className={`pill provider-badge ${cls}`.trim()}>{providerLabel(adapter)}</span>;
}
