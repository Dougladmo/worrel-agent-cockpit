// SlashCommandMenu: popover de autocomplete do "/" no composer. Apresentacional —
// estado (abertura, item ativo, query) vive no pai; selecionar insere o texto no
// input, como no Claude Code.

import { useEffect, useRef } from 'react';
import type { SlashItem, SlashKind } from './slashCommands';

const KIND_TAG: Record<SlashKind, string> = {
  command: 'comando',
  skill: 'skill',
  agent: 'agent',
};

interface Props {
  items: SlashItem[];
  activeIndex: number; // seleção do teclado; o hover é só visual (CSS :hover)
  onSelect: (item: SlashItem) => void;
}

export default function SlashCommandMenu({ items, activeIndex, onSelect }: Props) {
  const menuRef = useRef<HTMLDivElement>(null);
  const activeRef = useRef<HTMLButtonElement>(null);

  // Mantém o item ativo visível rolando só o container — scrollIntoView poderia
  // rolar a página. offsetTop é relativo ao .slash-menu (offsetParent).
  useEffect(() => {
    const menu = menuRef.current;
    const item = activeRef.current;
    if (!menu || !item) return;
    const top = item.offsetTop;
    const bottom = top + item.offsetHeight;
    if (top < menu.scrollTop) menu.scrollTop = top;
    else if (bottom > menu.scrollTop + menu.clientHeight) menu.scrollTop = bottom - menu.clientHeight;
  }, [activeIndex]);

  if (items.length === 0) return null;
  return (
    <div className="slash-menu" role="listbox" ref={menuRef}>
      {items.map((it, i) => (
        <button
          key={it.label}
          ref={i === activeIndex ? activeRef : null}
          type="button"
          role="option"
          aria-selected={i === activeIndex}
          className={`slash-item${i === activeIndex ? ' is-active' : ''}`}
          // onMouseDown (não onClick) seleciona antes do textarea sofrer blur.
          onMouseDown={(e) => { e.preventDefault(); onSelect(it); }}
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
