import { h } from './dom.js';
import { chip, button, linkButton, rule } from './components.js';
import { openRegionSheet } from './profile-form.js';
import { themePicker, openBackupSheet } from './profile.js';
import { aboutChips, applyAbout, parseAbout, withoutChips } from '../core/about.js';
import { AGES, COUNTRIES, GENDERS, HOUSINGS, OCCUPATIONS, SELLS, TOPICS, WORKS, topicTitle } from '../core/taxonomy.js';
import { toggleInterest, toggleStopTopic } from '../core/filter.js';
import { currentEdition } from './today.js';
import { plural } from '../core/dates.js';
import { regionTitle } from '../core/regions.js';

const EXAMPLES = [
  'Живу в Казани, ИП на патенте, езжу на машине, люблю космос',
  'Студентка из Новосибирска, снимаю квартиру, интересуют наука и кино',
  'Работаю по найму, ипотека, не люблю футбол и сплетни',
];

function top(current, back) {
  return h('div', { style: 'margin-top:8px' },
    h('div', { class: 'row' }, h('button', { class: 'tap', type: 'button', onClick: back, style: 'font-size:15px;font-weight:500' }, '← Назад'), h('span', { class: 'meta' }, `0${current} / 04`)),
    h('div', { class: 'progress', 'aria-hidden': 'true' }, [1, 2, 3, 4].map((i) => h('i', { class: i <= current ? 'on' : '' }))));
}
const footer = (...nodes) => h('div', { class: 'footer' }, nodes);
const privacy = (text) => h('div', { style: 'display:flex;gap:10px;align-items:flex-start;margin-top:24px' }, h('span', { class: 'dot', style: 'margin-top:8px' }), h('p', { style: 'font-size:13px;line-height:1.5;color:var(--muted)' }, text));

export function onboardingScreen(ctx) {
  const local = (ctx.local.onb ??= { step: 'welcome', about: '', dismissed: new Set() });
  const go = (step) => { local.step = step; ctx.rerender(); document.getElementById('scroll')?.scrollTo(0, 0); };
  const skip = () => { ctx.store.completeOnboarding(); ctx.nav('/'); };
  switch (local.step) {
    case 'about': return aboutStep(ctx, local, go, skip);
    case 'who': return whoStep(ctx, local, go, skip);
    case 'read': return readStep(ctx, go, skip);
    case 'look': return lookStep(ctx, go);
    case 'building': return buildingStep(ctx, local);
    default: return welcomeStep(ctx, go);
  }
}

