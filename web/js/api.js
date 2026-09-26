// Клиент API. Сервер отдаёт всем одну и ту же ленту; подбор делает устройство. Кэшированием и ETag занимается браузер.
export async function getJson(path, { fetchFn = fetch, signal } = {}) {
  const response = await fetchFn(path, { headers: { Accept: 'application/json' }, cache: 'no-cache', signal });
  if (!response.ok) throw new Error(`Сервер ответил ${response.status}`);
  return response.json();
}

export async function loadContent(options) {
  const [feed, lawsResponse] = await Promise.all([getJson('/v1/feed', options), getJson('/v1/laws', options)]);
  return { feed, laws: lawsResponse.laws ?? [] };
}
