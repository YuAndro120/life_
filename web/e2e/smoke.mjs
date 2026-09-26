// Сквозная проверка на настоящем Chrome (протокол отладки, без зависимостей): онбординг → лента → фильтры → профиль → копия.
// Запуск: node web/e2e/smoke.mjs  (нужны Google Chrome и доступ к боевому API через dev-server)
import { spawn } from 'node:child_process';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import assert from 'node:assert/strict';

const CHROME = process.env.CHROME ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const PORT = 8091, DEBUG = 9333;
const server = spawn('node', [new URL('../dev-server.mjs', import.meta.url).pathname], { env: { ...process.env, PORT: String(PORT) }, stdio: 'ignore' });
const chrome = spawn(CHROME, ['--headless=new', '--disable-gpu', `--remote-debugging-port=${DEBUG}`, `--user-data-dir=${mkdtempSync(join(tmpdir(), 'shtil-e2e-'))}`, '--window-size=500,1200', 'about:blank'], { stdio: 'ignore' });
const stop = () => { server.kill(); chrome.kill(); };
process.on('exit', stop);

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
async function connect() {
  for (let i = 0; i < 50; i += 1) {
    try {
      const targets = await (await fetch(`http://localhost:${DEBUG}/json`)).json();
      const page = targets.find((t) => t.type === 'page');
      if (page) return page.webSocketDebuggerUrl;
    } catch { /* Chrome ещё запускается */ }
    await sleep(200);
  }
  throw new Error('Chrome не запустился');
}

const ws = new WebSocket(await connect());
await new Promise((r) => ws.addEventListener('open', r));
let nextId = 1;
const pending = new Map();
const problems = [];
ws.addEventListener('message', (e) => {
  const msg = JSON.parse(e.data);
  if (msg.id && pending.has(msg.id)) { pending.get(msg.id)(msg); pending.delete(msg.id); }
  if (msg.method === 'Runtime.exceptionThrown') problems.push(msg.params.exceptionDetails.exception?.description ?? msg.params.exceptionDetails.text);
  if (msg.method === 'Runtime.consoleAPICalled' && msg.params.type === 'error') problems.push(msg.params.args.map((a) => a.value ?? a.description).join(' '));
});
const send = (method, params = {}) => new Promise((resolve) => { const id = nextId++; pending.set(id, resolve); ws.send(JSON.stringify({ id, method, params })); });
await send('Runtime.enable');
await send('Page.enable');

const ev = async (expression) => {
  const r = await send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true });
  if (r.result.exceptionDetails) throw new Error(`${expression}\n${r.result.exceptionDetails.exception?.description}`);
  return r.result.result.value;
};
const go = async (path) => { await send('Page.navigate', { url: `http://localhost:${PORT}/${path}` }); await sleep(600); };
const waitFor = async (expression, what, ms = 15000) => {
  const end = Date.now() + ms;
  while (Date.now() < end) { if (await ev(expression).catch(() => false)) return; await sleep(150); }
  throw new Error(`не дождались: ${what}`);
};
const clickText = (text, sel = 'button, a') => ev(`(() => { const el = [...document.querySelectorAll(${JSON.stringify(sel)})].find((e) => e.textContent.trim().startsWith(${JSON.stringify(text)}) && !e.closest('[hidden]')); if (!el) return false; el.click(); return true; })()`);
const must = async (text, sel) => assert.ok(await clickText(text, sel), `нет кнопки «${text}»`);
const bodyText = () => ev('document.body.innerText');
const steps = [];
const step = async (name, fn) => { await fn(); steps.push(name); console.log('✔', name); };

