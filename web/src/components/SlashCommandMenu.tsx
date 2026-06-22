// SlashCommandMenu: popover de autocomplete acionado por "/" no campo de prompt.
// Une três fontes — comandos "/" nativos do CLI, skills e agents do worrel — numa
// lista filtrável. É puramente apresentacional: o estado (abertura, item ativo,
// query) vive no componente pai; aqui só renderizamos a lista já filtrada e
// avisamos seleção/hover. Selecionar INSERE o texto no input (o usuário revisa e
// envia), espelhando o comportamento do Claude Code.

import type { SlashItem, SlashKind } from './slashCommands';

const KIND_TAG: Record<SlashKind, string> = {
  command: 'comando',
  skill: 'skill',
  agent: 'agent',
};

interface Props {
  items: SlashItem[]; // já filtrados pelo pai
  activeIndex: number;
  onSelect: (item: SlashItem) => void;
  onHover: (index: number) => void;
}

export default function SlashCommandMenu({ items, activeIndex, onSelect, onHover }: Props) {
  if (items.length === 0) return null;
  return (
    <div className="slash-menu" role="listbox">
      {items.map((it, i) => (
        <button
          key={it.label}
          type="button"
          role="option"
          aria-selected={i === activeIndex}
          className={`slash-item${i === activeIndex ? ' is-active' : ''}`}
          // onMouseDown (não onClick) para selecionar antes do textarea perder foco/blur.
          onMouseDown={(e) => { e.preventDefault(); onSelect(it); }}
          onMouseEnter={() => onHover(i)}
        >
          <span className="slash-item-main">
            <span className="slash-item-label">{it.label}</span>
            {it.description && <span className="slash-item-desc">{it.description}</span>}
          </span>
          <span className="slash-item-tag" data-kind={it.kind}>{KIND_TAG[it.kind]}</span>
        </button>
      ))}
    </div>
  );
}
