export const story = (over = {}) => ({
  id: 's1', topic: 'economy', info_type: 'fact', heaviness: 'neutral', title: 'Заголовок', meaning: null, summary: 'Краткое содержание',
  post_count: 4, source_count: 2, sources: [], region_code: null, country: null, updated_at: '2026-09-25T01:00:00Z', ...over,
});
export const law = (over = {}) => ({
  id: 'l1', title: 'Закон', what_changed: 'Что', who_affected: 'Кого', actions: [], audience_tags: ['all'], region_code: null, status: 'signed',
  dates: { effective: '2026-10-01' }, official_url: null, bill_url: null, act_number: null, verified_at: null, ...over,
});
export { DEFAULT_PREFS, DEFAULT_PROFILE } from '../js/core/taxonomy.js';
