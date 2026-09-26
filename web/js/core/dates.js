// Русское форматирование дат без Intl, чтобы результат не зависел от локали устройства.
const MONTHS_GEN = ['января', 'февраля', 'марта', 'апреля', 'мая', 'июня', 'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря'];
const MONTHS_SHORT = ['Янв', 'Фев', 'Мар', 'Апр', 'Май', 'Июн', 'Июл', 'Авг', 'Сен', 'Окт', 'Ноя', 'Дек'];
const WEEKDAYS_SHORT = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'];
const WEEKDAYS_FULL = ['Воскресенье', 'Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота'];

export const two = (n) => String(n).padStart(2, '0');

/** «yyyy-MM-dd» или полный ISO-8601 → {y, m, d}; иначе null. */
export function parseDay(s) {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s ?? '');
  if (!m) return null;
  return { y: Number(m[1]), m: Number(m[2]), d: Number(m[3]) };
}

export const dayOf = (date) => ({ y: date.getFullYear(), m: date.getMonth() + 1, d: date.getDate() });
const utc = (d) => Date.UTC(d.y, d.m - 1, d.d);
export const dayKey = (d) => `${d.y}-${two(d.m)}-${two(d.d)}`;

/** Целых дней от `from` до `to`. */
export const daysBetween = (from, to) => Math.round((utc(to) - utc(from)) / 86400000);
export const compareDays = (a, b) => utc(a) - utc(b);
export function addDays(d, n) {
  const t = new Date(Date.UTC(d.y, d.m - 1, d.d + n));
  return { y: t.getUTCFullYear(), m: t.getUTCMonth() + 1, d: t.getUTCDate() };
}

export function plural(n, one, few, many) {
  const n10 = Math.abs(n) % 10, n100 = Math.abs(n) % 100;
  if (n10 === 1 && n100 !== 11) return one;
  if (n10 >= 2 && n10 <= 4 && !(n100 >= 12 && n100 <= 14)) return few;
  return many;
}

export const dayMonth = (d, today) => `${d.d} ${MONTHS_GEN[d.m - 1]}${d.y === today.y ? '' : ' ' + d.y}`;
export const dayMonthYear = (d) => `${d.d} ${MONTHS_GEN[d.m - 1]} ${d.y}`;
export const numeric = (d) => `${two(d.d)}.${two(d.m)}.${d.y}`;
export const numericShort = (d, today) => `${two(d.d)}.${two(d.m)}${d.y === today.y ? '' : '.' + two(d.y % 100)}`;
export const calendarTile = (d, today) => ({ day: two(d.d), month: d.y === today.y ? MONTHS_SHORT[d.m - 1] : `${MONTHS_SHORT[d.m - 1]} ${two(d.y % 100)}` });
export const weekdayIndex = (d) => new Date(utc(d)).getUTCDay();
export const weekdayShort = (d) => WEEKDAYS_SHORT[weekdayIndex(d)];
export const longDay = (d) => `${WEEKDAYS_FULL[weekdayIndex(d)]}, ${d.d} ${MONTHS_GEN[d.m - 1]}`;
export const countdownShort = (days) => (days <= 0 ? 'Сегодня' : `Д–${days}`);
export const daysWords = (n) => `${n} ${plural(n, 'день', 'дня', 'дней')}`;
export function countdownWords(days) {
  if (days < 1) return 'сегодня';
  if (days === 1) return 'завтра';
  if (days < 60) return `через ${daysWords(days)}`;
  const months = Math.floor(days / 30);
  return `через ${months} ${plural(months, 'месяц', 'месяца', 'месяцев')}`;
}
export const clock = (minutes) => `${two(Math.floor(minutes / 60) % 24)}:${two(minutes % 60)}`;
export const editionStamp = (date) => `${weekdayShort(dayOf(date))} ${two(date.getDate())}.${two(date.getMonth() + 1)} — ${two(date.getHours())}:${two(date.getMinutes())}`;
export function greeting(hour) {
  if (hour >= 5 && hour < 12) return 'Доброе утро.';
  if (hour >= 12 && hour < 18) return 'Добрый день.';
  if (hour >= 18 && hour < 23) return 'Добрый вечер.';
  return 'Доброй ночи.';
}

/** Ближайший выпуск строго после `now`: {date, isToday, minutes}. */
export function nextEdition(settings, now) {
  const slots = settings.schedule === 'am' ? [settings.morningMinutes] : settings.schedule === 'pm' ? [settings.eveningMinutes]
    : [settings.morningMinutes, settings.eveningMinutes].sort((a, b) => a - b);
  for (let offset = 0; offset <= 1; offset += 1) {
    for (const minutes of slots) {
      const c = new Date(now.getFullYear(), now.getMonth(), now.getDate() + offset, 0, minutes);
      if (c > now) return { date: c, isToday: offset === 0, minutes };
    }
  }
  return null;
}
export const nextCaption = (settings, now) => {
  const n = nextEdition(settings, now);
  return n ? `${n.isToday ? 'сегодня' : 'завтра'} в ${clock(n.minutes)}` : null;
};
