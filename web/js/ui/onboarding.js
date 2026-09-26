import { h } from './dom.js';
import { chip, button, linkButton, rule, screenTitle, sectionTitle, segments, switchRow } from './components.js';
import { profileForm, openRegionSheet } from './profile-form.js';
import { themePicker, openBackupSheet } from './profile.js';
import { aboutChips, applyAbout, parseAbout, withoutChips } from '../core/about.js';
import { COUNTRIES, TOPICS, topicTitle } from '../core/taxonomy.js';
import { toggleInterest } from '../core/filter.js';
import { currentEdition } from './today.js';
import { plural } from '../core/dates.js';
import { regionTitle } from '../core/regions.js';

const EXAMPLES = [
  'Живу в Казани, ИП на патенте, езжу на машине, люблю космос',
  'Студентка из Новосибирска, снимаю квартиру, интересуют наука и кино',
  'Работаю по найму, ипотека, не люблю футбол и сплетни',
];
const HIDEABLE = ['politics', 'crime', 'incidents', 'disasters', 'showbiz', 'sport', 'crypto'];

function top(current, back) {
  return h('div', { style: 'margin-top:8px' },
    h('div', { class: 'row' }, h('button', { class: 'tap', type: 'button', onClick: back, style: 'font-size:15px;font-weight:500' }, '← Назад'), h('span', { class: 'meta' }, `0${current} / 05`)),
    h('div', { class: 'progress', 'aria-hidden': 'true' }, [1, 2, 3, 4, 5].map((i) => h('i', { class: i <= current ? 'on' : '' }))));
}
const footer = (...nodes) => h('div', { class: 'footer' }, nodes);
const privacy = (text) => h('div', { style: 'display:flex;gap:10px;align-items:flex-start;margin-top:24px' }, h('span', { class: 'dot', style: 'margin-top:8px' }), h('p', { style: 'font-size:13px;line-height:1.5;color:var(--muted)' }, text));

export function onboardingScreen(ctx) {
  const local = (ctx.local.onb ??= { step: 'welcome', about: '', dismissed: new Set() });
  const go = (step) => { local.step = step; ctx.rerender(); window.scrollTo(0, 0); };
  const skip = () => { ctx.store.completeOnboarding(); ctx.nav('/'); };
  switch (local.step) {
    case 'about': return aboutStep(ctx, local, go, skip);
    case 'profile': return h('section', { class: 'screen' }, top(2, () => go('about')), screenTitle('Что про тебя важно знать', '?'),
      h('p', { class: 'lead' }, 'Покажем только те законы и изменения, которые касаются тебя.'), h('div', { style: 'margin-top:28px' }, profileForm(ctx)),
      privacy('Профиль хранится только в этом браузере. Сервер не знает, кто ты и что читаешь.'), footer(button({ title: 'Дальше', onClick: () => go('interests') }), linkButton('Пропустить', skip)));
    case 'interests': return interestsStep(ctx, go);
    case 'calm': return calmStep(ctx, go);
    case 'theme': return themeStep(ctx, go);
    case 'building': return buildingStep(ctx, local);
    default: return welcomeStep(ctx, go);
  }
}

function welcomeStep(ctx, go) {
  const point = (n, text) => h('div', { class: 'point' }, h('span', { class: 'meta accent', style: 'padding-top:4px' }, n), h('span', { style: 'font-size:16px;line-height:1.5' }, text));
  return h('section', { class: 'screen' },
    h('div', { class: 'row meta', style: 'padding-top:14px' }, h('span', null, 'Выпуск № 1'), h('span', null, 'Без регистрации')),
    h('h1', { class: 'title welcome-mast' }, 'Штиль', h('span', { class: 'mark' }, '.')),
    h('h2', { class: 'title welcome-lines' }, 'Новости без шума.', h('span', { class: 'm' }, 'Законы — только твои.')),
    h('div', { style: 'margin-top:32px' }, point('01', 'Два выпуска в день, каждый можно дочитать до конца'), point('02', 'Изменения в законах, которые касаются именно тебя'), point('03', 'Настройки живут в браузере, сервер не знает, кто ты')),
    footer(button({ title: 'Начать', onClick: () => go('about') }), h('p', { class: 'meta', style: 'text-align:center;margin-top:10px' }, '5 шагов · около минуты'),
      linkButton('У меня есть копия настроек', () => openBackupSheet(ctx, 'restore', () => ctx.nav('/')))));
}

