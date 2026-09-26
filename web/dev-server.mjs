// Локальный сервер для разработки: раздаёт web/ и проксирует /v1/ на боевой API (или на API_ORIGIN).
// Запуск: node web/dev-server.mjs   → http://localhost:8090
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { extname, join, normalize } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('.', import.meta.url));
const origin = process.env.API_ORIGIN ?? 'https://shtil.tech';
const port = Number(process.env.PORT ?? 8090);
const types = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.json': 'application/json', '.webmanifest': 'application/manifest+json', '.png': 'image/png', '.ttf': 'font/ttf', '.txt': 'text/plain; charset=utf-8' };

createServer(async (req, res) => {
  const url = new URL(req.url, 'http://localhost');
  if (url.pathname.startsWith('/v1/')) {
    const upstream = await fetch(origin + url.pathname + url.search, { headers: { Accept: 'application/json' } }).catch(() => null);
    res.writeHead(upstream?.status ?? 502, { 'Content-Type': 'application/json' });
    return res.end(upstream ? Buffer.from(await upstream.arrayBuffer()) : '{"error":"нет связи с API"}');
  }
  const rel = normalize(decodeURIComponent(url.pathname)).replace(/^(\.\.[/\\])+/, '');
  const file = join(root, rel === '/' || rel === '\\' ? 'index.html' : rel);
  try {
    const body = await readFile(file);
    res.writeHead(200, { 'Content-Type': types[extname(file)] ?? 'application/octet-stream', 'Cache-Control': 'no-cache' });
    res.end(body);
  } catch {
    res.writeHead(404); res.end('нет такого файла');
  }
}).listen(port, () => console.log(`http://localhost:${port} → API ${origin}`));
