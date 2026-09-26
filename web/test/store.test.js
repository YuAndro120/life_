import test from 'node:test';
import assert from 'node:assert/strict';
import { createStore } from '../js/store.js';
import { encryptBackup, decryptBackup, BackupError } from '../js/backup.js';
import { resolveTheme } from '../js/theme.js';
import { ACTIONS } from '../js/core/filter.js';

const memory = () => {
  const m = new Map();
  return { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => m.set(k, String(v)), removeItem: (k) => m.delete(k), m };
};

test('настройки сохраняются между запусками, регион в prefs не пишется', () => {
  const storage = memory();
  const a = createStore(storage);
  a.setProfile({ regionCode: '16', work: ['ip'] });
  a.setPrefs({ interests: ['space'], blockedWords: ['сплетн'] });
  const b = createStore(storage);
  assert.equal(b.state.profile.regionCode, '16');
  assert.deepEqual(b.state.prefs.interests, ['space']);
  assert.equal(b.effectivePrefs().homeRegion, '16');
  assert.equal(JSON.parse(storage.getItem('shtil.prefs')).homeRegion, null);
});

test('повреждённые или старые данные заменяются значениями по умолчанию', () => {
  const storage = memory();
  storage.setItem('shtil.prefs', '{сломано');
  storage.setItem('shtil.profile', JSON.stringify({ work: ['ip'] }));
  const s = createStore(storage);
  assert.equal(s.state.prefs.hideWar, true);
  assert.deepEqual(s.state.profile.occupations, []);
  assert.deepEqual(s.state.profile.work, ['ip']);
});

test('работает без хранилища (приватный режим)', () => {
  const broken = { getItem: () => { throw new Error('нет'); }, setItem: () => { throw new Error('нет'); }, removeItem: () => {} };
  const s = createStore(broken);
  s.setPrefs({ storyLimit: 5 });
  assert.equal(s.state.prefs.storyLimit, 5);
});

test('«Не интересно»: применить, отменить, повторное действие ничего не меняет', () => {
  const s = createStore(memory());
  assert.equal(s.applyFeedback(ACTIONS.hideStory('a')), true);
  assert.deepEqual(s.state.prefs.hiddenStories, ['a']);
  assert.equal(s.applyFeedback(ACTIONS.hideStory('a')), false);
  assert.deepEqual(s.state.undo, ACTIONS.hideStory('a'));
  s.undoFeedback();
  assert.deepEqual(s.state.prefs.hiddenStories, []);
  assert.equal(s.state.undo, null);
});

test('скрытых сюжетов хранится не больше лимита, последние остаются', () => {
  const s = createStore(memory());
  for (let i = 0; i < 310; i += 1) s.applyFeedback(ACTIONS.hideStory(`s${i}`));
  assert.equal(s.state.prefs.hiddenStories.length, 300);
  assert.equal(s.state.prefs.hiddenStories.at(-1), 's309');
});

test('кэш ленты переживает перезапуск; без сети и без кэша — ошибка', async () => {
  const storage = memory();
  const a = createStore(storage);
  a.setLoadFailed('нет связи');
  assert.equal(a.state.loadError, 'нет связи');
  a.setContent({ feed: { stories: [] }, laws: [{ id: 'l' }] });
  await new Promise((r) => setTimeout(r, 20)); // запись кэша откладывается на простой
  const b = createStore(storage);
  assert.deepEqual(b.state.laws, [{ id: 'l' }]);
  b.setLoadFailed('нет связи');
  assert.equal(b.state.offline, true);
  assert.equal(b.state.loadError, null);
});

test('номер выпуска растёт раз в утро и вечер', () => {
  const s = createStore(memory());
  s.completeOnboarding();
  const n = () => s.state.counter.number;
  s.bumpEdition(new Date(2026, 8, 26, 9, 0));
  const first = n();
  s.bumpEdition(new Date(2026, 8, 26, 10, 0));
  assert.equal(n(), first);
  s.bumpEdition(new Date(2026, 8, 26, 19, 0));
  assert.equal(n(), first + 1);
});

test('зашифрованная копия: круг, неверный пароль, испорченный код', async () => {
  const s = createStore(memory());
  s.setProfile({ regionCode: '16', work: ['ip'], occupations: ['trade'] });
  s.setPrefs({ interests: ['space'], hideWar: false });
  const code = await encryptBackup(s.snapshot(), 'секретный пароль', 1000);
  assert.ok(code.startsWith('SHTIL1.'));
  const back = await decryptBackup(code, 'секретный пароль');
  assert.equal(back.profile.regionCode, '16');
  assert.deepEqual(back.prefs.interests, ['space']);
  await assert.rejects(decryptBackup(code, 'другой'), BackupError);
  await assert.rejects(decryptBackup('чужое', 'x'), /SHTIL1/);
  await assert.rejects(decryptBackup(`${code.slice(0, -6)}AAAAAA`, 'секретный пароль'), BackupError);
  const fresh = createStore(memory());
  fresh.restore(back);
  assert.equal(fresh.state.profile.regionCode, '16');
  assert.equal(fresh.state.prefs.hideWar, false);
});

test('тема: вечером и ночью «Сумерки», если включено автоматически', () => {
  const settings = { theme: 'sage', autoDusk: true };
  assert.equal(resolveTheme(settings, new Date(2026, 8, 26, 12)), 'sage');
  assert.equal(resolveTheme(settings, new Date(2026, 8, 26, 20)), 'dusk');
  assert.equal(resolveTheme(settings, new Date(2026, 8, 26, 3)), 'dusk');
  assert.equal(resolveTheme({ ...settings, autoDusk: false }, new Date(2026, 8, 26, 20)), 'sage');
});
