import test from 'node:test';
import assert from 'node:assert/strict';
import { lawIcs } from '../js/ics.js';
import { law } from './helpers.js';

test('файл календаря: весь день вступления, два напоминания, спецсимволы экранируются', () => {
  const raw = lawIcs(law({ id: 'lw_1', title: 'Закон; с, запятой', what_changed: 'Строка 1\nСтрока 2', actions: ['Подать заявление'], dates: { effective: '2026-10-01' }, official_url: 'https://pravo.gov.ru/x' }), new Date(Date.UTC(2026, 8, 26, 10, 0, 0)));
  const ics = raw.replaceAll('\r\n ', ''); // разворачиваем сложенные строки, как это делает календарь
  assert.match(ics, /DTSTART;VALUE=DATE:20261001/);
  assert.match(ics, /DTEND;VALUE=DATE:20261002/);
  assert.equal((ics.match(/BEGIN:VALARM/g) ?? []).length, 2);
  assert.match(ics, /TRIGGER:-P7D/);
  assert.ok(ics.includes('SUMMARY:Вступает в силу: Закон\\; с\\, запятой'));
  assert.match(ics, /UID:lw_1@shtil.tech/);
  assert.match(ics, /DTSTAMP:20260926T100000Z/);
  assert.ok(raw.endsWith('\r\n'));
  assert.ok(raw.split('\r\n').every((l) => Buffer.byteLength(l) <= 75), 'строки не длиннее 75 байт');
});

test('конец месяца и года считается верно; без даты файла нет', () => {
  assert.match(lawIcs(law({ dates: { effective: '2026-12-31' } })), /DTEND;VALUE=DATE:20270101/);
  assert.equal(lawIcs(law({ dates: { effective: null } })), null);
});
