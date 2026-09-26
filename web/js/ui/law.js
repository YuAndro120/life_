import { h, externalLink } from './dom.js';
import { meta, rule, topbar } from './components.js';
import { audienceTags, lawLabel, lawReason } from '../core/audience.js';
import { countdownWords, dayMonth, dayMonthYear, daysBetween, daysWords, dayOf, numeric, numericShort, parseDay } from '../core/dates.js';
import { lawIcs } from '../ics.js';
import { currentEdition } from './today.js';
import { regionTitle } from '../core/regions.js';

const STEPS = [['introduced', 'Внесён'], ['passed', 'Принят'], ['signed', 'Подписан'], ['effective', 'В силе']];
const ORDER = { introduced: 0, passed: 1, signed: 2, in_force: 3 };

function downloadIcs(law, now) {
  const ics = lawIcs(law, now);
  if (!ics) return;
  const url = URL.createObjectURL(new Blob([ics], { type: 'text/calendar;charset=utf-8' }));
  const a = h('a', { href: url, download: `shtil-${law.id}.ics` });
  document.body.append(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 5000);
}

/** Экран закона: сколько до вступления, что меняется, кого касается, что сделать, откуда сведения. */
export function lawScreen(ctx, id) {
  const { state } = ctx.store;
  const law = state.laws.find((l) => l.id === id);
  if (!law) return h('section', { class: 'screen' }, topbar({ backLabel: '← Выпуск', onBack: () => ctx.nav('/') }), h('div', { class: 'empty' }, 'Закон не найден'));
  const theme = ctx.theme;
  const now = ctx.now();
  const today = dayOf(now);
  const e = currentEdition(ctx);
  const tags = audienceTags(state.profile);
  const eff = parseDay(law.dates?.effective);
  const days = eff ? daysBetween(today, eff) : null;
  const audience = lawLabel(law, tags);
  const done = ctx.local.done ??= {};
  const doneSet = (done[law.id] ??= new Set());
  const verified = law.verified_at ? ` Сверено с официальным текстом ${numeric(parseDay(law.verified_at))}.` : '';
  const isRegional = Boolean(law.region_code);
  const url = law.official_url || law.bill_url;

  const level = ORDER[law.status] ?? 0;
  const steps = STEPS.map(([key, title], i) => {
    const d = parseDay(law.dates?.[key]);
    const cls = i < level ? 'done' : i === level ? 'now' : '';
    return h('div', { class: `step ${cls}` }, h('div', { class: 's' }, title), h('div', { class: 'd' }, d ? (theme === 'sage' ? dayMonth(d, today) : numericShort(d, today)) : '—'));
  });

  const fact = (label, text, accent = false) => h('div', null, rule(), h('div', { class: 'fact' }, h('div', { class: `k${accent ? ' accent' : ''}` }, label), h('div', { class: 'v' }, text)));

  return h('section', { class: 'screen' },
    topbar({
      backLabel: theme === 'sage' ? '‹ Выпуск' : `← Выпуск ${e?.number ?? 1}`, onBack: () => ctx.nav('/'),
      right: url && navigator.share ? h('button', { class: 'share', type: 'button', onClick: () => navigator.share({ title: law.title, text: law.title, url }).catch(() => {}) }, 'Поделиться') : h('span'),
    }),
    h('div', { style: 'margin-top:20px' }, meta(theme === 'sage' ? audience : `Закон / ${audience}${isRegional && regionTitle(law.region_code) ? ` · ${regionTitle(law.region_code)}` : ''}`)),
    h('h1', { class: 'title detail-title' }, law.title),
    eff ? h('div', { class: 'big-date' },
      h('span', { class: 'n' }, days > 0 ? String(days) : '0'),
      h('span', null, h('div', { style: 'font-size:16px;font-weight:500' }, days > 0 ? `${daysWords(days).split(' ')[1]} до вступления в силу` : 'Вступает в силу'),
        h('div', { class: 'meta' }, `${dayMonthYear(eff)} · ${countdownWords(days)}`))) : h('p', { class: 'meta', style: 'margin-top:16px' }, 'Дата вступления в силу не указана'),
    h('div', { class: 'steps' }, steps),
    h('div', { style: 'margin-top:20px' }, fact('Что изменилось', law.what_changed), fact('Кого касается', law.who_affected), fact('Почему тебе', lawReason(law, state.profile), true)),
    law.actions?.length ? h('div', { style: 'margin-top:8px' }, meta('Что сделать'), law.actions.map((a, i) => h('div', null, rule(), h('button', { class: 'todo', type: 'button', 'aria-pressed': String(doneSet.has(i)), onClick: () => { doneSet.has(i) ? doneSet.delete(i) : doneSet.add(i); ctx.rerender(); } },
      h('span', { class: 'check', 'aria-hidden': 'true' }, doneSet.has(i) ? '✓' : ''), h('span', { class: 'tx', style: 'font-size:16px' }, a)))), rule()) : null,
    eff && days >= 0 ? h('div', { style: 'margin-top:20px' }, h('button', { class: 'btn', type: 'button', onClick: () => downloadIcs(law, now) }, h('span', null, 'Напомнить в календаре'), h('span', { 'aria-hidden': 'true' }, '＋')),
      h('p', { class: 'meta', style: 'margin-top:8px' }, 'Файл календаря: напоминание за неделю и за день до вступления в силу.')) : null,
    h('div', { style: 'margin-top:28px' }, meta('Источники', 'ink'),
      law.official_url ? h('div', { style: 'margin-top:8px' }, rule(), externalLink(law.official_url, { class: 'link-row' }, h('span', null, h('div', { class: 't' }, 'Официальный текст'), h('div', { class: 's' }, law.act_number ?? 'publication.pravo.gov.ru')), h('span', { 'aria-hidden': 'true' }, '↗'))) : null,
      law.bill_url ? h('div', null, rule(), externalLink(law.bill_url, { class: 'link-row' }, h('span', null, h('div', { class: 't' }, 'Карточка законопроекта'), h('div', { class: 's' }, 'sozd.duma.gov.ru')), h('span', { 'aria-hidden': 'true' }, '↗'))) : null, rule()),
    h('p', { class: 'disclaimer' }, `Пересказ простым языком, не юридическая консультация.${verified}`),
  );
}
