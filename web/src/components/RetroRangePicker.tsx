import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { InventoryReport } from '../retroApi';

const DAY_MS = 24 * 60 * 60 * 1000;

export interface RangeValue {
  since: number; // ms epoch; 0 = colapsa para o mais antigo (sem limite inferior)
  until: number; // ms epoch; 0 = colapsa para o mais recente (sem limite superior)
}

// fmt formata um instante para data curta no fuso LOCAL (só leitura).
function fmt(ms: number): string {
  return new Date(ms).toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' });
}

interface Props {
  report: InventoryReport | null;
  excludedClis: Record<string, boolean>;
  value: RangeValue;
  onChange: (v: RangeValue) => void;
}

// RetroRangePicker: slider de intervalo ARRASTÁVEL (dois polegares) sobre o
// domínio real [mais antigo, mais recente] das sessões dos provedores liberados.
// Recalcula bounds/contagem quando os provedores são ativados/desativados.
export default function RetroRangePicker({ report, excludedClis, value, onChange }: Props) {
  const { t } = useTranslation();

  const { bounds, allMs } = useMemo(() => {
    const entries = Object.entries(report?.per_cli ?? {}).filter(([cli]) => !excludedClis[cli]);
    const ms: number[] = [];
    for (const [, ci] of entries) ms.push(...(ci.sessions_ms ?? []));
    if (ms.length === 0) return { bounds: null as null | { oldest: number; newest: number }, allMs: ms };
    let oldest = ms[0];
    let newest = ms[0];
    for (const m of ms) {
      if (m < oldest) oldest = m;
      if (m > newest) newest = m;
    }
    return { bounds: { oldest, newest }, allMs: ms };
  }, [report, excludedClis]);

  if (!bounds) {
    return <p className="retro-field-hint">{t('retro.wizard.rangeLocked')}</p>;
  }

  const span = Math.max(1, bounds.newest - bounds.oldest);
  const since = value.since || bounds.oldest;
  const until = value.until || bounds.newest;
  const count = allMs.filter((m) => m >= since && m <= until).length;
  const pct = (ms: number) => ((ms - bounds.oldest) / span) * 100;

  function setSince(ms: number) {
    const v = Math.min(ms, until);
    onChange({ since: v <= bounds!.oldest ? 0 : v, until: value.until });
  }
  function setUntil(ms: number) {
    const v = Math.max(ms, since);
    onChange({ since: value.since, until: v >= bounds!.newest ? 0 : v });
  }
  function preset(kind: 'all' | 'd30' | 'd7') {
    if (kind === 'all') return onChange({ since: 0, until: 0 });
    const days = kind === 'd30' ? 30 : 7;
    const lo = Math.max(bounds!.oldest, bounds!.newest - days * DAY_MS);
    onChange({ since: lo, until: 0 });
  }

  return (
    <div className="retro-range">
      <div className="retro-range-slider">
        <div className="retro-range-rail" />
        <div className="retro-range-fill" style={{ left: `${pct(since)}%`, right: `${100 - pct(until)}%` }} />
        <input
          type="range"
          className="retro-range-thumb"
          min={bounds.oldest}
          max={bounds.newest}
          step={DAY_MS}
          value={since}
          aria-label={t('retro.wizard.rangeFrom')}
          onChange={(e) => setSince(Number(e.target.value))}
        />
        <input
          type="range"
          className="retro-range-thumb"
          min={bounds.oldest}
          max={bounds.newest}
          step={DAY_MS}
          value={until}
          aria-label={t('retro.wizard.rangeTo')}
          onChange={(e) => setUntil(Number(e.target.value))}
        />
      </div>

      <div className="retro-range-labels">
        <span>{fmt(since)}</span>
        <span>{fmt(until)}</span>
      </div>

      <div className="retro-segmented" role="group" aria-label={t('retro.wizard.window')}>
        <button type="button" className="retro-seg" onClick={() => preset('all')}>{t('retro.wizard.all')}</button>
        <button type="button" className="retro-seg" onClick={() => preset('d30')}>30d</button>
        <button type="button" className="retro-seg" onClick={() => preset('d7')}>7d</button>
      </div>

      <div className="retro-range-count">
        {t('retro.wizard.rangeSelected')} <strong className="mono">{count}</strong> {t('retro.wizard.sessions')}
      </div>
    </div>
  );
}
