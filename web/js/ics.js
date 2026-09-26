// Напоминание о вступлении закона в силу как файл календаря (.ics): в вебе нет надёжных локальных уведомлений,
// поэтому напоминание живёт в календаре телефона и приходит даже при закрытом приложении.
import { parseDay, two } from './core/dates.js';

const esc = (s) => String(s).replaceAll('\\', '\\\\').replaceAll(';', '\\;').replaceAll(',', '\\,').replaceAll('\n', '\\n');
const ymd = (d) => `${d.y}${two(d.m)}${two(d.d)}`;

function fold(line) {
  // Строки .ics не длиннее 75 октетов; кириллица — 2 октета на символ, поэтому режем по 34 символа.
  const parts = [];
  let rest = line;
  while (rest.length > 34) { parts.push(rest.slice(0, 34)); rest = ` ${rest.slice(34)}`; }
  parts.push(rest);
  return parts.join('\r\n');
}

/** Событие на весь день вступления закона в силу, с напоминаниями за неделю и за день. Возвращает null, если даты нет. */
export function lawIcs(law, now = new Date()) {
  const eff = parseDay(law.dates?.effective);
  if (!eff) return null;
  const end = new Date(Date.UTC(eff.y, eff.m - 1, eff.d + 1));
  const endDay = { y: end.getUTCFullYear(), m: end.getUTCMonth() + 1, d: end.getUTCDate() };
  const stamp = `${now.getUTCFullYear()}${two(now.getUTCMonth() + 1)}${two(now.getUTCDate())}T${two(now.getUTCHours())}${two(now.getUTCMinutes())}${two(now.getUTCSeconds())}Z`;
  const actions = law.actions?.length ? `\\nЧто сделать: ${law.actions.map(esc).join('; ')}` : '';
  const alarm = (trigger, text) => ['BEGIN:VALARM', 'ACTION:DISPLAY', `DESCRIPTION:${esc(text)}`, `TRIGGER:${trigger}`, 'END:VALARM'];
  const lines = [
    'BEGIN:VCALENDAR', 'VERSION:2.0', 'PRODID:-//Штиль//Законы//RU', 'CALSCALE:GREGORIAN',
    'BEGIN:VEVENT', `UID:${law.id}@shtil.tech`, `DTSTAMP:${stamp}`,
    `DTSTART;VALUE=DATE:${ymd(eff)}`, `DTEND;VALUE=DATE:${ymd(endDay)}`,
    `SUMMARY:${esc(`Вступает в силу: ${law.title}`)}`,
    `DESCRIPTION:${esc(law.what_changed)}${actions}`,
    ...(law.official_url ? [`URL:${law.official_url}`] : []),
    ...alarm('-P7D', `Через неделю: ${law.title}`), ...alarm('-P1D', `Завтра: ${law.title}`),
    'END:VEVENT', 'END:VCALENDAR',
  ];
  return `${lines.map(fold).join('\r\n')}\r\n`;
}