function aboutStep(ctx, local, go, skip) {
  const chipsBox = h('div');
  const cont = h('span');
  const parsed = () => withoutChips(parseAbout(local.about), local.dismissed);
  const textarea = h('textarea', { class: 'field big', rows: 5, placeholder: 'Например: живу в Казани, ИП на патенте, езжу на машине, люблю космос, не люблю футбол', 'aria-label': 'О себе',
    onInput: (e) => { local.about = e.target.value; draw(); } });
  textarea.value = local.about;
  function draw() {
    const p = parsed();
    const chips = aboutChips(p);
    const manual = !p.regionCode && ctx.store.state.profile.regionCode ? regionTitle(ctx.store.state.profile.regionCode) : null;
    chipsBox.replaceChildren(
      !local.about.trim()
        ? h('div', null, h('p', { class: 'meta', style: 'margin-top:20px' }, 'Или начни с примера'), EXAMPLES.map((ex) => h('div', null,
          h('button', { class: 'tap', type: 'button', style: 'width:100%;text-align:left;padding:10px 0;font-size:14px;color:var(--body)', onClick: () => { local.about = ex; local.dismissed = new Set(); textarea.value = ex; draw(); } }, ex), rule())))
        : h('div', null, h('p', { class: 'meta', style: 'margin-top:20px' }, chips.length ? 'Мы поняли так' : 'Пока ничего не поняли'),
          chips.length || manual ? h('div', { class: 'chips', style: 'margin-top:10px' }, manual ? chip({ title: manual, cls: 'region', on: false }) : null, chips.map((c) => chip({ title: c.title, removable: true, cls: c.kind === 'region' ? 'region' : '', label: `${c.title}, убрать`, onClick: () => { local.dismissed.add(c.id); draw(); } })))
            : h('p', { style: 'font-size:14px;color:var(--muted);margin-top:8px' }, 'Напиши город, работу или что тебе интересно. Или выбери регион вручную.'),
          p.regionCode || ctx.store.state.profile.regionCode ? null : h('button', { class: 'row tap', type: 'button', style: 'width:100%;margin-top:6px', onClick: () => openRegionSheet(ctx) }, h('span', { style: 'font-size:15px' }, 'Регион не нашли'), h('span', { class: 'meta accent' }, 'Выбрать →'))));
    cont.textContent = chips.length ? 'Дальше, всё верно' : 'Дальше';
  }
  const save = () => {
    const { profile, prefs } = applyAbout(parsed(), ctx.store.state.profile, ctx.store.state.prefs);
    ctx.store.setProfile(profile);
    ctx.store.setPrefs(prefs);
    go('profile');
  };
  const node = h('section', { class: 'screen' }, top(1, () => go('welcome')), screenTitle('Расскажи о себе', '?'),
    h('p', { class: 'lead' }, 'Пара слов: где живёшь, чем занимаешься, что любишь и что надоело. Лента подстроится сразу.'),
    h('div', { style: 'margin-top:24px' }, textarea), chipsBox,
    privacy('Текст разбирается в этом браузере и никуда не отправляется. Его даже не сохраняем: остаются только плашки.'),
    footer(h('button', { class: 'btn', type: 'button', onClick: save }, cont, h('span', { 'aria-hidden': 'true' }, '→')), linkButton('Пропустить', skip)));
  draw();
  return node;
}

