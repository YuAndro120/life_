import test from 'node:test';
import assert from 'node:assert/strict';
import { story, law, DEFAULT_PREFS, DEFAULT_PROFILE } from './helpers.js';
import { isAllowed, buildEdition, storyOrder, applyAction, revertAction, ACTIONS, toggleInterest, toggleStopTopic } from '../js/core/filter.js';
import { isWar, isWarEnd, isWarLaw, isPolitical, normalizeWord, matchesWords } from '../js/core/markers.js';
import { matches, audienceTags, relevantLaws } from '../js/core/audience.js';
import { findRegion, searchRegions, REGIONS } from '../js/core/regions.js';
import { parseAbout, applyAbout, withoutChips, aboutChips, profileSummary } from '../js/core/about.js';

const prefs = (over = {}) => ({ ...DEFAULT_PREFS(), infoTypes: ['fact', 'official', 'opinion', 'forecast', 'rumor'], stopTopics: [], ...over });
const profile = (over = {}) => ({ ...DEFAULT_PROFILE(), ...over });

// ---------- СВО ----------
test('СВО скрыто по умолчанию и включается обратно', () => {
  assert.equal(DEFAULT_PREFS().hideWar, true);
  assert.equal(isAllowed(story({ topic: 'incidents', title: 'Вооружённые силы России нанесли удары по объектам инфраструктуры Украины' }), prefs()), false);
  assert.equal(isAllowed(story({ topic: 'culture', title: 'Участник СВО получил награду' }), prefs()), false);
  assert.equal(isAllowed(story({ topic: 'incidents', title: 'Над Белгородом сбили беспилотники' }), prefs()), false);
  assert.equal(isAllowed(story({ topic: 'culture', title: 'Участник СВО получил награду' }), prefs({ hideWar: false })), true);
});

test('единственное исключение — официальное заявление об окончании СВО', () => {
  const p = prefs({ stopTopics: ['politics'] });
  assert.equal(isAllowed(story({ topic: 'politics', info_type: 'official', title: 'Кремль официально объявил о завершении специальной военной операции' }), p), true);
  assert.equal(isAllowed(story({ topic: 'politics', info_type: 'rumor', title: 'Источники сообщили о скором окончании СВО' }), p), false);
  assert.equal(isAllowed(story({ info_type: 'official', title: 'Завершено строительство моста' }), p), true);
  assert.equal(isWarEnd(story({ info_type: 'fact', title: 'Завершение СВО' })), false);
});

test('обычные новости не задеты фильтром СВО', () => {
  assert.equal(isAllowed(story({ topic: 'finance', title: 'Ключевая ставка сохранена на уровне 16 процентов' }), prefs()), true);
  assert.equal(isAllowed(story({ topic: 'sport', title: 'Свой первый матч сыграл футбольный клуб' }), prefs()), true);
  assert.equal(isWar(story({ title: 'Свободное место' })), false);
});

test('законы про участников СВО скрываются переключателем', () => {
  assert.ok(isWarLaw(law({ title: 'Сохранение права на жильё по соцнайму семьям погибших участников СВО' })));
  assert.ok(isWarLaw(law({ title: 'Уточнён перечень лиц, относимых к ветеранам и инвалидам боевых действий' })));
  assert.ok(isWarLaw(law({ title: 'Льготы детям', who_affected: 'Дети погибших военнослужащих' })));
  assert.ok(!isWarLaw(law({ title: 'Меняется срок уведомлений об авансовых платежах' })));
  const feed = { stories: [], stats: {} };
  const laws = [law({ id: 'a', title: 'Жильё семьям погибших участников СВО' }), law({ id: 'd', title: 'Авансовые платежи для ИП' })];
  const ids = (hideWar) => buildEdition({ number: 1, feed, laws, profile: profile(), prefs: prefs({ hideWar }), window: { start: null, end: '2026-09-25T05:00:00Z' }, today: { y: 2026, m: 9, d: 25 } }).laws.map((l) => l.id);
  assert.deepEqual(ids(true), ['d']);
  assert.deepEqual(ids(false), ['a', 'd']);
});

