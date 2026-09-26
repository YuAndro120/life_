// Разбор текста «расскажите о себе» на устройстве: регион, работа, жильё, авто, интересы, что не показывать.
// Текст никуда не отправляется. Правила по словам; результат показывается плашками, любую можно убрать.
import { alnumTokens, normalize } from './text.js';
import { findRegion, regionTitle } from './regions.js';
import { normalizeWord, MAX_WORDS } from './markers.js';
import {
  AGES, GENDERS, HOUSINGS, OCCUPATIONS, SELLS, TOPICS, WORKS, genderTitle, housingTitle, occupationTitle, sellsTitle, topicTitle, workTitle,
} from './taxonomy.js';
import { toggleInterest } from './filter.js';

const OCCUPATION_WORDS = {
  it: ['программист', 'разработчик', 'тестировщик', 'айти', 'devops', 'сисадмин', 'frontend', 'backend', 'разрабатываю'],
  trade: ['продавец', 'торгу', 'торговл', 'магазин', 'маркетплейс', 'wildberries', 'вайлдберриз', 'ozon', 'озон', 'менеджер по продажам'],
  food: ['повар', 'кафе', 'ресторан', 'бариста', 'общепит', 'кондитер', 'пекар', 'официант', 'гостиниц', 'отел'],
  education: ['учител', 'преподават', 'педагог', 'воспитател', 'репетитор'],
  health: ['врач', 'медсестр', 'медик', 'фармацевт', 'стоматолог', 'клиник', 'аптек'],
  construction: ['строител', 'ремонт', 'прораб', 'архитектор', 'отделк', 'монтаж'],
  transport: ['таксист', 'дальнобойщик', 'курьер', 'логист', 'перевозк', 'грузоперевозк', 'экспедитор'],
  industry: ['завод', 'инженер', 'производств', 'токар', 'сварщик', 'фабрик', 'цех'],
  agriculture: ['фермер', 'агроном', 'сельск', 'хозяйств', 'животновод', 'земледел'],
  finance: ['бухгалтер', 'банкир', 'финансист', 'страховщик', 'аудитор', 'экономист', 'налоговик'],
  legal: ['юрист', 'адвокат', 'нотариус', 'юрисконсульт', 'следователь', 'прокурор'],
  publicService: ['госслужащ', 'чиновник', 'полицейск', 'военнослужащ', 'бюджетник', 'муниципал', 'росгвард'],
  beauty: ['парикмахер', 'барбер', 'маникюр', 'косметолог', 'массаж', 'салон', 'визажист', 'бровист', 'лашмейкер'],
  creative: ['дизайнер', 'фотограф', 'блогер', 'журналист', 'художник', 'музыкант', 'видеограф', 'копирайтер', 'контент'],
};

const SELLS_WORDS = {
  marked: ['маркировк', 'обув', 'одежд', 'духи', 'шин', 'молочк', 'велосипед', 'фотоаппарат', 'лекарств'],
  alcohol: ['алкогол', 'пиво', 'вино', 'сигарет', 'табак', 'вейп', 'кальян'],
  food: ['продукты', 'продуктов', 'выпечк', 'фермерск', 'доставк еды', 'торты', 'сладост'],
  online: ['маркетплейс', 'wildberries', 'вайлдберриз', 'ozon', 'озон', 'интернет-магазин', 'авито', 'онлайн-торговл'],
  services: ['услуг', 'консультац', 'мастер', 'обслуживан'],
  transport: ['перевозк', 'грузоперевозк', 'такси', 'доставк'],
  rent: ['сдаю', 'посуточ'],
  goods: ['товар', 'продаю', 'торгую', 'опт', 'розниц'],
};

