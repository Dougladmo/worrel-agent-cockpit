// Seletor do modo de execução de um motor (grava em __trigger). Cards simples
// com ícone representativo + descrição. Estilos escopados em .exmode-*.

type Mode = { value: string; label: string; desc: string; icon: string }

const MODES: Mode[] = [
  {
    value: 'project_open_close',
    label: 'Ao encerrar a sessão',
    desc: 'Destila quando a sessão termina. Ao abrir o app, recupera as que fecharam sem análise.',
    icon: '🏁',
  },
  {
    value: 'realtime',
    label: 'Ao vivo',
    desc: 'Destila durante a sessão, revisitando as sessões em andamento periodicamente.',
    icon: '🔴',
  },
  {
    value: 'agent_self',
    label: 'O agente decide',
    desc: 'Injeta a regra no início; o próprio agente registra (via MCP) quando percebe algo.',
    icon: '🤖',
  },
  {
    value: 'on_demand',
    label: 'Sob demanda',
    desc: 'Nada automático: você dispara com o botão “Rodar agora”.',
    icon: '▶️',
  },
]

export default function ExecutionMode({ value, onChange, allowed }: {
  value: string
  onChange: (v: string) => void
  allowed?: string[] // valores suportados pelo motor; os demais ficam indisponíveis
}) {
  return (
    <div className="exmode">
      <style>{EXMODE_CSS}</style>
      <div className="exmode-grid">
        {MODES.map(m => {
          const supported = !allowed || allowed.includes(m.value)
          const on = value === m.value
          return (
            <button
              key={m.value}
              type="button"
              className={`exmode-opt${on ? ' on' : ''}${!supported ? ' off' : ''}`}
              disabled={!supported}
              onClick={() => supported && onChange(m.value)}
            >
              <span className="exmode-icon" aria-hidden>{m.icon}</span>
              <div className="exmode-text">
                <b>{m.label}</b>
                <span>{m.desc}</span>
              </div>
              {!supported && <span className="exmode-badge">indisponível</span>}
            </button>
          )
        })}
      </div>
    </div>
  )
}

const EXMODE_CSS = `
.exmode-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.exmode-opt {
  position: relative; display: flex; align-items: flex-start; gap: 11px;
  text-align: left; padding: 12px 13px; border-radius: 10px; cursor: pointer;
  border: 1.5px solid var(--line-strong, #3a3a3a); background: var(--surface-sunk, rgba(255,255,255,0.02));
  color: inherit; transition: border-color .16s ease, background .16s ease, transform .16s ease, box-shadow .16s ease;
}
.exmode-opt:hover:not(:disabled) { border-color: var(--orange, #e08a3c); background: var(--fill-amber, rgba(224,138,60,0.08)); transform: translateY(-1px); box-shadow: 0 4px 14px rgba(0,0,0,0.14); }
.exmode-opt.on { border-color: var(--orange, #e08a3c); background: var(--fill-amber, rgba(224,138,60,0.10)); box-shadow: inset 0 0 0 1px var(--orange, #e08a3c); }
.exmode-opt.off { opacity: 0.5; cursor: not-allowed; }
.exmode-icon { font-size: 1.35rem; line-height: 1.2; flex: none; }
.exmode-text { display: flex; flex-direction: column; gap: 3px; }
.exmode-opt b { color: var(--ink, #eee); font-size: 0.92rem; }
.exmode-opt .exmode-text span { font-size: 0.78rem; color: var(--muted, #999); line-height: 1.35; }
.exmode-badge { position: absolute; top: 9px; right: 9px; font-size: 0.6rem; letter-spacing: .04em; text-transform: uppercase;
  padding: 2px 7px; border-radius: 999px; background: var(--line-strong, #444); color: var(--muted, #aaa); }
@media (max-width: 640px) { .exmode-grid { grid-template-columns: 1fr; } }
`
