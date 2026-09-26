import { h } from './dom.js';
import { chip, meta, rule, screenTitle, sectionTitle, segments, sheet, switchRow } from './components.js';
import { toggleInterest, toggleStopTopic } from '../core/filter.js';
import { COUNTRIES, INFO_TYPES, TOPICS, topicTitle } from '../core/taxonomy.js';
import { MAX_WORDS, normalizeWord } from '../core/markers.js';
import { clock } from '../core/dates.js';

const INFO_HINTS = {
  fact: ['Факты', 'Что произошло, без оценок'], official: ['Официальные решения', 'Законы, постановления, ведомства'],
  opinion: ['Мнения экспертов', 'Оценки и комментарии'], forecast: ['Прогнозы', '«Что будет, если…»'], rumor: ['Неподтверждённое', '«По данным источников…»'],
};
const HEAVY = [['hide', 'Скрывать'], ['fold', 'Сворачивать'], ['show', 'Показывать']];
const LIMITS = [10, 15, 20, 30];
const toMinutes = (value) => { const [hh, mm] = value.split(':').map(Number); return (hh || 0) * 60 + (mm || 0); };

function topicPicker(ctx) {
  const { store } = ctx;
  const body = h('div', { class: 'chips', style: 'margin-top:12px' });
  const close = () => { backdrop.dispose?.(); backdrop.remove(); };
  const draw = () => body.replaceChildren(...TOPICS.filter((t) => !store.state.prefs.stopTopics.includes(t.id)).map((t) => chip({ title: t.title, onClick: () => { store.setPrefs((p) => toggleStopTopic(p, t.id)); draw(); } })));
  const backdrop = sheet(h('div', null, h('div', { class: 'row', style: 'padding-top:14px' }, h('h2', { class: 'title', style: 'font-size:22px' }, 'Скрыть тему'),
    h('button', { class: 'tap', type: 'button', onClick: close, style: 'color:var(--accent);font-weight:500' }, 'Готово')), body), close);
  draw();
  document.body.append(backdrop);
}

