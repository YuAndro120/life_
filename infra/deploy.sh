#!/usr/bin/env bash
# Деплой Штиль на сервер: образы собираются на Mac под linux/amd64, грузятся на сервер по SSH, стек поднимается.
# Предварительно на сервере один раз выполнить infra/root-setup.sh (пользователь shtilapp, nginx, сертификат).
#   infra/deploy.sh            — выкатить текущую версию
#   HOST=shtil-prod            — SSH-алиас из ~/.ssh/config (по умолчанию shtil-prod)
set -euo pipefail

HOST=${HOST:-shtil-prod}
VERSION=${SHTIL_VERSION:-$(date +%Y%m%d-%H%M)}
ROOT=$(cd "$(dirname "$0")/.." && pwd)
REMOTE=/home/shtilapp/shtil

echo "== версия $VERSION, сервер $HOST"
cd "$ROOT/backend"
for target in api collector worker migrate seed; do
  echo "-- сборка shtil/$target"
  docker buildx build --platform linux/amd64 --target "$target" -t "shtil/$target:$VERSION" --load . >/dev/null
done

echo "== загрузка образов на сервер"
docker save "shtil/api:$VERSION" "shtil/collector:$VERSION" "shtil/worker:$VERSION" "shtil/migrate:$VERSION" "shtil/seed:$VERSION" | gzip | ssh "$HOST" 'gunzip | docker load' | sed 's/^/   /'
if ! ssh "$HOST" 'docker image inspect pgvector/pgvector:pg16 >/dev/null 2>&1'; then
  echo "-- образ PostgreSQL (один раз)"
  docker pull -q --platform linux/amd64 pgvector/pgvector:pg16 >/dev/null
  docker save pgvector/pgvector:pg16 | gzip | ssh "$HOST" 'gunzip | docker load' | sed 's/^/   /'
fi

echo "== конфигурация и запуск"
ssh "$HOST" "mkdir -p $REMOTE"
scp -q "$ROOT/infra/docker-compose.prod.yml" "$HOST:$REMOTE/docker-compose.yml"
# Пароль БД создаётся на сервере при первом деплое и никогда не покидает его.
ssh "$HOST" "cd $REMOTE && { [ -f .env ] || { umask 077; printf 'POSTGRES_PASSWORD=%s\n' \"\$(openssl rand -hex 16)\" > .env; }; } && sed -i '/^SHTIL_VERSION=/d' .env && echo SHTIL_VERSION=$VERSION >> .env && chmod 600 .env"
ssh "$HOST" "cd $REMOTE && docker compose up -d --remove-orphans && docker compose run --rm sync-sources"

echo "== проверка"
for i in $(seq 1 20); do
  if ssh "$HOST" 'curl -fsS localhost:8081/v1/health' 2>/dev/null; then echo; echo "готово"; exit 0; fi
  sleep 3
done
echo "API не ответил на /v1/health за 60 секунд: docker compose logs api"; exit 1
