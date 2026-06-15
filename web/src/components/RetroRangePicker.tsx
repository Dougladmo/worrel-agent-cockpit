import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { InventoryReport } from '../retroApi';

const DAY_MS = 24 * 60 * 60 * 1000;

export interface RangeValue {
  since: number; // ms epoch; 0 = sem limite inferior
  until: number; // ms epoch; 0 = sem limite superior
}

// toDateInput converte ms epoch para "YYYY-MM-DD" no fuso LOCAL (input type=date).
// Usar componentes locais evita o drift de até 1 dia em fusos negativos (ex.: BRT),
// onde toISOString() (UTC) recuaria o dia exibido.
function toDateInput(ms: number): string {
  if (!ms) return '';
  const d = new Date(ms);
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${mm}-${dd}`;
}

// fromDateInput converte "YYYY-MM-DD" para ms epoch no início do dia LOCAL,
// consistente com toDateInput (mesma referência de fuso nos dois sentidos).
function fromDateInput(v: string): number {
  if (!v) return 0;
  const [y, m, d] = v.split('-').map(Number);
  return new Date(y, m - 1, d).getTime();
}

interface Props {
  report: InventoryReport | null;
  excludedClis: Record<string, boolean>;
  value: RangeValue;
  onChange: (v: RangeValue) => void;
}

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

  const since = value.since || bounds?.oldest || 0;
  const until = value.until || bounds?.newest || 0;

  const count = useMemo(
    () => allMs.filter((m) => m >= since && m <= until).length,
    [allMs, since, until],
  );

  if (!bounds) {
    return <div className="retro-inventory-panel muted">{t('retro.wizard.inventoryEmpty')}</div>;
  }

  function preset(kind: 'all' | 'd30' | 'd7') {
    if (!bounds) return;
    if (kind === 'all') return onChange({ since: 0, until: 0 });
    const days = kind === 'd30' ? 30 : 7;
    const lo = Math.max(bounds.oldest, bounds.newest - days * DAY_MS);
    onChange({ since: lo, until: bounds.newest });
  }

  return (
    <div className="retro-range">
      <div className="retro-range-bounds faint">
        <span>{toDateInput(bounds.oldest)} · {t('retro.wizard.rangeOldest')}</span>
        <span>{toDateInput(bounds.newest)} · {t('retro.wizard.rangeNewest')}</span>
      </div>

      <div className="retro-range-inputs">
        <label>
          {t('retro.wizard.rangeFrom')}
          <input
            type="date"
            min={toDateInput(bounds.oldest)}
            max={toDateInput(until || bounds.newest)}
            value={toDateInput(since)}
            onChange={(e) => onChange({ since: fromDateInput(e.target.value), until: value.until })}
          />
        </label>
        <label>
          {t('retro.wizard.rangeTo')}
          <input
            type="date"
            min={toDateInput(since || bounds.oldest)}
            max={toDateInput(bounds.newest)}
            value={toDateInput(until)}
            onChange={(e) => onChange({ since: value.since, until: fromDateInput(e.target.value) })}
          />
        </label>
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