function interestsStep(ctx, go) {
  const { store } = ctx;
  const { prefs } = store.state;
  return h('section', { class: 'screen' }, top(3, () => go('profile')), screenTitle('Что тебе интересно', '?'),
    h('p', { class: 'lead' }, 'Сюжеты по выбранным темам пойдут первыми. Всё можно поменять в фильтрах.'),
    sectionTitle('01', 'Откуда новости', 'paper'),
    h('div', { class: 'chips', style: 'margin-top:14px' }, COUNTRIES.map((c) => chip({ title: c.title, on: prefs.countries.includes(c.id), onClick: () => {
      const on = prefs.countries.includes(c.id);
      if (on && prefs.countries.length === 1) return;
      store.setPrefs({ countries: on ? prefs.countries.filter((x) => x !== c.id) : [...prefs.countries, c.id] });
    } }))),
    h('p', { style: 'font-size:13px;color:var(--muted);margin-top:10px' }, 'Пересказ иностранных изданий всегда по-русски.'),
    sectionTitle('02', 'Темы', 'paper'),
    h('div', { class: 'chips', style: 'margin-top:14px' }, TOPICS.map((t) => chip({ title: t.title, on: prefs.interests.includes(t.id), onClick: () => store.setPrefs((p) => toggleInterest(p, t.id)) }))),
    footer(button({ title: 'Дальше', onClick: () => go('calm') })));
}

function calmStep(ctx, go) {
  const { store } = ctx;
  const { prefs } = store.state;
  const strict = !prefs.infoTypes.includes('rumor') && !prefs.infoTypes.includes('forecast');
  const mode = (id, title, hint) => h('div', null, h('button', { class: 'row', type: 'button', style: 'width:100%;min-height:64px;gap:14px', 'aria-pressed': String(prefs.heavyMode === id), onClick: () => store.setPrefs({ heavyMode: id }) },
    h('span', { style: `width:22px;height:22px;border-radius:50%;flex:none;border:${prefs.heavyMode === id ? '7px solid var(--accent)' : '1.5px solid var(--muted)'}` }),
    h('span', { style: 'flex:1' }, h('div', { style: 'font-size:17px;font-weight:500' }, title), h('div', { style: 'font-size:13px;color:var(--muted)' }, hint))), rule());
  return h('section', { class: 'screen' }, top(4, () => go('interests')), screenTitle('Что тебе не показывать', '?'),
    h('p', { class: 'lead' }, 'Всё это можно поменять потом в фильтрах.'),
    sectionTitle('01', 'Скрыть темы', 'paper'),
    h('div', { class: 'chips', style: 'margin-top:14px' }, HIDEABLE.map((id) => chip({ title: topicTitle(id), on: prefs.stopTopics.includes(id), removable: true, onClick: () => store.setPrefs((p) => (p.stopTopics.includes(id) ? { ...p, stopTopics: p.stopTopics.filter((x) => x !== id) } : { ...p, stopTopics: [...p.stopTopics, id], interests: p.interests.filter((x) => x !== id) })) }))),
    sectionTitle('02', 'Тяжёлые новости', 'paper'),
    h('div', { style: 'margin-top:8px' }, mode('hide', 'Скрывать', 'Не показывать совсем'), mode('fold', 'Сворачивать', 'Одна нейтральная сводка в конце выпуска'), mode('show', 'Показывать', 'Как обычные сюжеты')),
    sectionTitle('03', 'Тон', 'paper'),
    switchRow({ title: 'Без слухов и прогнозов', hint: 'Только факты и официальные решения', on: strict,
      onChange: (v) => store.setPrefs((p) => ({ ...p, infoTypes: v ? p.infoTypes.filter((t) => t !== 'rumor' && t !== 'forecast') : [...new Set([...p.infoTypes, 'rumor', 'forecast'])] })) }),
    footer(button({ title: 'Дальше', onClick: () => go('theme') })));
}

function themeStep(ctx, go) {
  const { store } = ctx;
  return h('section', { class: 'screen' }, top(5, () => go('calm')), screenTitle('Как будет выглядеть выпуск', '?'),
    h('p', { class: 'lead' }, 'Выбери тему, экран сразу покажет её. Сменить можно в любой момент.'), themePicker(ctx),
    h('p', { style: 'font-size:15px;font-weight:500;margin:20px 0 10px' }, 'Когда ждать выпуск'),
    segments({ options: [['both', 'Утро и вечер'], ['am', 'Только утро'], ['pm', 'Только вечер']], value: store.state.settings.schedule, onChange: (v) => store.setSettings({ schedule: v }) }),
    footer(button({ title: 'Собрать первый выпуск', onClick: () => go('building') })));
}