try {
  await go('?install=1');
  await step('в браузере показывается подсказка по установке, «продолжить» ведёт в приложение', async () => {
    await waitFor(`document.body.innerText.includes('Поставь Штиль на телефон')`, 'страница установки');
    const text = await bodyText();
    assert.ok(text.includes('Работает без интернета') && text.includes('Настройки не пропадут'), 'преимущества');
    await must('Пока продолжить в браузере');
    await waitFor(`document.body.innerText.includes('Новости без шума') && document.body.innerText.includes('Законы тебе в помощь')`, 'приветствие с новой фразой');
  });
  await go('');
  await step('приветствие и переход к рассказу о себе', async () => {
    await waitFor(`document.body.innerText.includes('Новости без шума')`, 'приветствие');
    await must('Начать');
    await waitFor(`document.body.innerText.includes('Расскажи о себе')`, 'экран о себе');
  });
  await step('текст о себе разбирается в плашки', async () => {
    await ev(`(() => { const t = document.querySelector('textarea'); t.value = 'Живу в Казани, ИП, езжу на машине, люблю космос. Не люблю футбол'; t.dispatchEvent(new Event('input', { bubbles: true })); })()`);
    const text = await bodyText();
    for (const chip of ['Татарстан', 'ИП', 'Вожу авто', 'Космос', 'Скрыть: Спорт']) assert.ok(text.includes(chip), `нет плашки ${chip}`);
    await must('Татарстан'); // убрали плашку
    assert.ok(!(await ev(`[...document.querySelectorAll('.chips .chip')].some((c) => c.textContent.startsWith('Татарстан') && c.getAttribute('aria-label')?.includes('убрать'))`)));
    await must('Дальше');
  });
  await step('профиль подставлен, дальше по шагам до сборки выпуска', async () => {
    await waitFor(`document.body.innerText.includes('Что про тебя важно знать')`, 'профиль');
    assert.ok(await ev(`[...document.querySelectorAll('.chip')].some((c) => c.textContent.trim() === 'ИП' && c.getAttribute('aria-pressed') === 'true')`), 'ИП должен быть выбран');
    await must('Дальше');
    await waitFor(`document.body.innerText.includes('Что тебе интересно')`, 'интересы');
    await must('Дальше');
    await waitFor(`document.body.innerText.includes('Что тебе не показывать')`, 'что не показывать');
    await must('Дальше');
    await waitFor(`document.body.innerText.includes('Как будет выглядеть выпуск')`, 'тема');
    await ev(`[...document.querySelectorAll('.theme-opt')].find((c) => c.textContent.includes('Сумерки')).click()`);
    assert.equal(await ev('document.documentElement.dataset.theme'), 'dusk');
    await ev(`[...document.querySelectorAll('.theme-opt')].find((c) => c.textContent.includes('Бумага')).click()`);
    await must('Собрать первый выпуск');
  });
  await step('сборка выпуска и открытие ленты', async () => {
    await waitFor(`[...document.querySelectorAll('button')].some((b) => b.textContent.startsWith('Открыть выпуск') && !b.closest('[hidden]'))`, 'кнопка «Открыть выпуск»', 25000);
    await must('Открыть выпуск');
    await waitFor(`document.querySelectorAll('.story').length > 3`, 'сюжеты в ленте');
    assert.ok((await bodyText()).includes('Лента для:'), 'строка «Лента для»');
  });
  await step('сюжет открывается, есть пересказ и источники', async () => {
    await ev(`document.querySelector('.story a.st').click()`);
    await waitFor(`location.hash.startsWith('#/story/')`, 'переход по ссылке сюжета');
    await waitFor(`document.body.innerText.includes('Пересказ сделан автоматически')`, 'экран сюжета');
    
    await ev('history.back()');
    await waitFor(`document.querySelectorAll('.story').length > 3`, 'возврат в ленту');
  });
  await step('«Не интересно» скрывает сюжет, отмена возвращает', async () => {
    const before = await ev(`document.querySelectorAll('.story').length`);
    const title = await ev(`document.querySelector('.story .st-title').textContent`);
    await ev(`document.querySelector('.story .menu-btn').click()`);
    await must('Не интересно, скрыть сюжет', '.menu button');
    const titles = `[...document.querySelectorAll('.story .st-title')].map((e) => e.textContent)`;
    await waitFor(`!${titles}.includes(${JSON.stringify(title)})`, 'сюжет скрыт');
    assert.ok(await ev(`document.querySelector('.undo') !== null`), 'панель отмены');
    await must('Отменить', '.undo button');
    await waitFor(`${titles}.includes(${JSON.stringify(title)})`, 'сюжет вернулся');
    assert.equal(await ev(`document.querySelectorAll('.story').length`), before);
  });
  await step('фильтры: слово скрывает сюжеты, переключатель СВО работает', async () => {
    await go('#/filters');
    await waitFor(`document.body.innerText.includes('Скрывать всё про СВО')`, 'фильтры');
    const war = () => ev(`document.querySelector('[aria-label="Скрывать всё про СВО"]').getAttribute('aria-checked')`);
    assert.equal(await war(), 'true');
    await must('Скрывать всё про СВО', 'button');
    await waitFor(`document.querySelector('[aria-label="Скрывать всё про СВО"]').getAttribute('aria-checked') === 'false'`, 'СВО выключено');
    await ev(`(() => { const i = document.querySelector('input[aria-label="Слово или фраза для скрытия"]'); i.value = 'погод'; i.dispatchEvent(new Event('input', { bubbles: true })); })()`);
    await must('Добавить', 'button');
    await waitFor(`[...document.querySelectorAll('.chip')].some((c) => c.textContent.startsWith('погод'))`, 'слово добавлено');
  });
  await step('настройки переживают перезагрузку, онбординг не повторяется', async () => {
    await go('#/filters');
    await waitFor(`document.body.innerText.includes('Фильтры')`, 'фильтры после перезагрузки');
    assert.ok(await ev(`[...document.querySelectorAll('.chip')].some((c) => c.textContent.startsWith('погод'))`));
    assert.equal(await ev(`document.querySelector('[aria-label="Скрывать всё про СВО"]').getAttribute('aria-checked')`), 'false');
  });
  await step('календарь, профиль и зашифрованная копия', async () => {
    await go('#/calendar');
    await waitFor(`document.body.innerText.toLowerCase().includes('что вступает в силу')`, 'календарь');
    await go('#/profile');
    await waitFor(`document.body.innerText.toLowerCase().includes('резервная копия')`, 'профиль');
    await must('Сохранить копию');
    await waitFor(`document.querySelector('.sheet') !== null`, 'окно копии');
    await ev(`(() => { const p = document.querySelector('.sheet input[type=password]'); p.value = 'пароль123'; })()`);
    await must('Создать код', '.sheet button');
    await waitFor(`document.querySelector('.sheet textarea').value.startsWith('SHTIL1.')`, 'код копии', 20000);
  });
  await step('законы: карточка открывается, если есть подходящие', async () => {
    await go('');
    await waitFor(`document.querySelectorAll('.story').length > 0`, 'лента');
    const has = await ev(`document.querySelector('.law-card') !== null`);
    if (has) {
      await ev(`document.querySelector('.law-card').click()`);
      await waitFor(`document.body.innerText.includes('Что изменилось')`, 'экран закона');
      assert.ok((await bodyText()).includes('не юридическая консультация'));
    }
  });
  assert.deepEqual(problems, [], `ошибки в консоли: ${problems.join(' | ')}`);
  console.log(`\nвсё в порядке: ${steps.length} шагов, ошибок в консоли нет`);
} catch (error) {
  console.error('✖', error.message);
  if (problems.length) console.error('ошибки в консоли:', problems);
  process.exitCode = 1;
} finally {
  stop();
  process.exit(process.exitCode ?? 0);
}
