// Сервис-воркер: оболочка приложения работает без сети. Статику берём из сети, а при её отсутствии — из кэша;
// ответы API (/v1/) не кэшируем здесь: последнюю ленту хранит само приложение.
const CACHE = 'shtil-shell-v1';
const SHELL = ['/', '/index.html', '/manifest.webmanifest', '/css/app.css', '/js/app.js', '/icons/icon-192.png', '/fonts/Onest.ttf'];

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL)).then(() => self.skipWaiting()));
});

self.addEventListener('activate', (event) => {
  event.waitUntil(caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))).then(() => self.clients.claim()));
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  if (request.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/v1/')) return;
  event.respondWith(
    fetch(request).then((response) => {
      if (response.ok) { const copy = response.clone(); caches.open(CACHE).then((c) => c.put(request, copy)); }
      return response;
    }).catch(() => caches.match(request).then((hit) => hit ?? (request.mode === 'navigate' ? caches.match('/index.html') : Response.error()))),
  );
});