const TOPIC_WORDS = {
  space: ['космос', 'космичес', 'наса', 'nasa', 'роскосмос', 'астроном', 'ракет', 'spacex'],
  science: ['наук', 'учен', 'физик', 'биолог', 'хими', 'исследован'],
  tech_ai: ['технолог', 'нейросет', 'ии', 'ai', 'айти', 'it', 'программир', 'гаджет', 'стартап'],
  sport: ['спорт', 'футбол', 'хоккей', 'теннис', 'баскетбол', 'бокс', 'формул'],
  culture: ['кино', 'фильм', 'музык', 'театр', 'книг', 'литератур', 'искусств', 'выставк', 'культур'],
  showbiz: ['шоубиз', 'звезд', 'знаменитост'],
  crypto: ['крипт', 'биткоин', 'bitcoin', 'блокчейн'],
  health: ['здоровь', 'медицин', 'врач', 'лекарств'],
  education: ['образован', 'школ', 'университет', 'егэ', 'обучени'],
  economy: ['экономик', 'бизнес', 'инвестиц', 'рынк'],
  finance: ['финанс', 'вклад', 'кредит', 'акци', 'курс'],
  housing: ['недвижим', 'жкх'],
  transport: ['транспорт', 'метро', 'авиа', 'самолет', 'поезд'],
  politics: ['политик', 'выбор', 'депутат', 'власт'],
  crime: ['криминал', 'преступлен', 'происшествия'],
};

const NEGATIVE_CUES = ['не люблю', 'не хочу', 'не интересует', 'не интересно', 'надоел', 'надоела', 'надоело', 'надоели', 'не читаю', 'без ', 'устал от', 'устала от', 'бесит', 'бесят'];
const STOP_WORDS = new Set([
  'и', 'а', 'но', 'или', 'в', 'на', 'с', 'по', 'про', 'о', 'об', 'от', 'до', 'для', 'как', 'что', 'это', 'все', 'весь', 'надоел', 'надоела',
  'надоело', 'надоели', 'бесит', 'бесят', 'устал', 'устала', 'люблю', 'хочу', 'очень', 'сильно', 'новости', 'новостей', 'новость', 'любые',
  'разные', 'такие', 'тоже', 'уже', 'просто', 'чтобы', 'когда',
]);

const hasStem = (words, stems) => words.some((w) => stems.some((s) => w.startsWith(s)));
const topicsIn = (words) => {
  const out = new Set();
  for (const w of words) {
    for (const [topic, stems] of Object.entries(TOPIC_WORDS)) {
      // Короткие основы («ии», «it») должны совпадать целиком, остальные — по началу слова.
      if (stems.some((s) => (s.length <= 3 ? w === s : w.startsWith(s)))) out.add(topic);
    }
  }
  return out;
};

/** Делит текст на предложения; в отрицательные попадает то, что идёт после подсказок вроде «не люблю». */
function split(text) {
  const positive = [], negative = [];
  for (const sentence of text.split(/[.!?;\n]+/).filter((s) => s.trim())) {
    let handled = false;
    for (const cue of NEGATIVE_CUES) {
      const i = sentence.indexOf(cue);
      if (i >= 0) {
        positive.push(sentence.slice(0, i));
        negative.push(sentence.slice(i + cue.length));
        handled = true;
        break;
      }
    }
    if (!handled) positive.push(sentence);
  }
  return { positive: positive.join('. '), negative: negative.join('. ') };
}

function negativeTerms(negative) {
  const out = [];
  for (const w of alnumTokens(negative)) {
    if (w === 'но' || w === 'зато') break;
    if (STOP_WORDS.has(w) || w.length < 3) continue;
    if (!out.includes(w)) out.push(w);
    if (out.length === 6) break;
  }
  return out;
}

const stem = (word) => (word.length >= 6 ? word.slice(0, -1) : word);

function regionIn(positive) {
  // «Живу в Казани, работаю в Москве»: предпочитаем место после «живу».
  for (const cue of ['живу', 'проживаю', 'из города', 'родом из', 'переехал в', 'переехала в']) {
    const i = positive.indexOf(cue);
    if (i >= 0) {
      const item = findRegion(positive.slice(i + cue.length));
      if (item) return item.code;
    }
  }
  return findRegion(positive)?.code ?? null;
}