function welcomeStep(ctx, go) {
  const point = (n, text) => h('div', { class: 'point' }, h('span', { class: 'meta accent', style: 'padding-top:4px' }, n), h('span', { style: 'font-size:16px;line-height:1.5' }, text));
  return h('section', { class: 'screen' },
    h('div', { class: 'row meta', style: 'padding-top:14px' }, h('span', null, 'Выпуск № 1'), h('span', null, 'Без регистрации')),
    h('h1', { class: 'title welcome-mast' }, 'Штиль', h('span', { class: 'mark' }, '.')),
    h('h2', { class: 'title welcome-lines' }, 'Новости без шума.', h('span', { class: 'm' }, 'Законы тебе в помощь.')),
    h('div', { style: 'margin-top:32px' }, point('01', 'Два выпуска в день, каждый можно дочитать до конца'), point('02', 'Изменения в законах, которые касаются именно тебя'), point('03', 'Настройки живут в браузере, сервер не знает, кто ты')),
    footer(button({ title: 'Начать', onClick: () => go('about') }), h('p', { class: 'meta', style: 'text-align:center;margin-top:10px' }, '4 шага · около минуты'),
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
    go('who');
  };
  const node = h('section', { class: 'screen' }, top(1, () => go('welcome')), h('h1', { class: 'title h-onb' }, 'Расскажи о себе', h('span', { class: 'mark' }, '?')),
    h('p', { class: 'lead short' }, 'Где живёшь, чем занимаешься, что любишь. Лента подстроится сразу.'),
    h('div', { style: 'margin-top:24px' }, textarea), chipsBox,
    privacy('Текст разбирается в этом браузере и никуда не отправляется. Его даже не сохраняем: остаются только плашки.'),
    footer(h('button', { class: 'btn', type: 'button', onClick: save }, cont, h('span', { 'aria-hidden': 'true' }, '→')), linkButton('Пропустить', skip)));
  draw();
  return node;
}

const toggleIn = (list, id) => (list.includes(id) ? list.filter((x) => x !== id) : [...list, id]);

const ask = (title, sub, ...body) => h('div', { class: 'ask' }, h('div', { class: 'q' }, title, sub ? h('span', { class: 'sub' }, sub) : null), body);

/** «Немного о тебе»: сначала работа, возраст и регион; остальное появляется по мере ответов или под «Ещё о себе». */
function whoStep(ctx, local, go, skip) {
  const { store } = ctx;
  const p = store.state.profile;
  const set = (patch) => store.setProfile(patch);
  const showSphere = p.work.some((w) => ['employee', 'ip', 'selfemployed'].includes(w));
  const showSells = p.work.includes('ip') || p.work.includes('selfemployed');
  const chips = (list, current, key) => h('div', { class: 'chips' }, list.map((x) => chip({ title: x.title, on: current.includes(x.id), onClick: () => set({ [key]: toggleIn(current, x.id) }) })));
  const single = (list, current, key) => h('div', { class: 'chips' }, list.map((x) => chip({ title: x.title, on: current === x.id, onClick: () => set({ [key]: current === x.id ? null : x.id }) })));
  const region = regionTitle(p.regionCode);
  return h('section', { class: 'screen' }, top(2, () => go('about')),
    h('h1', { class: 'title h-onb' }, 'Немного о тебе'),
    h('p', { class: 'lead short' }, 'Всё необязательно. Чем точнее, тем меньше лишних законов.'),
    ask('Чем занимаешься', null, chips(WORKS, p.work, 'work')),
    showSphere ? ask('Сфера', null, chips(OCCUPATIONS, p.occupations, 'occupations')) : null,
    showSells ? ask('Что продаёшь', null, chips(SELLS, p.sells, 'sells')) : null,
    ask('Возраст', null, single(AGES, p.age, 'age')),
    h('div', { class: 'ask' }, h('button', { class: 'region-row', type: 'button', onClick: () => openRegionSheet(ctx) },
      h('span', null, h('div', { class: 'l' }, 'Регион'), h('div', { class: 'v' }, region ?? 'Не выбран')), h('span', { class: 'meta accent' }, region ? 'Изменить' : 'Выбрать →'))),
    h('button', { class: 'more-btn', type: 'button', 'aria-expanded': String(Boolean(local.more)), onClick: () => { local.more = !local.more; ctx.rerender(); } },
      h('span', null, 'Ещё о себе'), h('span', { class: 'm' }, local.more ? 'Свернуть' : 'пол, жильё, авто ▾')),
    local.more ? h('div', null,
      ask('Пол', null, single(GENDERS, p.gender, 'gender')),
      ask('Жильё', null, chips(HOUSINGS, p.housing, 'housing')),
      ask('Транспорт', null, h('div', { class: 'chips' },
        chip({ title: 'Вожу авто', on: p.drives === true, onClick: () => set({ drives: p.drives === true ? null : true }) }),
        chip({ title: 'Не вожу', on: p.drives === false, onClick: () => set({ drives: p.drives === false ? null : false }) })))) : null,
    footer(button({ title: 'Дальше', onClick: () => go('read') }), linkButton('Пропустить', skip)));
}

/** Тап по теме: интересно → скрыть → как обычно (и по кругу). */
export function cycleTopic(prefs, id) {
  if (prefs.interests.includes(id)) return toggleStopTopic(prefs, id);
  if (prefs.stopTopics.includes(id)) return { ...prefs, stopTopics: prefs.stopTopics.filter((x) => x !== id) };
  return toggleInterest(prefs, id);
}

/** «Что читать»: одна сетка тем вместо двух экранов и компактная строка изданий. */
function readStep(ctx, go, skip) {
  const { store } = ctx;
  const { prefs } = store.state;
  const state = (id) => (prefs.interests.includes(id) ? 'interest' : prefs.stopTopics.includes(id) ? 'hidden' : '');
  const title = (t) => (state(t.id) === 'hidden' ? `✕ ${t.title}` : t.title);
  return h('section', { class: 'screen' }, top(3, () => go('who')),
    h('h1', { class: 'title h-onb' }, 'Что читать', h('span', { class: 'mark' }, '?')),
    h('p', { class: 'lead short' }, 'Тап по теме: интересно, ещё тап: скрыть.'),
    ask('Издания', null, h('div', { class: 'chips' }, COUNTRIES.map((c) => chip({ title: c.title, on: prefs.countries.includes(c.id), onClick: () => {
      const on = prefs.countries.includes(c.id);
      if (on && prefs.countries.length === 1) return;
      store.setPrefs({ countries: on ? prefs.countries.filter((x) => x !== c.id) : [...prefs.countries, c.id] });
    } })))),
    ask('Темы', null, h('div', { class: 'chips' }, TOPICS.map((t) => chip({ title: title(t), state: state(t.id), label: `${t.title}: ${{ interest: 'интересно', hidden: 'скрыто', '': 'как обычно' }[state(t.id)]}`, onClick: () => store.setPrefs((p) => cycleTopic(p, t.id)) })))),
    h('div', { class: 'legend', 'aria-hidden': 'true' },
      h('span', null, h('i', { style: 'background:var(--accent)' }), 'интересно'), h('span', null, h('i', { style: 'border:1.5px dashed var(--muted)' }), 'скрыто'), h('span', null, h('i', { style: 'border:1.5px solid var(--chip-border,var(--line))' }), 'как обычно')),
    footer(button({ title: 'Дальше', onClick: () => go('look') }), linkButton('Пропустить', skip)));
}

function lookStep(ctx, go) {
  return h('section', { class: 'screen' }, top(4, () => go('read')),
    h('h1', { class: 'title h-onb' }, 'Как оформить', h('span', { class: 'mark' }, '?')),
    h('p', { class: 'lead short' }, 'Выбери тему, экран сразу покажет её. Сменить можно в любой момент.'),
    themePicker(ctx),
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
