import { useEffect, useState } from 'react';
import { getEngineHealth, setEngineConfigValue, type EngineHealth } from '../api';
import { useEvents } from '../useEvents';
import { HARNESSES, ModelPicker } from './HomeEngineConfig';

// Rótulo amigável por engine de IA. Cai no próprio id quando desconhecido.
const ENGINE_LABEL: Record<string, string> = {
  summary: 'Resumo de progresso',
  interpret: 'Interpretação para resposta',
};

const BANNER_CSS = `
.ehb { position: sticky; top: 0; z-index: 60; background: var(--fill-amber, rgba(224,138,60,0.14));
  border-bottom: 1px solid var(--orange, #e08a3c); color: var(--ink, inherit);
  padding: 10px 16px; font-size: 0.88rem; display: flex; flex-direction: column; gap: 8px; }
.ehb-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.ehb-warn { font-weight: 700; }
.ehb-muted { color: var(--muted); }
.ehb-btn { margin-left: auto; padding: 5px 11px; border-radius: 999px; cursor: pointer; font-size: 0.82rem;
  border: 1.5px solid var(--orange, #e08a3c); background: transparent; color: inherit; white-space: nowrap; }
.ehb-btn:hover { background: var(--orange, #e08a3c); color: #fff; }
.ehb-edit { display: grid; grid-template-columns: 88px 1fr; gap: 8px 12px; align-items: center;
  padding: 8px 0 2px; }
.ehb-pills { display: flex; gap: 6px; flex-wrap: wrap; }
.ehb-pill { padding: 5px 11px; border-radius: 999px; cursor: pointer; color: inherit; font-size: 0.8rem;
  border: 1.5px solid var(--line-strong, #3a3a3a); background: var(--surface-sunk, rgba(255,255,255,0.03)); }
.ehb-pill.on { border-color: var(--orange, #e08a3c); background: var(--fill-amber, rgba(224,138,60,0.18)); font-weight: 600; }
.ehb-input, .ec-input { width: 100%; max-width: 300px; padding: 7px 10px; border-radius: 8px;
  border: 1.5px solid var(--line-strong, #3a3a3a); background: var(--surface-sunk, rgba(255,255,255,0.03)); color: inherit; }
`;

function harnessLabel(v: string): string {
  return HARNESSES.find((h) => h.value === v)?.label ?? (v || 'padrão');
}

// Editor inline de harness/modelo de UM engine, gravando no escopo GLOBAL. O
// banner some sozinho quando a próxima geração desse engine der certo.
function EngineFix({ engine }: { engine: EngineHealth }) {
  const [harness, setHarness] = useState(engine.harness);
  const [model, setModel] = useState(engine.model);
  const pickHarness = (h: string) => {
    setHarness(h); setModel('');
    setEngineConfigValue(engine.engine_id, 'harness', h);
    setEngineConfigValue(engine.engine_id, 'model', '');
  };
  const pickModel = (m: string) => { setModel(m); setEngineConfigValue(engine.engine_id, 'model', m); };
  return (
    <div className="ehb-edit">
      <label style={{ fontWeight: 600 }}>Harness</label>
      <div className="ehb-pills">
        {HARNESSES.map((h) => (
          <button key={h.value} type="button" className={`ehb-pill${harness === h.value ? ' on' : ''}`}
            onClick={() => pickHarness(h.value)}>{h.label}</button>
        ))}
      </div>
      <label style={{ fontWeight: 600 }}>Modelo</label>
      <ModelPicker harness={harness} current={model} onSelect={pickModel} />
    </div>
  );
}

// EngineHealthBanner: barra fixa global que acende quando um engine de IA para
// de responder (provider fora/rate-limit → timeout). Permite trocar harness/
// modelo ali mesmo. Some quando o engine volta a responder (evento de bus).
export default function EngineHealthBanner() {
  const [engines, setEngines] = useState<EngineHealth[]>([]);
  const [open, setOpen] = useState<string | null>(null);

  const refresh = () => getEngineHealth().then(setEngines).catch(() => {});
  useEffect(() => { refresh(); }, []);
  useEvents((ev) => { if (ev.type === 'engine.health.changed') refresh(); }, refresh);

  if (engines.length === 0) return null;

  return (
    <div className="ehb">
      <style>{BANNER_CSS}</style>
      {engines.map((e) => {
        const label = ENGINE_LABEL[e.engine_id] ?? e.engine_id;
        const isOpen = open === e.engine_id;
        return (
          <div key={e.engine_id}>
            <div className="ehb-row">
              <span className="ehb-warn">⚠️ IA indisponível</span>
              <span>
                <strong>{label}</strong>{' '}
                <span className="ehb-muted">({harnessLabel(e.harness)}/{e.model || 'padrão'}) não está respondendo.</span>
              </span>
              <button type="button" className="ehb-btn"
                onClick={() => setOpen(isOpen ? null : e.engine_id)}>
                {isOpen ? 'Fechar' : 'Trocar modelo'}
              </button>
            </div>
            {isOpen && <EngineFix engine={e} />}
          </div>
        );
      })}
    </div>
  );
}
