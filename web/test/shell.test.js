import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const root = new URL('..', import.meta.url).pathname;
const walk = (dir) => readdirSync(dir).flatMap((n) => (statSync(join(dir, n)).isDirectory() ? walk(join(dir, n)) : [join(dir, n)]));

test('все модули приложения предзагружаются параллельно (modulepreload в index.html)', () => {
  const html = readFileSync(join(root, 'index.html'), 'utf8');
  const listed = [...html.matchAll(/rel="modulepreload" href="([^"]+)"/g)].map((m) => m[1]).sort();
  const actual = walk(join(root, 'js')).filter((f) => f.endsWith('.js')).map((f) => `/${relative(root, f)}`).sort();
  assert.deepEqual(listed, actual, 'после добавления модуля обновите список modulepreload в index.html');
});

test('шрифты в WOFF2 и не тяжелее 400 КБ суммарно; TTF в вебе не остались', () => {
  const files = readdirSync(join(root, 'fonts'));
  assert.ok(!files.some((f) => f.endsWith('.ttf')), 'TTF должны быть заменены на WOFF2');
  const total = files.filter((f) => f.endsWith('.woff2')).reduce((n, f) => n + statSync(join(root, 'fonts', f)).size, 0);
  assert.ok(total < 400 * 1024, `шрифты весят ${Math.round(total / 1024)} КБ`);
  const css = readFileSync(join(root, 'css/app.css'), 'utf8');
  for (const f of files.filter((x) => x.endsWith('.woff2'))) assert.ok(css.includes(f), `шрифт ${f} не подключён в CSS`);
});

test('сервис-воркер содержит метки версии и списка файлов для сервера', () => {
  const sw = readFileSync(join(root, 'sw.js'), 'utf8');
  assert.ok(sw.includes("'__BUILD__'") && sw.includes("'__ASSETS__'"));
});