// ---------- Политика и слова ----------
test('запрет политики скрывает и ошибочно классифицированные сюжеты', () => {
  const p = prefs({ stopTopics: ['politics'] });
  const s = story({ topic: 'culture', title: 'Владимир Путин выступил на форуме объединённых культур' });
  assert.equal(isAllowed(s, p), false);
  assert.equal(isAllowed(s, prefs()), true);
  assert.equal(isPolitical(story({ title: 'В парке открыли новый трамплин для прыжков' })), false);
  assert.equal(isPolitical(story({ title: 'Минюст включил проект в реестр иноагентов' })), true);
  assert.equal(isAllowed(story({ topic: 'culture', title: 'Путин посетил выставку' }), prefs({ stopTopics: ['politics'], interests: ['culture'] })), true);
});

test('слова пользователя: по началу слова и фразой', () => {
  const p = prefs({ hideWar: false, blockedWords: ['футбол'] });
  assert.equal(isAllowed(story({ topic: 'sport', title: 'Футболист перешёл в новый клуб' }), p), false);
  assert.equal(isAllowed(story({ topic: 'sport', title: 'Матч', summary: 'Прошёл футбольный матч' }), p), false);
  assert.equal(isAllowed(story({ topic: 'sport', title: 'Хоккей: матч закончился вничью' }), p), true);
  const ph = prefs({ hideWar: false, blockedWords: ['ключевая ставка'] });
  assert.equal(isAllowed(story({ topic: 'finance', title: 'Ключевая ставка снижена' }), ph), false);
  assert.equal(isAllowed(story({ topic: 'finance', title: 'Ставка по вкладам выросла' }), ph), true);
  assert.equal(normalizeWord('  Ёлка  '), 'елка');
  assert.equal(normalizeWord('а'), '');
  assert.equal(normalizeWord('Ключевая   Ставка'), 'ключевая ставка');
  assert.equal(normalizeWord('я'.repeat(60)), '');
  assert.equal(matchesWords(story(), []), false);
});

// ---------- Регионы ----------
test('чужие регионы скрыты по умолчанию', () => {
  const regional = (code) => story({ topic: 'city', title: 'Новость города', region_code: code });
  const p = prefs({ hideWar: false, homeRegion: '16' });
  assert.equal(isAllowed(regional(null), p), true);
  assert.equal(isAllowed(regional('16'), p), true);
  assert.equal(isAllowed(regional('78'), p), false);
  assert.equal(isAllowed(regional('16'), { ...p, homeRegion: null }), false);
  assert.equal(isAllowed(regional('78'), { ...p, hideOtherRegions: false }), true);
});

test('регионы находятся по падежам и не выдумываются', () => {
  const code = (t) => parseAbout(t).regionCode;
  assert.equal(code('Я из Москвы'), '77');
  assert.equal(code('живу в Санкт-Петербурге'), '78');
  assert.equal(code('живу в питере'), '78');
  assert.equal(code('Екатеринбург'), '66');
  assert.equal(code('живу в Нижнем Новгороде'), '52');
  assert.equal(code('живу в Новом Уренгое'), '89');
  assert.equal(code('Красноярский край'), '24');
  assert.equal(code('Работаю в Москве, но живу в Туле'), '71');
  assert.equal(code('Я читал книгу про Казанову'), null);
  assert.equal(code('Ответ тверже камня'), null);
  assert.equal(code('Люблю читать'), null);
  assert.equal(REGIONS.length, 85);
  assert.equal(new Set(REGIONS.map((r) => r.code)).size, 85);
  assert.deepEqual(searchRegions('каз').map((r) => r.code), ['16']);
  assert.equal(findRegion('нет такого'), null);
});

