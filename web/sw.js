// Сервис-воркер: оболочка приложения кэшируется целиком и отдаётся без сети (запуск без единого запроса).
// Сервер подставляет BUILD (хэш содержимого web/) и список файлов ASSETS: при любом изменении кода воркер обновляется сам.
const BUILD = '__BUILD__';
const ASSETS = JSON.parse('__ASSETS__');
const CACHE = `shtil-${BUILD}`;
// Шрифты тем «Шалфей» кэшируются при первом использовании, чтобы не тянуть их всем на установке.
const LAZY = /Golos|Spectral/;

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE).then((cache) => cache.addAll(ASSETS.filter((a) => !LAZY.test(a)).map((a) => new Request(a, { cache: 'reload' })))).then(() => self.skipWaiting()),
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))).then(() => self.clients.claim()));
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  if (request.method !== 'GET') return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin || url.pathname.startsWith('/v1/')) return; // API ходит в сеть, последнюю ленту хранит приложение
  if (request.mode === 'navigate') {
    event.respondWith(caches.match('/index.html').then((hit) => hit ?? fetch(request)));
    return;
  }
  event.respondWith(
    caches.match(request, { ignoreSearch: true }).then((hit) => hit ?? fetch(request).then((response) => {
      if (response.ok) { const copy = response.clone(); caches.open(CACHE).then((c) => c.put(request, copy)); }
      return response;
    })),
  );
});
