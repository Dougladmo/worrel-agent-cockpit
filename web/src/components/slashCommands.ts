// Tipos e lógica de filtro do menu de comandos "/". Mantidos fora do componente
// (SlashCommandMenu.tsx) para não quebrar o Fast Refresh, que exige que arquivos
// de componente só exportem componentes.

export type SlashKind = 'command' | 'skill' | 'agent';

export interface SlashItem {
  // label é o rótulo "/..." mostrado e usado para filtrar (ex.: "/sc:analyze").
  label: string;
  description: string;
  // insertText é o que vai pro campo de prompt ao selecionar.
  insertText: string;
  kind: SlashKind;
}

// filterSlashItems filtra por substring da query (texto após a "/"), comparando
// contra o label sem a barra inicial. Fonte única de verdade: o pai usa para
// navegar/selecionar e o menu para renderizar, evitando divergência de índices.
export function filterSlashItems(items: SlashItem[], query: string): SlashItem[] {
  const q = query.trim().toLowerCase();
  if (!q) return items;
  return items.filter((it) => it.label.replace(/^\//, '').toLowerCase().includes(q));
}
