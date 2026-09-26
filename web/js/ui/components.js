import { h } from './dom.js';

export const meta = (text, cls = '') => h('span', { class: `meta ${cls}` }, text);
export const rule = (strong = false) => h('hr', { class: `rule${strong ? ' strong' : ''}` });

export function button({ title, trailing = '→', onClick, disabled = false }) {
  return h('button', { class: 'btn', type: 'button', onClick, disabled }, h('span', null, title), trailing ? h('span', { 'aria-hidden': 'true' }, trailing) : null);
}

export const linkButton = (title, onClick) => h('button', { class: 'link-btn', type: 'button', onClick }, title);

export function chip({ title, on = false, removable = false, onClick, cls = '', label }) {
  return h('button', { class: `chip${removable ? ' removable' : ''} ${cls}`.trim(), type: 'button', 'aria-pressed': String(on), 'aria-label': label, onClick },
    title, removable ? h('span', { class: 'x', 'aria-hidden': 'true' }, '×') : null);
}

export function switchRow({ title, hint, on, onChange }) {
  const sw = h('span', { class: 'switch', role: 'switch', 'aria-checked': String(on), 'aria-label': title });
  const row = h('button', { class: 'switch-row', type: 'button', onClick: () => onChange(!on) },
    h('span', { class: 'text' }, h('div', { class: 't' }, title), hint ? h('div', { class: 'hint' }, hint) : null), sw);
  return row;
}

export function segments({ options, value, onChange }) {
  return h('div', { class: 'segments', role: 'group' }, options.map(([id, title]) => h('button', { type: 'button', 'aria-pressed': String(id === value), onClick: () => onChange(id) }, title)));
}

/** Заголовок раздела: «01 — Тон» (Бумага, Сумерки) или обычный подзаголовок (Шалфей). */
export function sectionTitle(index, title, theme) {
  return theme === 'sage' ? h('h2', { class: 'section-title' }, title) : h('h2', { class: 'section-title meta ink' }, `${index} — ${title}`);
}

export const screenTitle = (text, mark = '') => h('h1', { class: 'title h-screen' }, text, mark ? h('span', { class: 'mark' }, mark) : null);

export function topbar({ backLabel, onBack, right }) {
  return h('div', null, h('div', { class: 'topbar' }, h('button', { class: 'back', type: 'button', onClick: onBack }, backLabel), right ?? h('span')), rule(true));
}

/** Окно поверх экрана; закрывается по фону и по Escape. */
export function sheet(content, onClose) {
  const backdrop = h('div', { class: 'sheet-backdrop', onClick: (e) => { if (e.target === backdrop) onClose(); } }, h('div', { class: 'sheet', role: 'dialog', 'aria-modal': 'true' }, content));
  const onKey = (e) => { if (e.key === 'Escape') { onClose(); document.removeEventListener('keydown', onKey); } };
  document.addEventListener('keydown', onKey);
  backdrop.dispose = () => document.removeEventListener('keydown', onKey);
  return backdrop;
}