export function filtersScreen(ctx) {
  const { store } = ctx;
  const { prefs, settings, profile } = store.state;
  const theme = ctx.theme;
  const set = (patch) => store.setPrefs(patch);
  const section = (index, title, ...body) => h('div', null, sectionTitle(index, title, theme), h('div', { class: 'section-body' }, body));
  const note = (text) => h('p', { style: 'font-size:13px;color:var(--muted);margin-bottom:12px' }, text);

  // Слова пользователя: поле и список
  let word = '';
  const wordsBox = h('div');
  const add = () => {
    const w = normalizeWord(word);
    word = '';
    const p = store.state.prefs;
    if (!w || p.blockedWords.includes(w) || p.blockedWords.length >= MAX_WORDS) return draw();
    set({ blockedWords: [...p.blockedWords, w] });
  };
  const draw = () => {
    const input = h('input', { class: 'field', type: 'text', placeholder: 'Слово или фраза', value: word, autocomplete: 'off', autocapitalize: 'none', spellcheck: 'false', enterkeyhint: 'done', 'aria-label': 'Слово или фраза для скрытия',
      onInput: (e) => { word = e.target.value; addBtn.disabled = !normalizeWord(word); }, onKeydown: (e) => { if (e.key === 'Enter') add(); } });
    const addBtn = h('button', { class: 'tap', type: 'button', disabled: !normalizeWord(word), style: 'color:var(--accent);font-weight:500', onClick: add }, 'Добавить');
    wordsBox.replaceChildren(h('div', { class: 'row' }, h('div', { style: 'flex:1' }, input), addBtn),
      store.state.prefs.blockedWords.length ? h('div', { class: 'chips', style: 'margin-top:12px' }, store.state.prefs.blockedWords.map((w) => chip({ title: w, removable: true, on: false, label: `Убрать слово ${w}`, onClick: () => set({ blockedWords: store.state.prefs.blockedWords.filter((x) => x !== w) }) }))) : null);
  };
  draw();

  const hasHidden = prefs.mutedSources.length || prefs.hiddenStories.length;
  return h('section', { class: 'screen' },
    theme === 'sage' ? h('p', { class: 'meta', style: 'padding-top:14px' }, 'Хранятся в этом браузере') : h('div', { class: 'row meta', style: 'padding-top:14px' }, h('span', null, 'Настройки выпуска'), h('span', null, 'Хранятся в браузере')),
    screenTitle('Фильтры'),
    theme === 'sage' ? h('p', { class: 'meta', style: 'margin-top:6px' }, 'что попадает в выпуск') : null,

    section('01', 'Тон', switchRow({ title: 'Спокойный режим', hint: 'Нейтральные заголовки, без «срочно»', on: prefs.calmMode, onChange: (v) => set({ calmMode: v }) })),
    section('02', 'Откуда новости', note('Издания этих стран. Пересказ всегда по-русски.'),
      h('div', { class: 'chips' }, COUNTRIES.map((c) => chip({ title: c.title, on: prefs.countries.includes(c.id), onClick: () => {
        const on = prefs.countries.includes(c.id);
        if (on && prefs.countries.length === 1) return;
        set({ countries: on ? prefs.countries.filter((x) => x !== c.id) : [...prefs.countries, c.id] });
      } }))),
      switchRow({ title: 'Только мой регион и федеральные', hint: profile.regionCode ? 'Новости других регионов скрыты' : 'Регион не выбран: региональные новости скрыты', on: prefs.hideOtherRegions, onChange: (v) => set({ hideOtherRegions: v }) })),
    section('03', 'Мои интересы', note('Сюжеты по этим темам идут первыми и отмечены точкой.'),
      h('div', { class: 'chips' }, TOPICS.map((t) => chip({ title: t.title, on: prefs.interests.includes(t.id), onClick: () => set((p) => toggleInterest(p, t.id)) }))),
      prefs.interests.length ? switchRow({ title: 'Только мои интересы', hint: 'Остальные темы скрыты', on: prefs.onlyInterests, onChange: (v) => set({ onlyInterests: v }) }) : null),
    section('04', 'Размер выпуска', note('Сколько сюжетов показывать: выпуск можно дочитать до конца.'),
      segments({ options: LIMITS.map((n) => [n, String(n)]), value: prefs.storyLimit, onChange: (n) => set({ storyLimit: n }) })),
    section('05', 'Тип информации', INFO_TYPES.map((t) => h('div', null, switchRow({ title: INFO_HINTS[t.id][0], hint: INFO_HINTS[t.id][1], on: prefs.infoTypes.includes(t.id),
      onChange: (v) => set({ infoTypes: v ? [...prefs.infoTypes, t.id] : prefs.infoTypes.filter((x) => x !== t.id) }) }), rule()))),
    section('06', 'Тяжёлые темы', note('Насилие, катастрофы, тяжёлые преступления.'), segments({ options: HEAVY, value: prefs.heavyMode, onChange: (m) => set({ heavyMode: m }) }),
      prefs.heavyMode === 'show' ? h('div', { style: 'margin-top:12px' }, note('Сколько показывать (остальные сворачиваются)'), segments({ options: [1, 3, 5].map((n) => [n, String(n)]), value: prefs.maxHeavy, onChange: (n) => set({ maxHeavy: n }) })) : null),
    section('07', 'Стоп-темы', switchRow({ title: 'Скрывать всё про СВО', hint: 'Кроме официального заявления, что СВО закончилась', on: prefs.hideWar, onChange: (v) => set({ hideWar: v }) }),
      h('div', { class: 'chips', style: 'margin-top:12px' },
        prefs.stopTopics.map((id) => chip({ title: topicTitle(id), removable: true, label: `Убрать стоп-тему ${topicTitle(id)}`, onClick: () => set((p) => toggleStopTopic(p, id)) })),
        chip({ title: '+ Добавить', cls: 'dashed', onClick: () => topicPicker(ctx) }))),
    section('08', 'Слова', note('Сюжеты с этими словами в заголовке или пересказе скрываются. Форму слова учитываем: «футбол» найдёт и «футболист».'), wordsBox),
    section('09', 'Реклама', switchRow({ title: 'Скрывать рекламные посты', hint: 'По маркировке «Реклама» и erid', on: prefs.hideAds, onChange: (v) => set({ hideAds: v }) })),
    section('10', 'Расписание', segments({ options: [['both', 'Утро и вечер'], ['am', 'Только утро'], ['pm', 'Только вечер']], value: settings.schedule, onChange: (v) => store.setSettings({ schedule: v }) }),
      settings.schedule !== 'pm' ? timeRow('Утренний выпуск', settings.morningMinutes, (m) => store.setSettings({ morningMinutes: m })) : null,
      settings.schedule !== 'am' ? timeRow('Вечерний выпуск', settings.eveningMinutes, (m) => store.setSettings({ eveningMinutes: m })) : null,
      note('В браузере нет надёжных уведомлений: время влияет на подпись «Следующий выпуск» и на номер выпуска.')),
    hasHidden ? section('11', 'Скрытое',
      prefs.mutedSources.length ? h('div', null, h('p', { style: 'font-size:13px;color:var(--muted);margin:4px 0' }, 'Скрытые источники'),
        h('div', { class: 'chips' }, prefs.mutedSources.map((t) => chip({ title: t, removable: true, label: `Вернуть источник ${t}`, onClick: () => set({ mutedSources: prefs.mutedSources.filter((x) => x !== t) }) })))) : null,
      prefs.hiddenStories.length ? h('button', { class: 'row tap', type: 'button', style: 'width:100%;min-height:60px', onClick: () => set({ hiddenStories: [] }) },
        h('span', { style: 'font-size:17px;font-weight:500' }, `Скрытых сюжетов: ${prefs.hiddenStories.length}`), h('span', { class: 'meta accent' }, 'Вернуть все →')) : null) : null,
  );
}

function timeRow(title, minutes, onChange) {
  return h('label', { class: 'row', style: 'min-height:56px' }, h('span', { style: 'font-size:16px' }, title),
    h('input', { class: 'field', type: 'time', value: clock(minutes), style: 'width:auto', onChange: (e) => e.target.value && onChange(toMinutes(e.target.value)) }));
}
export { meta };
