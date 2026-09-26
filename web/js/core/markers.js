// Правила по словам: политика, СВО, слова пользователя. Работают только на устройстве.
import { alnumTokens, letterTokens, normalize } from './text.js';

// Текст сюжета разбирается один раз: выпуск пересобирается при каждом действии, а сюжетов сотни.
const parsed = new WeakMap();
function parse(story) {
  let p = parsed.get(story);
  if (!p) {
    const text = normalize(`${story.title} ${story.summary}`);
    p = { text, words: letterTokens(text), full: null };
    parsed.set(story, p);
  }
  return p;
}
function parseFull(story) {
  const p = parse(story);
  if (!p.full) {
    const text = normalize(`${story.title} ${story.summary} ${story.meaning ?? ''}`);
    p.full = { text, tokens: alnumTokens(text) };
  }
  return p.full;
}

const hasStem = (words, list) => words.some((w) => list.some(([stem, tail]) => w.startsWith(stem) && w.length - stem.length <= tail));

// --- Политика: политические фигуры и институты, органы власти, санкции, реестры иноагентов ---
const POLITICAL = [
  ['путин', 2], ['кремл', 2], ['песков', 2], ['лавров', 2], ['мишустин', 2], ['зеленск', 4], ['трамп', 2], ['госдум', 2], ['володин', 2],
  ['макрон', 2], ['шольц', 2], ['эрдоган', 2], ['байден', 2], ['депутат', 3], ['сенат', 3], ['конгресс', 3], ['мид', 0], ['иноагент', 4],
  ['нежелательн', 4], ['санкц', 4], ['минюст', 2], ['минобороны', 0],
];
export const isPolitical = (story) => hasStem(parse(story).words, POLITICAL);

// --- СВО ---
const WAR_WORDS = [
  ['сво', 0], ['всу', 0], ['бпла', 1], ['беспилотн', 5], ['спецопераци', 3], ['донбасс', 3], ['днр', 0], ['лнр', 0], ['мобилизац', 3],
  ['обстрел', 3], ['фронт', 2], ['украин', 4], ['зеленск', 4], ['минобороны', 0],
];
const WAR_PHRASES = ['специальной военной операции', 'специальная военная операция', 'вооруженные силы', 'вооруженных сил', 'боевые действия', 'боевых действий'];
const END_WORDS = ['заверш', 'окончен', 'окончани', 'прекращ'];
const LAW_PHRASES = [
  'специальной военной операции', 'боевых действий', 'вооруженного вторжения', 'погибших военнослужащих', 'погибших участников',
  'погибших сотрудников', 'участников сво', 'участникам сво', 'участники сво',
];

export function isWar(story) {
  const { text, words } = parse(story);
  if (WAR_PHRASES.some((p) => text.includes(p))) return true;
  return hasStem(words, WAR_WORDS);
}

/** Официальное заявление о том, что СВО закончилась: единственное исключение из «всё про СВО скрыто». */
export function isWarEnd(story) {
  if (story.info_type !== 'official') return false;
  const { text, words } = parse(story);
  const mentionsWar = text.includes('специальной военной операции') || text.includes('специальная военная операция') || words.includes('сво');
  return mentionsWar && words.some((w) => END_WORDS.some((e) => w.startsWith(e)));
}

/** Закон про участников СВО, ветеранов боевых действий и семьи погибших. */
export function isWarLaw(law) {
  const text = normalize(`${law.title} ${law.what_changed} ${law.who_affected}`);
  return LAW_PHRASES.some((p) => text.includes(p)) || letterTokens(text).includes('сво');
}

// --- Слова пользователя ---
export const MAX_WORDS = 50;
export const MAX_WORD_LENGTH = 40;

/** Слово для хранения: строчные, «ё» → «е», лишние пробелы убраны; пустая строка, если не подходит. */
export function normalizeWord(raw) {
  const squeezed = normalize(raw).trim().split(/\s+/).filter(Boolean).join(' ');
  return squeezed.length >= 2 && squeezed.length <= MAX_WORD_LENGTH ? squeezed : '';
}

/** Слово ищется по началу слов («футбол» найдёт «футболист»), фраза из нескольких слов — как подстрока. */
export function matchesWords(story, words) {
  if (!words.length) return false;
  const { text, tokens } = parseFull(story);
  return words.some((w) => (w.includes(' ') ? text.includes(w) : tokens.some((t) => t.startsWith(w))));
}
