# infra

## Dev

```bash
docker compose -f infra/docker-compose.dev.yml up -d   # Postgres 16 + pgvector, только 127.0.0.1:5432
cd backend
make migrate        # goose up
make seed           # тестовые данные (даты относительно «сейчас»); очищает таблицы данных
make run            # API на :8080
make test-integration   # тесты на живой БД (мигрирует и засевает)
make sync-sources   # каталог seeds/sources.yaml -> БД (включаются только проверенные по реестрам)
make collect        # один проход сборщика по источникам
make collect-loop   # сборщик в цикле
```

`DATABASE_URL` по умолчанию `postgres://life:life_dev@127.0.0.1:5432/life?sslmode=disable` (см. `backend/Makefile`).
Быстрая проверка: `curl -s localhost:8080/v1/feed | jq .stats`.

## Приложение → сервер

- Симулятор ходит на `http://127.0.0.1:8080` (build setting `API_BASE_URL` в `ios/Life/project.yml`).
- Реальный iPhone: `127.0.0.1` — это сам телефон. Запусти API на Mac, узнай его адрес в сети (`ipconfig getifaddr en0`)
  и передай в схеме Xcode аргумент `-lifeAPI http://<адрес>:8080` (Edit Scheme → Run → Arguments), либо поменяй `API_BASE_URL`.
  Mac и iPhone должны быть в одной Wi-Fi сети, macOS-файрвол должен пускать входящие на порт 8080.
- Без сервера приложение покажет последний сохранённый выпуск с плашкой «Нет связи».
- `-lifeSource fixtures` (DEBUG) — работать на встроенных фикстурах без сервера.

Prod (docker-compose.prod.yml, Caddyfile) — фаза 7.
