import { h, externalLink } from './dom.js';
import { meta, rule, sectionTitle } from './components.js';
import { buildEdition } from '../core/filter.js';
import { audienceTags, lawLabel } from '../core/audience.js';
import { profileSummary } from '../core/about.js';
import { countdownShort, dayOf, daysBetween, editionStamp, nextCaption, two } from '../core/dates.js';
import { ACTIONS } from '../core/filter.js';
import { infoTitle, topicTitle, countryTitle } from '../core/taxonomy.js';
import { regionTitle } from '../core/regions.js';

// Выпуск пересобирается только при изменении данных, профиля, настроек или дня: на экраны «сюжет» и «закон» он берётся из кэша.
let memo = { key: '', edition: null };

/** Выпуск для текущего состояния: сборка на устройстве. */
export function currentEdition(ctx) {
  const { state } = ctx.store;
  if (!state.feed) return null;
  const now = ctx.now();
  const today = dayOf(now);
  const prefs = ctx.store.effectivePrefs();
  const key = JSON.stringify([state.fetchedAt, state.feed.generated_at, state.laws.length, state.counter.number, state.profile, prefs, today]);
  if (memo.key === key) return memo.edition;
  const end = new Date(Math.max(Date.parse(state.feed.generated_at) || 0, now.getTime())).toISOString();
  const edition = buildEdition({
    number: Math.max(state.counter.number, 1), feed: state.feed, laws: state.laws, profile: state.profile, prefs,
    window: { start: null, end }, today,
  });
  memo = { key, edition };
  return edition;
}

const storyMeta = (story, theme) => {
  let text = theme === 'sage' ? `${topicTitle(story.topic)} · ${infoTitle(story.info_type).toLowerCase()}` : `${topicTitle(story.topic)} / ${infoTitle(story.info_type)}`;
  if (story.country && story.country !== 'RU') text += ` · ${countryTitle(story.country)}`;
  return text;
};

/** Меню «···» у сюжета: не интересно, меньше или больше про тему, скрыть источник. */
function storyMenu(ctx, story) {
  let open = null;
  const close = () => { open?.remove(); open = null; document.removeEventListener('click', onDoc, true); };
  const onDoc = (e) => { if (open && !open.contains(e.target) && !e.target.closest?.('.menu-btn')) close(); };
  const act = (action) => { close(); ctx.store.applyFeedback(action); };
  const interests = ctx.store.state.prefs.interests;
  const items = [
    ['Не интересно, скрыть сюжет', ACTIONS.hideStory(story.id)],
    [`Меньше про «${topicTitle(story.topic)}»`, ACTIONS.muteTopic(story.topic)],
    ...(interests.includes(story.topic) ? [] : [[`Больше про «${topicTitle(story.topic)}»`, ACTIONS.boostTopic(story.topic)]]),
    ...(story.sources ?? []).slice(0, 3).map((s) => [`Скрыть источник «${s.title}»`, ACTIONS.muteSource(s.title)]),
  ];
  const wrap = h('div', { style: 'position:relative' });
  const btn = h('button', { class: 'menu-btn', type: 'button', 'aria-label': 'Не интересно, ещё', 'aria-haspopup': 'menu', onClick: () => {
    if (open) return close();
    open = h('div', { class: 'menu', role: 'menu' }, items.map(([title, action]) => h('button', { type: 'button', role: 'menuitem', onClick: () => act(action) }, title)));
    wrap.append(open);
    setTimeout(() => document.addEventListener('click', onDoc, true));
  } }, '···');
  wrap.append(btn);
  return wrap;
}

export function storyRow(ctx, story, { interest = false } = {}) {
  const theme = ctx.theme;
  const meaning = story.meaning
    ? h('p', { class: 'meaning' }, h('b', null, theme === 'sage' ? 'Значит, ' : 'Значит: '), theme === 'sage' ? story.meaning.charAt(0).toLowerCase() + story.meaning.slice(1) : story.meaning)
    : null;
  return h('article', { class: 'story' },
    h('div', { class: 'head' }, interest ? h('span', { class: 'dot', 'aria-label': 'по вашим интересам' }) : null,
      h('span', { class: `meta grow${interest ? ' accent' : ''}` }, storyMeta(story, theme)), h('span', { class: 'meta' }, `${story.post_count} → 1`), storyMenu(ctx, story)),
    h('a', { class: 'st', href: `#/story/${encodeURIComponent(story.id)}` }, h('div', { class: 'st-title' }, story.title), meaning),
    story.sources?.length ? h('div', { class: 'srcs' }, story.sources.slice(0, 3).map((s) => externalLink(s.url, null, `${s.title} ↗`))) : null,
  );
}

function lawCard(ctx, law, tags, today) {
  const eff = law.dates?.effective ? { y: +law.dates.effective.slice(0, 4), m: +law.dates.effective.slice(5, 7), d: +law.dates.effective.slice(8, 10) } : null;
  const days = eff ? daysBetween(today, eff) : null;
  const status = { introduced: 'Внесён', passed: 'Принят', signed: 'Подписан', in_force: 'В силе' }[law.status] ?? '';
  const when = eff ? (law.status === 'in_force' || days <= 0 ? 'В силе' : `В силе с ${two(eff.d)}.${two(eff.m)}`) : status;
  return h('a', { class: 'law-card', href: `#/law/${encodeURIComponent(law.id)}` },
    h('div', { class: 'row' }, h('span', { class: 'meta accent' }, h('span', { class: 'dot', style: 'display:inline-block;margin-right:8px' }), when), days !== null ? h('span', { class: 'meta ink' }, countdownShort(days)) : null),
    h('div', { class: 't' }, law.title),
    h('div', { class: 'sub' }, `${lawLabel(law, tags)}${law.actions?.length ? ` · Сделать: ${law.actions[0].charAt(0).toLowerCase()}${law.actions[0].slice(1)}` : ''}`));
}