function buildingStep(ctx, local) {
  const { store } = ctx;
  const e = currentEdition(ctx);
  if (!e) {
    return h('section', { class: 'screen' }, h('div', { class: 'empty' }, store.state.loadError ?? 'Собираем выпуск…'),
      store.state.loadError ? h('div', null, button({ title: 'Повторить', trailing: '↻', onClick: () => ctx.refresh() }), linkButton('Продолжить без выпуска', () => { store.completeOnboarding(); ctx.nav('/'); })) : null);
  }
  const sources = new Set(store.state.feed.stories.flatMap((s) => (s.sources ?? []).map((x) => x.title))).size;
  const total = store.state.feed.stats?.posts_total ?? 0;
  const hidden = Math.max(0, total - e.stories.reduce((n, s) => n + s.post_count, 0));
  const texts = [
    `Собрали ${total} ${plural(total, 'пост', 'поста', 'постов')} из ${sources} ${plural(sources, 'источника', 'источников', 'источников')}`,
    hidden > 0 ? `Скрыли ${hidden}: реклама, стоп-темы, слухи` : 'Лишнего не нашлось',
    `Склеили повторы в ${e.stats.stories} ${plural(e.stats.stories, 'сюжет', 'сюжета', 'сюжетов')}`,
    e.laws.length ? `Нашли ${e.laws.length} ${plural(e.laws.length, 'изменение', 'изменения', 'изменений')} в законах для тебя` : 'Изменений в законах для тебя пока нет',
  ];
  const steps = texts.map((t) => h('div', { class: 'build-step' }, h('span', { class: 'c' }, ''), h('span', { style: 'font-size:15px' }, t)));
  const counter = h('div', { class: 'build-num', 'aria-live': 'polite' }, String(total));
  const label = h('p', { class: 'muted', style: 'margin-top:6px' }, `постов из ${sources} источников`);
  const status = h('span', { class: 'meta accent' }, 'Читаем источники');
  const open = h('div', { style: 'margin-top:24px', hidden: true }, button({ title: 'Открыть выпуск', onClick: () => { store.completeOnboarding(); store.bumpEdition(ctx.now()); ctx.nav('/'); } }));
  const node = h('section', { class: 'screen' }, h('div', { class: 'row meta', style: 'padding-top:14px' }, h('span', null, 'Выпуск № 1'), h('span', null, 'Собираем')),
    counter, label, h('div', { style: 'display:flex;gap:8px;align-items:center;margin-top:22px' }, h('span', { class: 'dot' }), status), h('div', { class: 'build-steps' }, steps), open);
  const show = (i) => { const s = steps[i]; if (!s) return; s.classList.add('ok'); s.firstChild.textContent = '✓'; };
  const finish = () => { steps.forEach((_, i) => show(i)); counter.textContent = String(e.stats.stories); label.textContent = 'сюжетов в первом выпуске'; status.textContent = 'Готово'; open.hidden = false; local.built = true; };
  if (local.built || matchMedia('(prefers-reduced-motion: reduce)').matches) { finish(); return node; }
  const stages = [[1200, 'Убираем рекламу и стоп-темы', 0], [2600, 'Склеиваем повторы', 1], [3900, 'Ищем законы про тебя', 2]];
  const start = performance.now();
  const tick = () => {
    if (!node.isConnected) return;
    const t = performance.now() - start;
    for (const [at, text, idx] of stages) if (t > at) { show(idx); status.textContent = text; }
    counter.textContent = String(Math.round(total - (total - e.stats.stories) * Math.min(1, Math.max(0, (t - 1200) / 2400))));
    if (t > 1600) label.textContent = t > 3800 ? 'сюжетов в первом выпуске' : 'осталось после фильтров';
    if (t > 4500) return finish();
    requestAnimationFrame(tick);
  };
  requestAnimationFrame(tick);
  return node;
}
