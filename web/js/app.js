// Точка входа: хранилище, маршрутизация по адресу (#/…), загрузка ленты, тема, «Отменить», сервис-воркер.
import { createStore } from './store.js';
import { loadContent } from './api.js';
import { resolveTheme } from './theme.js';
import { actionMessage } from './core/filter.js';
import { h, clear } from './ui/dom.js';
import { todayScreen } from './ui/today.js';
import { storyScreen } from './ui/story.js';
import { lawScreen } from './ui/law.js';
import { calendarScreen } from './ui/calendar.js';
import { filtersScreen } from './ui/filters.js';
import { profileScreen } from './ui/profile.js';
import { onboardingScreen } from './ui/onboarding.js';
import { installScreen } from './ui/install.js';
import { isStandalone } from './platform.js';
import { installDebugPanel } from './debug.js';

const memoryStorage = () => { const m = new Map(); return { getItem: (k) => m.get(k) ?? null, setItem: (k, v) => m.set(k, String(v)), removeItem: (k) => m.delete(k) }; };
function pickStorage() {
  try { localStorage.setItem('shtil.probe', '1'); localStorage.removeItem('shtil.probe'); return localStorage; } catch { return memoryStorage(); }
}

const params = new URLSearchParams(location.search);
const isDev = ['localhost', '127.0.0.1'].includes(location.hostname);
const store = createStore(pickStorage());
if (isDev && params.get('demo')) {
  // Только для разработки на localhost: готовый профиль, чтобы смотреть экраны без онбординга.
  store.setProfile({ regionCode: '16', work: ['ip'], occupations: ['trade'], drives: true, onboardingCompleted: true });
  store.setPrefs({ interests: ['space'] });
  store.completeOnboarding();
}
const appEl = document.getElementById('app');
const undoEl = document.getElementById('undo');
const REFRESH_AFTER_MS = 5 * 60 * 1000;
let lastRefresh = 0;
let refreshing = null;

const ctx = {
  store, local: {}, theme: 'paper',
  now: () => (isDev && params.get('now') ? new Date(params.get('now')) : new Date()),
  nav: (path) => { if (location.hash === `#${path}`) render(false); else location.hash = `#${path}`; },
  rerender: () => render(true),
  refresh: () => refresh(true),
};

// Установка: сайт нужно поставить на экран «Домой». В браузере показываем подсказку; «продолжить» запоминается до закрытия вкладки.
// На localhost (разработка) подсказка выключена, её можно посмотреть через ?install=1.
let installPrompt = null;
const installSkipped = () => { try { return sessionStorage.getItem('shtil.skipInstall') === '1'; } catch { return ctx.local.skipInstall === true; } };
function skipInstall() { try { sessionStorage.setItem('shtil.skipInstall', '1'); } catch { ctx.local.skipInstall = true; } render(false); }
const needsInstall = () => !isStandalone() && !installSkipped() && (!isDev || params.get('install') === '1');
window.addEventListener('beforeinstallprompt', (event) => { event.preventDefault(); installPrompt = event; if (needsInstall()) render(true); });
window.addEventListener('appinstalled', () => { installPrompt = null; render(false); });
async function runInstall() {
  if (!installPrompt) return;
  installPrompt.prompt();
  await installPrompt.userChoice.catch(() => {});
  installPrompt = null;
  render(true);
}

const TABS = [['/', 'Сегодня'], ['/calendar', 'Календарь'], ['/filters', 'Фильтры'], ['/profile', 'Профиль']];

