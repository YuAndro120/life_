import { h } from './dom.js';
import { meta, rule, screenTitle, switchRow } from './components.js';
import { relevantLaws, lawLabel, audienceTags, lawOrder } from '../core/audience.js';
import { isWarLaw } from '../core/markers.js';
import { calendarTile, compareDays, countdownShort, dayOf, daysBetween, longDay, numeric, parseDay } from '../core/dates.js';

/** Календарь: подписанные законы, которые ещё не вступили в силу, по дате вступления. */
export function calendarScreen(ctx) {
  const { state } = ctx.store;
  const theme = ctx.theme;
  const today = dayOf(ctx.now());
  const onlyMine = ctx.local.onlyMine ?? true;
  const hideWar = state.prefs.hideWar;
  const signed = state.laws.filter((l) => l.status === 'signed' && parseDay(l.dates?.effective) && !(hideWar && isWarLaw(l)));
  const laws = onlyMine ? relevantLaws(signed, state.profile, today) : signed.filter((l) => compareDays(parseDay(l.dates.effective), today) >= 0).sort(lawOrder);
  const tags = audienceTags(state.profile);

  return h('section', { class: 'screen' },
    h('div', { class: 'row meta', style: 'padding-top:14px' }, h('span', null, 'Календарь'), h('span', null, `Сегодня ${numeric(today)}`)),
    screenTitle('Что вступает в силу'),
    h('p', { class: 'meta', style: 'margin-top:10px' }, longDay(today)),
    h('div', { style: 'margin-top:14px' }, switchRow({ title: 'Только про меня', hint: 'Иначе — все подписанные законы', on: onlyMine, onChange: (v) => { ctx.local.onlyMine = v; ctx.rerender(); } })),
    rule(true),
    laws.length ? laws.map((l) => {
      const eff = parseDay(l.dates.effective);
      const days = daysBetween(today, eff);
      const tile = calendarTile(eff, today);
      return h('div', null, h('a', { class: 'cal-row', href: `#/law/${encodeURIComponent(l.id)}` },
        h('div', { class: 'cal-tile' }, h('div', { class: 'd' }, tile.day), h('div', { class: 'm' }, tile.month)),
        h('div', null, h('div', { class: 't' }, l.title), h('div', { class: 's' }, `${lawLabel(l, tags)} · `, h('span', { style: 'color:var(--accent)' }, countdownShort(days))))), rule());
    }) : h('div', { class: 'empty' }, onlyMine ? 'Пока нет подтверждённых законов, которые касаются вас.' : 'Пока нет подписанных законов с датой вступления в силу.'),
    theme === 'sage' ? null : h('p', { class: 'meta', style: 'padding:16px 0' }, 'Только подтверждённые законы'),
  );
}
export { meta };