const emptyParse = () => ({
  regionCode: null, gender: null, work: [], housing: [], occupations: [], sells: [], drives: null, interests: [], mutedTopics: [], blockedWords: [],
});

export function parseAbout(raw) {
  const r = emptyParse();
  const text = normalize(raw);
  if (!text.trim()) return r;
  const { positive, negative } = split(text);

  r.regionCode = regionIn(positive);
  const words = alnumTokens(positive);
  const has = (stems) => hasStem(words, stems);
  const exact = (list) => words.some((w) => list.includes(w));
  const push = (list, x) => { if (!list.includes(x)) list.push(x); };

  if (exact(['ип']) || has(['предпринимател'])) push(r.work, 'ip');
  if (has(['самозанят'])) push(r.work, 'selfemployed');
  if (has(['студент', 'учусь', 'школьник'])) push(r.work, 'student');
  if (positive.includes('по найму') || has(['наемн', 'сотрудник'])) push(r.work, 'employee');

  for (const [id, stems] of Object.entries(OCCUPATION_WORDS)) if (has(stems)) push(r.occupations, id);

  // Что продаёт: только если человек сам говорит, что торгует или оказывает услуги, либо он ИП или самозанятый.
  const sellingCue = has(['продаю', 'торгую', 'занимаюсь', 'бизнес', 'предлагаю', 'оказываю', 'изготавливаю', 'выпекаю', 'сдаю']);
  if (sellingCue || r.work.includes('ip') || r.work.includes('selfemployed')) {
    for (const [id, stems] of Object.entries(SELLS_WORDS)) if (has(stems)) push(r.sells, id);
  }
  // Без явного статуса тот, кто говорит о своём деле, ведёт его как предприниматель.
  if (sellingCue && r.sells.length && !r.work.some((w) => ['ip', 'selfemployed', 'employee'].includes(w))) push(r.work, 'ip');

  if (has(['снимаю', 'аренд'])) push(r.housing, 'renter');
  if (positive.includes('своя квартира') || positive.includes('свое жилье') || has(['собственник', 'владелец'])) push(r.housing, 'owner');
  if (has(['ипотек'])) push(r.housing, 'mortgage');

  if (text.includes('не вожу') || text.includes('без машины') || text.includes('без авто') || text.includes('нет машины')) r.drives = false;
  else if (has(['вожу', 'водител', 'автомобил', 'машин']) || positive.includes('за рулем')) r.drives = true;

  if (has(['мужчин', 'парень'])) r.gender = 'male';
  else if (has(['женщин', 'девушк']) || exact(['мама'])) r.gender = 'female';

  r.interests = [...topicsIn(words)];
  for (const term of negativeTerms(negative)) {
    const matched = [...topicsIn([term])];
    if (!matched.length) {
      const w = normalizeWord(stem(term));
      if (w && !r.blockedWords.includes(w)) r.blockedWords.push(w);
    } else {
      for (const t of matched) {
        push(r.mutedTopics, t);
        r.interests = r.interests.filter((x) => x !== t);
      }
    }
  }
  return r;
}

