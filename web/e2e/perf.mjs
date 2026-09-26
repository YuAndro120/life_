// Замер запуска на настоящем Chrome с имитацией мобильной сети: холодный (пустой кэш) и тёплый (повторный) запуск.
// Запуск: node web/e2e/perf.mjs [адрес]   (по умолчанию https://shtil.tech/)
import { spawn } from 'node:child_process';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const URL_ = process.argv[2] ?? 'https://shtil.tech/';
const CHROME = process.env.CHROME ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const DEBUG = 9444;
const chrome = spawn(CHROME, ['--headless=new', '--disable-gpu', `--remote-debugging-port=${DEBUG}`, `--user-data-dir=${mkdtempSync(join(tmpdir(), 'shtil-perf-'))}`, '--window-size=430,900', 'about:blank'], { stdio: 'ignore' });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let wsUrl;
for (let i = 0; i < 50 && !wsUrl; i += 1) {
  try { wsUrl = (await (await fetch(`http://localhost:${DEBUG}/json`)).json()).find((t) => t.type === 'page')?.webSocketDebuggerUrl; } catch { await sleep(200); }
}
const ws = new WebSocket(wsUrl);
await new Promise((r) => ws.addEventListener('open', r));
let id = 1;
const pending = new Map();
let requests = 0, bytes = 0;
ws.addEventListener('message', (e) => {
  const m = JSON.parse(e.data);
  if (m.id && pending.has(m.id)) { pending.get(m.id)(m); pending.delete(m.id); }
  if (m.method === 'Network.responseReceived') requests += 1;
  if (m.method === 'Network.loadingFinished') bytes += m.params.encodedDataLength ?? 0;
});
const send = (method, params = {}) => new Promise((r) => { const i = id++; pending.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
const ev = async (expression) => (await send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true })).result.result.value;
await send('Network.enable');
await send('Page.enable');
await send('Network.emulateNetworkConditions', { offline: false, latency: 120, downloadThroughput: (1.6 * 1024 * 1024) / 8, uploadThroughput: (750 * 1024) / 8 });

async function load(label) {
  requests = 0; bytes = 0;
  const t0 = Date.now();
  await send('Page.navigate', { url: URL_ });
  let firstScreen = null;
  while (Date.now() - t0 < 30000) {
    if (await ev(`document.querySelector('.screen') !== null && document.body.innerText.length > 50`)) { firstScreen = Date.now() - t0; break; }
    await sleep(25);
  }
  await sleep(1500);
  const nav = await ev(`(() => { const n = performance.getEntriesByType('navigation')[0]; return Math.round(n.loadEventEnd); })()`);
  console.log(`${label.padEnd(8)} первый экран: ${firstScreen} мс · load: ${nav} мс · запросов: ${requests} · по сети: ${(bytes / 1024).toFixed(0)} КБ`);
}
await send('Network.clearBrowserCache');
await load('холодный');
await load('тёплый');
await load('тёплый 2');
chrome.kill();
process.exit(0);
