export type ConfigOption = { value: string; label: string; description: string }
export type ConfigField = { key: string; label: string; type: string; default: string; options?: ConfigOption[] }
export type Spec = {
  id: string; name: string; description: string
  triggers: string[]; prompts: ConfigField[]; config: ConfigField[]
  output_type: string; default_on: boolean
}
export type EngineItem = { spec: Spec; config: Record<string, string> }

// Descrição de cada gatilho (modo de execução). Universal aos motores.
const TRIGGER_INFO: Record<string, { label: string; description: string }> = {
  project_open_close: {
    label: 'Automático',
    description: 'Analisa cada sessão assim que ela encerra. Ao abrir o app, processa sessões que fecharam sem análise (recuperação). Varre em segundo plano a cada ~2 min.',
  },
  periodic: {
    label: 'Periódico',
    description: 'Varre em segundo plano de tempos em tempos, processando as sessões encerradas ainda não analisadas.',
  },
  realtime: {
    label: 'Ao vivo (em breve)',
    description: 'Analisaria durante a sessão, em tempo real. Ainda não disponível.',
  },
  on_demand: {
    label: 'Sob demanda',
    description: 'Só roda quando você clica em "Rodar". Nada é analisado automaticamente.',
  },
}

function OptionCards({ options, current, onSelect }: {
  options: { value: string; label: string; description: string }[]
  current: string
  onSelect: (value: string) => void
}) {
  return (
    <div style={{ display: 'grid', gap: '0.4rem' }}>
      {options.map(o => {
        const sel = current === o.value
        return (
          <button key={o.value} type="button" onClick={() => onSelect(o.value)}
            style={{
              textAlign: 'left', cursor: 'pointer', padding: '0.5rem 0.65rem', borderRadius: '6px',
              border: sel ? '2px solid var(--green)' : '1px solid var(--border, #444)',
              background: sel ? 'rgba(80,200,120,0.10)' : 'transparent',
              color: 'inherit',
            }}>
            <div style={{ fontWeight: 600 }}>{o.label}{sel ? ' ✓' : ''}</div>
            <div style={{ fontSize: '0.8rem', color: 'var(--muted)' }}>{o.description}</div>
          </button>
        )
      })}
    </div>
  )
}

export default function EngineCard({ item, setConfig, onRun }: {
  item: EngineItem
  setConfig: (id: string, key: string, value: string) => void
  onRun?: (id: string) => void
}) {
  const { spec, config } = item
  const triggerOpts = spec.triggers.map(t => ({ value: t, ...(TRIGGER_INFO[t] ?? { label: t, description: '' }) }))
  return (
    <div className="card" style={{ maxWidth: '760px', marginBottom: '1rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>{spec.name}</strong>
        <label style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
          <input
            type="checkbox"
            checked={config['__enabled'] === 'true'}
            onChange={e => setConfig(spec.id, '__enabled', e.target.checked ? 'true' : 'false')}
          /> ativo
        </label>
      </div>
      <p style={{ marginTop: '0.4rem', color: 'var(--muted)' }}>{spec.description}</p>

      {triggerOpts.length > 0 && (
        <div style={{ marginBottom: '0.75rem' }}>
          <label style={{ display: 'block', marginBottom: '0.35rem' }}>Quando executar</label>
          <OptionCards options={triggerOpts} current={config['__trigger'] ?? spec.triggers[0]}
            onSelect={v => setConfig(spec.id, '__trigger', v)} />
        </div>
      )}

      {spec.config.map(f => {
        const current = config[f.key] ?? f.default
        return (
          <div key={f.key} style={{ marginBottom: '0.75rem' }}>
            <label style={{ display: 'block', marginBottom: '0.35rem' }}>{f.label}</label>
            {f.options && f.options.length > 0 ? (
              <OptionCards options={f.options} current={current} onSelect={v => setConfig(spec.id, f.key, v)} />
            ) : (
              <input defaultValue={current} onBlur={e => setConfig(spec.id, f.key, e.target.value)} />
            )}
          </div>
        )
      })}

      {spec.prompts.map(f => (
        <div key={f.key} style={{ marginBottom: '0.75rem' }}>
          <label style={{ display: 'block', marginBottom: '0.2rem' }}>{f.label}</label>
          <textarea defaultValue={config[f.key] ?? f.default} onBlur={e => setConfig(spec.id, f.key, e.target.value)}
            rows={4} style={{ width: '100%', fontFamily: 'var(--mono)', fontSize: '0.8rem' }} />
        </div>
      ))}
      {onRun && <button className="btn btn-primary" onClick={() => onRun(spec.id)}>Rodar sob demanda</button>}
    </div>
  )
}