// ---------- Профиль из текста ----------
test('пример из онбординга разбирается', () => {
  const r = parseAbout('Живу в Казани, ИП на патенте, езжу на машине, люблю космос, увлекаюсь футболом');
  assert.equal(r.regionCode, '16');
  assert.deepEqual(r.work, ['ip']);
  assert.equal(r.drives, true);
  assert.deepEqual([...r.interests].sort(), ['space', 'sport']);
});

test('отрицания становятся скрытием, а не интересами', () => {
  const r = parseAbout('Люблю космос. Не люблю футбол и политику, надоели сплетни');
  assert.deepEqual(r.interests, ['space']);
  assert.deepEqual([...r.mutedTopics].sort(), ['politics', 'sport']);
  assert.deepEqual(r.blockedWords, ['сплетн']);
  const h = parseAbout('Снимаю квартиру');
  assert.deepEqual(h.interests, []);
  assert.deepEqual(h.housing, ['renter']);
});

test('работа, сфера и что продаёт', () => {
  assert.deepEqual(parseAbout('Студент, снимаю квартиру, самозанятый').work.sort(), ['selfemployed', 'student']);
  assert.deepEqual(parseAbout('Я программист, живу в Москве').occupations, ['it']);
  const b = parseAbout('Занимаюсь торговлей на Wildberries, продаю одежду и обувь');
  assert.ok(b.sells.includes('online') && b.sells.includes('marked'));
  assert.deepEqual(b.work, ['ip']);
  assert.ok(b.occupations.includes('trade'));
  const s = parseAbout('Студент, снимаю квартиру, заказываю доставку');
  assert.deepEqual(s.sells, []);
  assert.deepEqual(s.work, ['student']);
  assert.equal(parseAbout('Не вожу, езжу на метро').drives, false);
  assert.equal(parseAbout('').regionCode, null);
});

test('плашки убираются, применение не стирает выбранное', () => {
  const r = parseAbout('Живу в Казани, ИП, люблю космос');
  const t = withoutChips(r, new Set(['region:16', 'interest:space']));
  assert.equal(t.regionCode, null);
  assert.deepEqual(t.interests, []);
  assert.deepEqual(t.work, ['ip']);
  const { profile: pr, prefs: pf } = applyAbout(parseAbout('Живу в Казани, ИП, люблю космос, не люблю футбол'), profile({ work: ['employee'] }), DEFAULT_PREFS());
  assert.equal(pr.regionCode, '16');
  assert.deepEqual(pr.work.sort(), ['employee', 'ip']);
  assert.ok(pf.interests.includes('space'));
  assert.ok(pf.stopTopics.includes('sport') && pf.stopTopics.includes('politics'));
  assert.ok(aboutChips(r).length >= 3);
});

test('финансы и право — разные сферы', () => {
  assert.deepEqual(parseAbout('Я юрист').occupations, ['legal']);
  assert.deepEqual(parseAbout('Работаю бухгалтером').occupations, ['finance']);
  assert.deepEqual(parseAbout('Я адвокат, а жена бухгалтер').occupations.sort(), ['finance', 'legal']);
});

test('строка «Лента для»', () => {
  const p = profile({ work: ['ip'], occupations: ['trade'], drives: true, regionCode: '16' });
  assert.equal(profileSummary(p, prefs({ interests: ['space'] })), 'Татарстан · ИП · Торговля · водитель · Космос');
  assert.equal(profileSummary(profile(), prefs()), null);
});

// ---------- Законы ----------
test('отрасль и товары уточняют отбор законов', () => {
  const m = (tags, p) => matches(law({ audience_tags: tags }), audienceTags(p), p.regionCode);
  const it = profile({ work: ['employee'], occupations: ['it'] });
  assert.equal(m(['work:employee', 'industry:health'], it), false);
  assert.equal(m(['work:employee', 'industry:it'], it), true);
  assert.equal(m(['work:employee'], it), true);
  assert.equal(m(['work:employee', 'industry:health'], profile({ work: ['employee'] })), true);
  const seller = profile({ work: ['ip'], sells: ['services'] });
  assert.equal(m(['work:ip', 'sells:alcohol'], seller), false);
  assert.equal(m(['work:ip', 'sells:services'], seller), true);
});