/** Плашки для показа: где, кто, что интересно, что скрыть. id нужен, чтобы убрать плашку. */
export function aboutChips(r) {
  const chips = [];
  const add = (kind, key, title) => chips.push({ id: `${kind}:${key}`, kind, key, title });
  if (r.regionCode && regionTitle(r.regionCode)) add('region', r.regionCode, regionTitle(r.regionCode));
  if (r.gender) add('gender', r.gender, genderTitle(r.gender));
  for (const w of WORKS) if (r.work.includes(w.id)) add('work', w.id, workTitle(w.id));
  for (const o of OCCUPATIONS) if (r.occupations.includes(o.id)) add('occupation', o.id, occupationTitle(o.id));
  for (const x of SELLS) if (r.sells.includes(x.id)) add('sells', x.id, `Продаю: ${sellsTitle(x.id).toLowerCase()}`);
  for (const h of HOUSINGS) if (r.housing.includes(h.id)) add('housing', h.id, housingTitle(h.id));
  if (r.drives !== null) add('drives', String(r.drives), r.drives ? 'Вожу авто' : 'Не вожу');
  for (const t of TOPICS) if (r.interests.includes(t.id)) add('interest', t.id, topicTitle(t.id));
  for (const t of TOPICS) if (r.mutedTopics.includes(t.id)) add('mute', t.id, `Скрыть: ${topicTitle(t.id)}`);
  for (const w of r.blockedWords) add('word', w, `Скрыть слово: ${w}`);
  return chips;
}

/** Разбор без плашек из `dismissed` (их убрал пользователь). */
export function withoutChips(r, dismissed) {
  const out = { ...r, work: [...r.work], housing: [...r.housing], occupations: [...r.occupations], sells: [...r.sells], interests: [...r.interests], mutedTopics: [...r.mutedTopics], blockedWords: [...r.blockedWords] };
  for (const chip of aboutChips(r)) {
    if (!dismissed.has(chip.id)) continue;
    switch (chip.kind) {
      case 'region': out.regionCode = null; break;
      case 'gender': out.gender = null; break;
      case 'work': out.work = out.work.filter((x) => x !== chip.key); break;
      case 'occupation': out.occupations = out.occupations.filter((x) => x !== chip.key); break;
      case 'sells': out.sells = out.sells.filter((x) => x !== chip.key); break;
      case 'housing': out.housing = out.housing.filter((x) => x !== chip.key); break;
      case 'drives': out.drives = null; break;
      case 'interest': out.interests = out.interests.filter((x) => x !== chip.key); break;
      case 'mute': out.mutedTopics = out.mutedTopics.filter((x) => x !== chip.key); break;
      default: out.blockedWords = out.blockedWords.filter((x) => x !== chip.key);
    }
  }
  return out;
}

const union = (a, b) => [...new Set([...a, ...b])];

/** Дополняет профиль и настройки: найденное добавляется, ранее выбранное не стирается. Возвращает {profile, prefs}. */
export function applyAbout(r, profile, prefs) {
  const p = { ...profile };
  if (r.regionCode) p.regionCode = r.regionCode;
  if (r.gender) p.gender = r.gender;
  p.work = union(p.work, r.work);
  p.housing = union(p.housing, r.housing);
  p.occupations = union(p.occupations, r.occupations);
  p.sells = union(p.sells, r.sells);
  if (r.drives !== null) p.drives = r.drives;
  let q = { ...prefs };
  for (const t of r.interests) if (!q.interests.includes(t)) q = toggleInterest(q, t);
  for (const t of r.mutedTopics) if (!q.interests.includes(t) && !q.stopTopics.includes(t)) q = { ...q, stopTopics: [...q.stopTopics, t] };
  for (const w of r.blockedWords) if (!q.blockedWords.includes(w) && q.blockedWords.length < MAX_WORDS) q = { ...q, blockedWords: [...q.blockedWords, w] };
  return { profile: p, prefs: q };
}

/** Строка «Лента для: …» из профиля и интересов; null, если сказать нечего. */
export function profileSummary(profile, prefs) {
  const parts = [];
  if (regionTitle(profile.regionCode)) parts.push(regionTitle(profile.regionCode));
  for (const w of WORKS) if (profile.work.includes(w.id)) parts.push(w.title);
  for (const o of OCCUPATIONS) if (profile.occupations.includes(o.id)) parts.push(o.title);
  if (profile.drives === true) parts.push('водитель');
  parts.push(...TOPICS.filter((t) => prefs.interests.includes(t.id)).slice(0, 3).map((t) => t.title));
  return parts.length ? parts.slice(0, 5).join(' · ') : null;
}
export { AGES, GENDERS };
