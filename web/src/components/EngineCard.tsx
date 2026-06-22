import ExecutionMode from './ExecutionMode'

export type ConfigOption = { value: string; label: string; description: string }
export type ConfigField = { key: string; label: string; type: string; default: string; options?: ConfigOption[] }
export type Spec = {
  id: string; name: string; description: string
  triggers: string[]; prompts: ConfigField[]; config: ConfigField[]
  output_type: string; default_on: boolean
}
export type EngineItem = { spec: Spec; config: Record<string, string> }

function OptionCards({ options, current, onSelect }: {
  options: ConfigOption[]
  current: string
  onSelect: (value: string) => void
}) {
  return (
    <div className="ec-cards">
      {options.map(o => (
        <button key={o.value} type="button"
          className={`ec-card${current === o.value ? ' on' : ''}`}
          onClick={() => onSelect(o.value)}>
          <b>{o.label}</b>
          <span>{o.description}</span>
        </button>
      ))}
    </div>
  )
}

export default function EngineCard({ item, setConfig, onRun }: {
  item: EngineItem
  setConfig: (id: string, key: string, value: string) => void
  onRun?: (id: string) => void
}) {
  const { spec, config } = item
  const enabled = config['__enabled'] === 'true'
  return (
    <div className="ec">
      <style>{EC_CSS}</style>

      <header className="ec-head">
        <div>
          <h3>{spec.name}</h3>
          <p>{spec.description}</p>
        </div>
        <button type="button" role="switch" aria-checked={enabled}
          className={`ec-switch${enabled ? ' on' : ''}`}
          onClick={() => setConfig(spec.id, '__enabled', enabled ? 'false' : 'true')}>
          <span className="ec-knob" />
        </button>
      </header>

      <fieldset className="ec-section" disabled={!enabled} style={{ opacity: enabled ? 1 : 0.5 }}>
        <div className="ec-field">
          <label>Quando executar</label>
          <ExecutionMode value={config['__trigger'] ?? spec.triggers[0]} allowed={spec.triggers}
            onChange={v => setConfig(spec.id, '__trigger', v)} />
        </div>

        {spec.config.map(f => {
          const current = config[f.key] ?? f.default
          return (
            <div key={f.key} className="ec-field">
              <label>{f.label}</label>
              {f.options && f.options.length > 0 ? (
                <OptionCards options={f.options} current={current} onSelect={v => setConfig(spec.id, f.key, v)} />
              ) : (
                <input className="ec-input" defaultValue={current} onBlur={e => setConfig(spec.id, f.key, e.target.value)} />
              )}
            </div>
          )
        })}

        {spec.prompts.map(f => (
          <div key={f.key} className="ec-field">
            <label>{f.label}</label>
            <textarea className="ec-textarea" defaultValue={config[f.key] ?? f.default}
              onBlur={e => setConfig(spec.id, f.key, e.target.value)} rows={4} />
          </div>
        ))}

        {onRun && (
          <div className="ec-run">
            <button type="button" className="btn btn-primary" onClick={() => onRun(spec.id)}>▶ Rodar agora</button>
            <span>Dispara o motor uma vez agora (modo sob demanda usa este botão).</span>
          </div>
        )}
      </fieldset>
    </div>
  )
}

const EC_CSS = `
.ec { max-width: 820px; margin: 0 auto 1.25rem; border: 1px solid var(--line-strong, #333); border-radius: 14px;
  background: var(--surface, rgba(255,255,255,0.015)); padding: 18px 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.10); }
.ec-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; }
.ec-head h3 { margin: 0; font-family: var(--display, inherit); font-size: 1.15rem; color: var(--ink, #eee); }
.ec-head p { margin: 4px 0 0; color: var(--muted, #999); font-size: 0.85rem; max-width: 56ch; }
.ec-switch { flex: none; width: 46px; height: 26px; border-radius: 999px; border: none; cursor: pointer;
  background: var(--line-strong, #444); position: relative; transition: background .2s ease; }
.ec-switch.on { background: var(--orange, #e08a3c); }
.ec-knob { position: absolute; top: 3px; left: 3px; width: 20px; height: 20px; border-radius: 50%; background: #fff;
  transition: transform .2s cubic-bezier(.3,1.4,.5,1); box-shadow: 0 1px 3px rgba(0,0,0,0.4); }
.ec-switch.on .ec-knob { transform: translateX(20px); }
.ec-section { border: none; margin: 0; padding: 16px 0 0; min-inline-size: auto; transition: opacity .25s ease; }
.ec-field { margin-bottom: 16px; }
.ec-field > label { display: block; margin-bottom: 8px; font-weight: 600; font-size: 0.9rem; color: var(--ink, #ddd); }
.ec-cards { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.ec-card { text-align: left; display: flex; flex-direction: column; gap: 4px; padding: 12px 13px; border-radius: 10px;
  cursor: pointer; color: inherit; border: 1.5px solid var(--line-strong, #3a3a3a);
  background: var(--surface-sunk, rgba(255,255,255,0.02)); transition: border-color .18s, background .18s, transform .18s, box-shadow .18s; }
.ec-card:hover { border-color: var(--orange, #e08a3c); background: var(--fill-amber, rgba(224,138,60,0.08)); transform: translateY(-2px); box-shadow: 0 6px 16px rgba(0,0,0,0.16); }
.ec-card.on { border-color: var(--orange, #e08a3c); background: var(--fill-amber, rgba(224,138,60,0.10)); box-shadow: inset 0 0 0 1px var(--orange, #e08a3c); }
.ec-card b { color: var(--ink, #eee); font-size: 0.9rem; }
.ec-card span { font-size: 0.78rem; color: var(--muted, #999); line-height: 1.35; }
.ec-input { width: 100%; max-width: 220px; padding: 8px 10px; border-radius: 8px; border: 1.5px solid var(--line-strong, #3a3a3a);
  background: var(--surface-sunk, rgba(255,255,255,0.02)); color: inherit; }
.ec-input:focus { outline: none; border-color: var(--orange, #e08a3c); }
.ec-textarea { width: 100%; padding: 10px 12px; border-radius: 8px; border: 1.5px solid var(--line-strong, #3a3a3a);
  background: var(--surface-sunk, rgba(255,255,255,0.02)); color: inherit; font-family: var(--mono, monospace); font-size: 0.8rem; line-height: 1.5; resize: vertical; }
.ec-textarea:focus { outline: none; border-color: var(--orange, #e08a3c); }
.ec-run { display: flex; align-items: center; gap: 12px; margin-top: 4px; }
.ec-run span { font-size: 0.78rem; color: var(--muted, #999); }
@media (max-width: 640px) { .ec-cards { grid-template-columns: 1fr; } }
`