function currentRoute() {
  const path = location.hash.replace(/^#/, '') || '/';
  const [, name = '', param = ''] = path.split('/');
  return { path, name, param: decodeURIComponent(param) };
}

const tabsBar = (active) => h('div', { class: 'bar tabs' }, h('div', { class: 'col tabbar' },
  h('nav', { 'aria-label': 'Разделы' }, TABS.map(([path, title]) => h('a', { href: `#${path}`, 'aria-current': path === active ? 'page' : null }, title)))));

function applyTheme() {
  const now = ctx.now();
  ctx.theme = isDev && params.get('theme') ? params.get('theme') : resolveTheme(store.state.settings, now);
  document.documentElement.dataset.theme = ctx.theme;
  const color = { paper: '#f2f1ec', sage: '#edf0ea', dusk: '#1c1d22' }[ctx.theme];
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', color);
}

let lastPath = null;
function render(keepScroll = false) {
  applyTheme();
  const route = currentRoute();
  const onboarding = !store.state.profile.onboardingCompleted;
  let screen;
  let tab = null;
  try {
    if (needsInstall()) {
      screen = installScreen(ctx, { onContinue: skipInstall, prompt: installPrompt, onInstall: runInstall });
    } else if (onboarding) {
      if (isDev && params.get('step') && !ctx.local.onb) ctx.local.onb = { step: params.get('step'), about: params.get('about') ?? '', dismissed: new Set() };
      screen = onboardingScreen(ctx);
    } else {
      switch (route.name) {
        case 'story': screen = storyScreen(ctx, route.param); break;
        case 'law': screen = lawScreen(ctx, route.param); break;
        case 'calendar': screen = calendarScreen(ctx); tab = '/calendar'; break;
        case 'filters': screen = filtersScreen(ctx); tab = '/filters'; break;
        case 'profile': screen = profileScreen(ctx); tab = '/profile'; break;
        default: screen = todayScreen(ctx); tab = '/';
      }
    }
  } catch (error) {
    console.error(error);
    screen = h('section', { class: 'screen' }, h('div', { class: 'empty' }, 'Что-то пошло не так. Обновите страницу.'));
  }
  // Каркас: прокручивается только внутренняя область; нижняя панель (кнопки шага или вкладки) стоит на месте.
  const footer = screen.querySelector(':scope > .footer');
  footer?.remove();
  const scroller = h('div', { class: 'scroll', id: 'scroll' }, h('div', { class: 'col' }, screen));
  const bar = footer ? h('div', { class: 'bar' }, h('div', { class: 'col' }, footer)) : tab !== null ? tabsBar(tab) : null;
  const previous = document.getElementById('scroll');
  const keep = keepScroll && lastPath === route.path && previous ? previous.scrollTop : 0;
  clear(appEl);
  appEl.append(scroller);
  if (bar) appEl.append(bar);
  scroller.scrollTop = keep;
  lastPath = route.path;
  renderUndo();
}

let undoTimer = null;
function renderUndo() {
  clear(undoEl);
  clearTimeout(undoTimer);
  const action = store.state.undo;
  if (!action) return;
  undoEl.append(h('div', { class: 'undo', role: 'status' }, h('span', null, actionMessage(action)), h('button', { type: 'button', onClick: () => store.undoFeedback() }, 'Отменить')));
  undoTimer = setTimeout(() => store.clearUndo(), 5000);
}

async function refresh(force = false) {
  if (refreshing) return refreshing;
  if (!force && Date.now() - lastRefresh < REFRESH_AFTER_MS) return undefined;
  refreshing = (async () => {
    try {
      store.setContent(await loadContent());
      store.bumpEdition(ctx.now());
      lastRefresh = Date.now();
    } catch (error) {
      store.setLoadFailed(navigator.onLine ? 'Не удалось загрузить выпуск. Попробуйте ещё раз.' : 'Нет соединения с интернетом.');
    } finally { refreshing = null; }
  })();
  return refreshing;
}

// Изменения состояния перерисовывают экран, сохраняя прокрутку. Ввод текста живёт в локальном состоянии экранов и сюда не попадает.
let scheduled = false;
store.subscribe(() => {
  if (scheduled) return;
  scheduled = true;
  queueMicrotask(() => { scheduled = false; render(true); });
});
window.addEventListener('hashchange', () => render(false));
document.addEventListener('visibilitychange', () => { if (!document.hidden) { refresh(); render(true); } });
window.addEventListener('online', () => refresh(true));
setInterval(() => { if (!document.hidden) render(true); }, 10 * 60 * 1000);

if (navigator.storage?.persist) navigator.storage.persist().then((ok) => { ctx.local.persisted = ok; }).catch(() => {});
if ('serviceWorker' in navigator && !isDev) navigator.serviceWorker.register('/sw.js').catch(() => {});

// iOS в режиме прозрачного статус-бара отдаёт странице окно на высоту статус-бара короче экрана (внизу остаётся пустая полоса).
// Если нехватка равна верхней безопасной зоне, растягиваем рамку на весь экран.
function fitFrame() {
  const probe = document.createElement('div');
  probe.style.cssText = 'position:fixed;visibility:hidden;padding-top:env(safe-area-inset-top)';
  document.body.append(probe);
  const safeTop = parseFloat(getComputedStyle(probe).paddingTop) || 0;
  probe.remove();
  const portrait = innerHeight > innerWidth;
  const full = portrait ? Math.max(screen.width, screen.height) : Math.min(screen.width, screen.height);
  const missing = full - innerHeight;
  const stretch = portrait && safeTop > 0 && missing > 0 && missing <= safeTop + 2;
  document.documentElement.style.setProperty('--frame-h', stretch ? `${full}px` : '100%');
}
fitFrame();
window.addEventListener('resize', fitFrame);
window.addEventListener('orientationchange', () => setTimeout(fitFrame, 300));

installDebugPanel();
render(false);
refresh(true).then(() => render(true));
