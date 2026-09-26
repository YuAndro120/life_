// Подбор законов под профиль (раздел 7 plan.md). Чистые функции.
import { compareDays, parseDay } from './dates.js';
import { occupationTitle, sellsTitle, ageTitle } from './taxonomy.js';
import { regionTitle } from './regions.js';

export const ALL_TAG = 'all';

/** Мужчина 18–30: возрастные группы грубее, поэтому берём их целиком. */
export const isMilitaryRegistered = (p) => p.gender === 'male' && ['u20', '20_25', '26_35'].includes(p.age);

export function audienceTags(p) {
  const tags = new Set();
  if (p.gender) tags.add(`gender:${p.gender}`);
  if (p.age) tags.add(`age:${p.age}`);
  for (const w of p.work) {
    tags.add(`work:${w}`);
    // В онбординге нет вопроса про налоговый режим: ИП получает и УСН-законы (лучше лишнее, чем пропущенный срок).
    if (w === 'ip') tags.add('work:ip_usn');
  }
  for (const h of p.housing) tags.add(`housing:${h}`);
  for (const o of p.occupations) tags.add(`industry:${o}`);
  for (const x of p.sells) tags.add(`sells:${x}`);
  if (p.drives === true) tags.add('transport:driver');
  if (p.regionCode) tags.add(`region:${p.regionCode}`);
  if (isMilitaryRegistered(p)) tags.add('military:registered');
  return tags;
}

/** Закон касается пользователя: регион подходит и есть пересечение тегов. Отрасль и товары уточняют выбор. */
export function matches(law, tags, regionCode) {
  if (law.region_code && law.region_code !== regionCode) return false;
  const lawTags = new Set(law.audience_tags);
  for (const prefix of ['industry:', 'sells:']) {
    const specific = [...lawTags].filter((t) => t.startsWith(prefix));
    const mine = [...tags].filter((t) => t.startsWith(prefix));
    if (specific.length && mine.length && !specific.some((t) => mine.includes(t))) return false;
  }
  // Тег «all» решает, только когда других тегов нет: модель ставит его вместе с конкретными, и тогда важнее они.
  const specific = [...lawTags].filter((t) => t !== ALL_TAG);
  if (!specific.length) return lawTags.has(ALL_TAG);
  return specific.some((t) => tags.has(t));
}

export function lawOrder(a, b) {
  const da = parseDay(a.dates?.effective), db = parseDay(b.dates?.effective);
  if (da && db) return compareDays(da, db) || a.id.localeCompare(b.id);
  if (da) return -1;
  if (db) return 1;
  return a.id.localeCompare(b.id);
}

/** Законы про пользователя по возрастанию даты вступления в силу (без даты — в конце). asOf отсекает уже вступившие. */
export function relevantLaws(laws, profile, asOf = null) {
  const tags = audienceTags(profile);
  return laws
    .filter((l) => matches(l, tags, profile.regionCode))
    .filter((l) => {
      const eff = parseDay(l.dates?.effective);
      return !asOf || !eff || compareDays(eff, asOf) >= 0;
    })
    .sort(lawOrder);
}

// --- Подписи ---
export function labelForTag(tag) {
  const fixed = {
    all: 'Всех', 'work:employee': 'Работающие по найму', 'work:ip': 'ИП', 'work:ip_usn': 'ИП на УСН', 'work:selfemployed': 'Самозанятые',
    'work:student': 'Студенты', 'housing:renter': 'Снимаю жильё', 'housing:owner': 'Владельцы жилья', 'housing:mortgage': 'Ипотека',
    'transport:driver': 'Водители', 'military:registered': 'Воинский учёт', 'gender:male': 'Мужчины', 'gender:female': 'Женщины',
    'age:u20': 'До 20 лет', 'age:20_25': '20–25 лет', 'age:26_35': '26–35 лет', 'age:36_50': '36–50 лет', 'age:50p': 'Старше 50',
  };
  if (fixed[tag]) return fixed[tag];
  if (tag.startsWith('industry:')) return occupationTitle(tag.slice(9));
  if (tag.startsWith('sells:')) return `Продают: ${sellsTitle(tag.slice(6)).toLowerCase()}`;
  return tag.startsWith('region:') ? 'Регион' : tag;
}

/** Короткая подпись закона: первая метка, совпавшая с профилем, иначе первая содержательная. */
export function lawLabel(law, profileTags) {
  const tags = law.audience_tags.filter((t) => !t.startsWith('region:'));
  const matched = tags.find((t) => profileTags.has(t));
  return labelForTag(matched ?? tags[0] ?? 'all');
}

/** «Почему тебе»: ответы профиля, из-за которых закон попал в блок. */
export function lawReason(law, profile) {
  const tags = audienceTags(profile);
  const lawTags = new Set(law.audience_tags);
  const specific = [...lawTags].filter((t) => t !== ALL_TAG);
  if (lawTags.has(ALL_TAG) && !specific.length) return 'Закон касается всех.';
  const answers = [];
  const add = (s) => { if (!answers.includes(s)) answers.push(s); };
  for (const tag of law.audience_tags) {
    if (!tags.has(tag)) continue;
    const fixed = {
      'work:ip': 'ИП', 'work:ip_usn': 'ИП', 'work:employee': 'По найму', 'work:selfemployed': 'Самозанятый', 'work:student': 'Студент',
      'housing:renter': 'Снимаю', 'housing:owner': 'Своё жильё', 'housing:mortgage': 'Ипотека', 'transport:driver': 'Вожу авто',
      'gender:male': 'Мужчина', 'gender:female': 'Женщина',
    };
    if (fixed[tag]) add(fixed[tag]);
    else if (tag === 'military:registered') add(`Мужчина ${profile.age ? ageTitle(profile.age) : ''}`.trim());
    else if (tag.startsWith('industry:')) add(occupationTitle(tag.slice(9)));
    else if (tag.startsWith('sells:')) add(`Продаю: ${sellsTitle(tag.slice(6)).toLowerCase()}`);
    else if (tag.startsWith('age:') && profile.age) add(ageTitle(profile.age));
    else if (tag.startsWith('region:') && regionTitle(profile.regionCode)) add(regionTitle(profile.regionCode));
  }
  if (!answers.length) return 'Подобрано по вашему профилю.';
  return `В профиле указано: ${answers.map((a) => `«${a}»`).join(', ')}.`;
}
