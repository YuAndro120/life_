// Таксономии из раздела 7 plan.md, общие для сервера и клиентов.
export const TOPICS = [
  ['economy', 'Экономика'], ['finance', 'Финансы'], ['law', 'Право'], ['tech_ai', 'ИИ и технологии'], ['city', 'Город'],
  ['health', 'Здоровье'], ['education', 'Образование'], ['transport', 'Транспорт'], ['housing', 'Жильё'], ['science', 'Наука'],
  ['space', 'Космос'], ['culture', 'Культура'], ['sport', 'Спорт'], ['showbiz', 'Шоу-бизнес'], ['crypto', 'Криптовалюты'],
  ['politics', 'Политика'], ['crime', 'Криминал'], ['incidents', 'Происшествия'], ['disasters', 'Катастрофы'],
].map(([id, title]) => ({ id, title }));

export const topicTitle = (id) => TOPICS.find((t) => t.id === id)?.title ?? id;

export const INFO_TYPES = [
  ['fact', 'Факт'], ['official', 'Решение'], ['opinion', 'Мнение'], ['forecast', 'Прогноз'], ['rumor', 'Неподтверждённое'],
].map(([id, title]) => ({ id, title }));
export const infoTitle = (id) => INFO_TYPES.find((t) => t.id === id)?.title ?? id;

export const COUNTRIES = [
  { id: 'RU', title: 'Россия' }, { id: 'US', title: 'США' }, { id: 'GB', title: 'Великобритания' }, { id: 'EU', title: 'Европа' },
];
export const countryTitle = (id) => COUNTRIES.find((c) => c.id === id)?.title ?? id;

export const GENDERS = [{ id: 'male', title: 'Мужчина' }, { id: 'female', title: 'Женщина' }];
export const AGES = [
  { id: 'u20', title: 'До 20' }, { id: '20_25', title: '20–25' }, { id: '26_35', title: '26–35' }, { id: '36_50', title: '36–50' }, { id: '50p', title: '50+' },
];
export const WORKS = [
  { id: 'employee', title: 'По найму' }, { id: 'ip', title: 'ИП' }, { id: 'selfemployed', title: 'Самозанятый' }, { id: 'student', title: 'Студент' },
];
export const HOUSINGS = [{ id: 'renter', title: 'Снимаю' }, { id: 'owner', title: 'Своё' }, { id: 'mortgage', title: 'Ипотека' }];
export const OCCUPATIONS = [
  ['it', 'IT и разработка'], ['trade', 'Торговля'], ['food', 'Общепит и гостиницы'], ['education', 'Образование'], ['health', 'Медицина'],
  ['construction', 'Строительство и ремонт'], ['transport', 'Транспорт и логистика'], ['industry', 'Производство'],
  ['agriculture', 'Сельское хозяйство'], ['finance', 'Финансы и право'], ['publicService', 'Госслужба и бюджет'],
  ['beauty', 'Красота и услуги'], ['creative', 'Творчество и медиа'],
].map(([id, title]) => ({ id, title }));
export const SELLS = [
  ['goods', 'Товары'], ['marked', 'Маркированные товары'], ['alcohol', 'Алкоголь и табак'], ['food', 'Продукты и еда'],
  ['online', 'Через маркетплейсы'], ['services', 'Услуги'], ['transport', 'Перевозки'], ['rent', 'Аренда и недвижимость'],
].map(([id, title]) => ({ id, title }));

const find = (list, id) => list.find((x) => x.id === id)?.title ?? id;
export const genderTitle = (id) => find(GENDERS, id);
export const ageTitle = (id) => find(AGES, id);
export const workTitle = (id) => find(WORKS, id);
export const housingTitle = (id) => find(HOUSINGS, id);
export const occupationTitle = (id) => find(OCCUPATIONS, id);
export const sellsTitle = (id) => find(SELLS, id);

export const THEMES = [
  { id: 'paper', title: 'Бумага', hint: 'Собранно, как хорошая газета' },
  { id: 'sage', title: 'Шалфей', hint: 'Мягко и спокойно, как утро' },
  { id: 'dusk', title: 'Сумерки', hint: 'Тёмная, бережёт глаза вечером' },
];

/** Профиль и настройки по умолчанию (как в iOS-приложении). */
export const DEFAULT_PROFILE = () => ({
  gender: null, age: null, work: [], housing: [], occupations: [], sells: [], drives: null, regionCode: null, onboardingCompleted: false,
});

export const DEFAULT_PREFS = () => ({
  calmMode: true,
  infoTypes: ['fact', 'official'],
  heavyMode: 'fold',
  maxHeavy: 3,
  stopTopics: ['politics', 'crime'],
  hideAds: true,
  countries: ['RU'],
  interests: [],
  onlyInterests: false,
  mutedSources: [],
  hiddenStories: [],
  hideOtherRegions: true,
  hideWar: true,
  blockedWords: [],
  storyLimit: 15,
  homeRegion: null,
});

export const DEFAULT_SETTINGS = () => ({
  theme: 'paper', autoDusk: true, schedule: 'both', morningMinutes: 8 * 60, eveningMinutes: 19 * 60, backupEnabled: true,
});