test('тег all не перебивает конкретные теги', () => {
  const m = (tags, p) => matches(law({ audience_tags: tags }), audienceTags(p), null);
  const p = profile({ work: ['employee'] });
  assert.equal(m(['all', 'housing:mortgage'], p), false);
  assert.equal(m(['all', 'work:employee'], p), true);
  assert.equal(m(['all'], p), true);
});

test('региональный закон виден только своему региону; ИП получает УСН', () => {
  const l = law({ audience_tags: ['work:ip'], region_code: '16' });
  assert.equal(matches(l, audienceTags(profile({ work: ['ip'], regionCode: '16' })), '16'), true);
  assert.equal(matches(l, audienceTags(profile({ work: ['ip'], regionCode: '78' })), '78'), false);
  assert.equal(matches(law({ audience_tags: ['work:ip_usn'] }), audienceTags(profile({ work: ['ip'] })), null), true);
  assert.deepEqual(relevantLaws([law({ id: 'b', dates: { effective: null } }), law({ id: 'a' })], profile(), { y: 2026, m: 9, d: 25 }).map((x) => x.id), ['a', 'b']);
});

// ---------- Порядок и действия ----------
test('порядок: интересы, официальное, источники, свежесть', () => {
  const a = story({ id: 'a', info_type: 'fact', source_count: 5 });
  const b = story({ id: 'b', info_type: 'official', source_count: 1 });
  const c = story({ id: 'c', topic: 'space', info_type: 'rumor' });
  assert.deepEqual([a, b, c].sort((x, y) => storyOrder(x, y, ['space'])).map((s) => s.id), ['c', 'b', 'a']);
});

test('«Не интересно» применяется, отменяется и не дублируется', () => {
  const p = DEFAULT_PREFS();
  const one = applyAction(p, ACTIONS.hideStory('st_01'));
  assert.equal(one.changed, true);
  assert.deepEqual(one.prefs.hiddenStories, ['st_01']);
  assert.equal(applyAction(one.prefs, ACTIONS.hideStory('st_01')).changed, false);
  assert.deepEqual(revertAction(one.prefs, ACTIONS.hideStory('st_01')).hiddenStories, []);
  const boost = applyAction(p, ACTIONS.boostTopic('politics')).prefs;
  assert.ok(boost.interests.includes('politics') && !boost.stopTopics.includes('politics'));
  assert.ok(toggleStopTopic(toggleInterest(p, 'sport'), 'sport').stopTopics.includes('sport'));
});

test('выпуск: скрытый сюжет исчезает, лимит режет, тяжёлые сворачиваются', () => {
  const stories = [story({ id: 'x1' }), story({ id: 'x2', topic: 'finance' }), story({ id: 'x3', heaviness: 'heavy', topic: 'health' })];
  const args = { number: 3, feed: { stories, stats: { posts_total: 10, ads_hidden: 2 } }, laws: [], profile: profile(), window: { start: null, end: '2026-09-25T05:00:00Z' }, today: { y: 2026, m: 9, d: 25 } };
  const e = buildEdition({ ...args, prefs: prefs({ hideWar: false }) });
  assert.deepEqual(e.stories.map((s) => s.id).sort(), ['x1', 'x2']);
  assert.deepEqual(e.foldedHeavy.map((s) => s.id), ['x3']);
  const hidden = buildEdition({ ...args, prefs: prefs({ hideWar: false, hiddenStories: ['x1'] }) });
  assert.deepEqual(hidden.stories.map((s) => s.id), ['x2']);
  const limited = buildEdition({ ...args, prefs: prefs({ hideWar: false, storyLimit: 1 }) });
  assert.equal(limited.stats.trimmed, 1);
  assert.equal(e.stats.postsTotal, 10);
});
