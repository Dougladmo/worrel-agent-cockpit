import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { listSuggestions, acceptSuggestion, rejectSuggestion } from '../api';
import type { Suggestion } from '../api';
import SuggestionBody from '../components/SuggestionBody';

interface Props {
  // Projeto ativo (terminal/projeto aberto). Vazio = sem escopo de projeto.
  activeProjectId: string | null;
  // Sinal para recarregar (ex.: evento suggestion.created).
  reloadKey: number;
}

// SuggestionsDrawer fica sempre visível (recolhível). Mostra sugestões globais
// (sem project_id) fixas + as do projeto ativo, que trocam conforme a navegação.
export default function SuggestionsDrawer({ activeProjectId, reloadKey }: Props) {
  const { t } = useTranslation();
  const [collapsed, setCollapsed] = useState(false);
  const [all, setAll] = useState<Suggestion[]>([]);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    listSuggestions(undefined, 'pending').then(setAll).catch(() => setAll([]));
  }, []);
  useEffect(() => { load(); }, [load, reloadKey]);

  const globals = all.filter((s) => !s.project_id);
  const scoped = activeProjectId ? all.filter((s) => s.project_id === activeProjectId) : [];
  // O badge reflete só o que está VISÍVEL no escopo atual (globais + projeto
  // ativo); o resto vira a nota "em outros projetos" para nada sumir em silêncio.
  const visible = globals.length + scoped.length;
  const others = all.length - visible;

  async function act(id: string, fn: (id: string) => Promise<unknown>) {
    if (busy) return;
    setBusy(true);
    try {
      await fn(id);
      setAll((prev) => prev.filter((s) => s.id !== id));
    } finally {
      setBusy(false);
    }
  }

  if (collapsed) {
    return (
      <aside className="drawer drawer-collapsed">
        <button className="drawer-toggle" aria-label={t('drawer.expand')} onClick={() => setCollapsed(false)}>
          ‹ {visible > 0 && <span className="badge">{visible}</span>}
        </button>
      </aside>
    );
  }

  function renderItem(sg: Suggestion) {
    return (
      <div key={sg.id} className="card drawer-card">
        <strong>{sg.title}</strong>
        <SuggestionBody sg={sg} />
        <div className="drawer-card-actions">
          <button className="btn btn-primary btn-sm" disabled={busy} onClick={() => act(sg.id, acceptSuggestion)}>
            {t('suggestions.accept')}
          </button>
          <button className="btn btn-secondary btn-sm" disabled={busy} onClick={() => act(sg.id, rejectSuggestion)}>
            {t('suggestions.reject')}
          </button>
        </div>
      </div>
    );
  }

  return (
    <aside className="drawer">
      <div className="drawer-head">
        <span>{t('nav.suggestions')}</span>
        {visible > 0 && <span className="badge">{visible}</span>}
        <button className="drawer-toggle" aria-label={t('drawer.collapse')} onClick={() => setCollapsed(true)}>›</button>
      </div>

      <div className="drawer-body">
        {activeProjectId && (
          <>
            <div className="drawer-section-label">{t('drawer.scoped')}</div>
            {scoped.length === 0 ? <p className="muted drawer-empty">{t('drawer.none')}</p> : scoped.map(renderItem)}
          </>
        )}
        <div className="drawer-section-label">{t('drawer.global')}</div>
        {globals.length === 0 ? <p className="muted drawer-empty">{t('drawer.none')}</p> : globals.map(renderItem)}
        {others > 0 && <p className="muted drawer-empty">{t('drawer.others', { n: others })}</p>}
      </div>
    </aside>
  );
}
