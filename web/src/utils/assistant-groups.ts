/**
 * Assistant type & grouping — pure helpers, zero module dependencies. Kept dependency-free on purpose: pages in the
 * router's static import graph (ai-assistant list) use these without dragging the service/request → store chain into
 * their module init order.
 */

export interface KbAssistantOption {
  id: string;
  name: string;
  /** pipeline architecture: simple / deep_think / deep_research / data_processing */
  type?: string;
}

/** the builtin knowledge-management assistant seeded by the wiki module */
export const BUILTIN_KM_ASSISTANT_ID = 'builtin-wiki-km';

export type KbAssistantGroup = 'chat' | 'data_processing' | 'deep_research' | 'deep_think' | 'search';

/**
 * usage group of an assistant: its type decides — data_processing assistants power the wiki pipelines, search
 * integrations tag their own, the rest are user-facing
 */
export function assistantGroupOf(a: KbAssistantOption, searchBoundIds?: Set<string>): KbAssistantGroup {
  if (a.type === 'data_processing') return 'data_processing';
  if (searchBoundIds?.has(a.id)) return 'search';
  if (a.type === 'deep_research') return 'deep_research';
  if (a.type === 'deep_think') return 'deep_think';
  return 'chat';
}

/**
 * collect assistant ids referenced anywhere inside search integration records (ai_overview.assistant,
 * deep_think_assistant, deep_research_assistant, ai_chat.assistants, …)
 */
export function collectSearchBoundAssistantIds(integrations: unknown[]): Set<string> {
  const ids = new Set<string>();
  const walk = (value: any, key = '') => {
    if (Array.isArray(value)) {
      value.forEach(item => walk(item, key));
      return;
    }
    if (value && typeof value === 'object') {
      for (const [k, v] of Object.entries(value)) {
        if (k === 'assistants' && Array.isArray(v)) {
          v.forEach(item => {
            const id = typeof item === 'string' ? item : item?.id;
            if (id) ids.add(id);
          });
        } else {
          walk(v, k);
        }
      }
      return;
    }
    const k = key.toLowerCase();
    if (typeof value === 'string' && value && (k === 'assistant' || k.endsWith('_assistant'))) {
      ids.add(value);
    }
  };
  integrations.forEach(i => walk(i));
  return ids;
}

/** KB-bound assistants as grouped select options — empty groups dropped */
export function groupKbAssistantOptions(
  list: KbAssistantOption[],
  t: (key: string) => string,
  searchBoundIds?: Set<string>
): { label: string; options: { value: string; label: string }[] }[] {
  const toOption = (a: KbAssistantOption) => ({ value: a.id, label: a.name });
  const order: { key: KbAssistantGroup; labelKey: string }[] = [
    { key: 'chat', labelKey: 'page.wiki.assistantGroup.chat' },
    { key: 'search', labelKey: 'page.wiki.assistantGroup.search' },
    { key: 'deep_think', labelKey: 'page.wiki.assistantGroup.deepThink' },
    { key: 'deep_research', labelKey: 'page.wiki.assistantGroup.deepResearch' },
    { key: 'data_processing', labelKey: 'page.wiki.assistantGroup.dataProcessing' }
  ];
  return order
    .map(({ key, labelKey }) => ({
      label: t(labelKey),
      options: list.filter(a => assistantGroupOf(a, searchBoundIds) === key).map(toOption)
    }))
    .filter(group => group.options.length > 0);
}