export function todayScreen(ctx) {
  const { state } = ctx.store;
  const theme = ctx.theme;
  const now = ctx.now();
  const e = currentEdition(ctx);
  if (!e) {
    return h('section', { class: 'screen' }, h('div', { class: 'empty' }, state.loadError ? state.loadError : 'Собираем выпуск…',
      state.loadError ? h('div', { style: 'margin-top:16px' }, h('button', { class: 'btn', type: 'button', onClick: () => ctx.refresh() }, h('span', null, 'Повторить'), h('span', null, '↻'))) : null));
  }
  const tags = audienceTags(state.profile);
  const today = dayOf(now);
  const strip = profileSummary(state.profile, state.prefs);
  const showAll = ctx.local.showAll;

  const stories = e.stories.map((s) => [rule(), storyRow(ctx, s, { interest: e.interestIds.has(s.id) })]);
  const hint = e.stats.trimmed > 0 && !showAll;

  return h('section', { class: 'screen' },
    state.offline ? h('div', { class: 'banner meta' }, '⚠ Нет связи · показан сохранённый выпуск') : null,
    h('div', { class: 'row meta', style: 'padding-top:14px' }, h('span', null, `Выпуск № ${e.number}`), h('span', null, editionStamp(now))),
    h('div', { class: 'row', style: 'align-items:flex-end' },
      h('h1', { class: 'title masthead' }, 'Штиль', h('span', { class: 'mark' }, '.')),
      h('a', { class: 'tap meta ink', href: '#/filters', style: 'text-decoration:none;gap:7px' }, h('span', { class: 'dot' }), state.prefs.calmMode ? 'Спокойно' : 'Обычный')),
    rule(true),
    h('div', { class: 'stats' },
      h('div', { class: 'stat' }, h('div', { class: 'v accent' }, two(e.stats.aboutYou)), meta('Про тебя')),
      h('div', { class: 'stat' }, h('div', { class: 'v' }, two(e.stats.stories)), meta('Сюжета')),
      h('div', { class: 'stat' }, h('div', { class: 'v' }, two(e.stats.readingMinutes)), meta('Мин чтения'))),
    rule(),
    strip ? h('a', { class: 'strip', href: '#/filters', style: 'text-decoration:none' }, h('span', { class: 'dot' }), h('span', { style: 'flex:1;font-weight:500;color:var(--body)' }, `Лента для: ${strip}`), h('span', { style: 'color:var(--accent)' }, 'Изменить')) : null,
    e.laws.length ? h('div', { class: 'plate' },
      h('div', { class: 'row', style: 'padding:14px 12px 6px' }, h('span', { class: 'meta accent' }, `${theme === 'sage' ? '' : '01 — '}Касается тебя`), h('a', { class: 'meta accent', href: '#/calendar', style: 'text-decoration:none;min-height:44px;display:inline-flex;align-items:center' }, 'Календарь →')),
      e.laws.map((l) => lawCard(ctx, l, tags, today))) : null,
    h('div', { style: 'margin-top:28px' }, theme === 'sage' ? h('h2', { class: 'title', style: 'font-size:26px' }, 'Главное за сутки') : h('h2', { class: 'meta ink' }, `${e.laws.length ? '02' : '01'} — Главное за сутки`)),
    stories.length ? stories : h('div', { class: 'empty' }, 'По вашим фильтрам сюжетов пока нет.'),
    hint ? h('p', { class: 'meta', style: 'padding:12px 0' }, `Показаны первые ${e.stories.length}. Ещё ${e.stats.trimmed} скрыто лимитом: его можно изменить в фильтрах.`) : null,
    e.foldedHeavy.length ? heavyBlock(ctx, e.foldedHeavy) : null,
    rule(),
    h('p', { class: 'meta', style: 'padding:16px 0;text-align:center' }, `Выпуск дочитан. Следующий — ${nextCaption(state.settings, now) ?? 'скоро'}`),
    e.stats.filteredOut > 0 ? h('p', { class: 'meta', style: 'text-align:center' }, `Скрыто фильтрами: ${e.stats.filteredOut}`) : null,
  );
}

function heavyBlock(ctx, stories) {
  let open = false;
  const box = h('div', { class: 'folded' });
  const draw = () => box.replaceChildren(
    rule(),
    h('button', { class: 'row more', type: 'button', onClick: () => { open = !open; draw(); } },
      h('span', { style: 'font-size:17px;font-weight:500' }, `Тяжёлые новости: ${stories.length}`), h('span', { class: 'meta accent' }, open ? 'Свернуть' : 'Показать →')),
    open ? stories.map((s) => [rule(), storyRow(ctx, s)]) : null);
  draw();
  return box;
}
export { regionTitle };
