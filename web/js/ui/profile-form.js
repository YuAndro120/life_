// Ответы «что про тебя важно знать» и выбор региона. Используется в онбординге и во вкладке «Профиль».
import { h } from './dom.js';
import { chip, sheet, rule } from './components.js';
import { AGES, GENDERS, HOUSINGS, OCCUPATIONS, SELLS, WORKS } from '../core/taxonomy.js';
import { REGIONS, regionTitle, searchRegions } from '../core/regions.js';

const toggle = (list, id) => (list.includes(id) ? list.filter((x) => x !== id) : [...list, id]);

export function openRegionSheet(ctx, onDone) {
  let query = '';
  const list = h('div');
  const close = () => { backdrop.dispose?.(); backdrop.remove(); onDone?.(); };
  const pick = (code) => { ctx.store.setProfile({ regionCode: code }); close(); };
  const draw = () => {
    list.replaceChildren(
      query ? null : h('button', { class: 'list-btn', type: 'button', onClick: () => pick(null) }, 'Не указывать'),
      ...searchRegions(query).map((r) => h('button', { class: 'list-btn', type: 'button', onClick: () => pick(r.code) }, r.title,
        ctx.store.state.profile.regionCode === r.code ? h('span', { class: 'muted', 'aria-label': 'выбрано' }, '✓') : null)),
    );
  };
  const input = h('input', { class: 'field', type: 'search', placeholder: 'Название или город', autocomplete: 'off', autocapitalize: 'none', spellcheck: 'false', 'aria-label': 'Поиск региона',
    onInput: (e) => { query = e.target.value; draw(); } });
  const backdrop = sheet(h('div', null,
    h('div', { class: 'row', style: 'padding-top:14px' }, h('h2', { class: 'title', style: 'font-size:22px' }, 'Регион'), h('button', { class: 'tap', type: 'button', onClick: close, style: 'color:var(--accent);font-weight:500' }, 'Готово')),
    h('div', { style: 'margin:10px 0' }, input), list), close);
  draw();
  document.body.append(backdrop);
  input.focus();
}

export function profileForm(ctx) {
  const { store } = ctx;
  const p = store.state.profile;
  const set = (patch) => store.setProfile(patch);
  const theme = ctx.theme;
  const heading = (index, title) => (theme === 'sage' ? h('h3', { style: 'font-size:15px;font-weight:600;color:var(--accent-deep)' }, title) : h('h3', { class: 'meta ink' }, `${index} — ${title}`));
  const group = (index, title, chips) => h('div', { role: 'group', 'aria-label': title, style: 'display:grid;gap:12px' }, heading(index, title), h('div', { class: 'chips' }, chips));

  const showSells = p.work.includes('ip') || p.work.includes('selfemployed');
  return h('div', { style: 'display:grid;gap:28px' },
    group('01', 'О себе', GENDERS.map((g) => chip({ title: g.title, on: p.gender === g.id, onClick: () => set({ gender: p.gender === g.id ? null : g.id }) }))),
    group('02', 'Возраст', AGES.map((a) => chip({ title: a.title, on: p.age === a.id, onClick: () => set({ age: p.age === a.id ? null : a.id }) }))),
    group('03', 'Работа', WORKS.map((w) => chip({ title: w.title, on: p.work.includes(w.id), onClick: () => set({ work: toggle(p.work, w.id) }) }))),
    group('04', 'Сфера работы', OCCUPATIONS.map((o) => chip({ title: o.title, on: p.occupations.includes(o.id), onClick: () => set({ occupations: toggle(p.occupations, o.id) }) }))),
    showSells ? group('05', 'Что продаёшь', SELLS.map((x) => chip({ title: x.title, on: p.sells.includes(x.id), onClick: () => set({ sells: toggle(p.sells, x.id) }) }))) : null,
    group('06', 'Жильё', HOUSINGS.map((x) => chip({ title: x.title, on: p.housing.includes(x.id), onClick: () => set({ housing: toggle(p.housing, x.id) }) }))),
    group('07', 'Транспорт', [
      chip({ title: 'Вожу авто', on: p.drives === true, onClick: () => set({ drives: p.drives === true ? null : true }) }),
      chip({ title: 'Не вожу', on: p.drives === false, onClick: () => set({ drives: p.drives === false ? null : false }) }),
    ]),
    h('div', { style: 'display:grid;gap:8px' }, heading('08', 'Регион'),
      h('button', { class: 'row tap', type: 'button', style: 'width:100%', onClick: () => openRegionSheet(ctx) },
        h('span', { style: 'font-size:15px' }, 'Для региональных новостей и законов'),
        h('span', { class: 'meta accent' }, regionTitle(p.regionCode) ? `${regionTitle(p.regionCode)} →` : 'Выбрать →'))),
  );
}
export { REGIONS, rule };
