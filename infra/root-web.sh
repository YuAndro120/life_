#!/usr/bin/env bash
# Подключает раздачу веб-приложения Штиль в существующий nginx. Запускать один раз от root на сервере:
#   ssh root@<сервер> 'bash -s' < infra/root-web.sh
# Скрипт: добавляет зону ограничения запросов, заменяет `location / { return 404; }` проксированием на API-контейнер
# во всех серверных блоках shtil.tech (в том числе созданных certbot), проверяет конфигурацию и перезагружает nginx.
# Перед правками сохраняется копия; при ошибке проверки всё откатывается. Скрипт безопасно запускать повторно.
set -euo pipefail

SITE=/etc/nginx/sites-available/shtil.tech
LIMITS=/etc/nginx/conf.d/shtil-limits.conf
STAMP=$(date +%Y%m%d-%H%M%S)

[ -f "$SITE" ] || { echo "нет $SITE: сначала infra/root-setup.sh" >&2; exit 1; }
cp "$SITE" "$SITE.bak-$STAMP"
[ -f "$LIMITS" ] && cp "$LIMITS" "$LIMITS.bak-$STAMP"

grep -q 'zone=shtil_web' "$LIMITS" 2>/dev/null || echo 'limit_req_zone $binary_remote_addr zone=shtil_web:10m rate=30r/s;' >> "$LIMITS"

if grep -q 'zone=shtil_web burst' "$SITE"; then
  echo "веб уже подключён"
else
  perl -0pi -e 's|location / \{\s*return 404;\s*\}|location / {\n        limit_req zone=shtil_web burst=60 nodelay;\n        proxy_pass http://127.0.0.1:8081;\n        proxy_http_version 1.1;\n        proxy_set_header Host \$host;\n        proxy_set_header X-Real-IP \$remote_addr;\n        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;\n        proxy_set_header X-Forwarded-Proto \$scheme;\n        proxy_read_timeout 20s;\n        proxy_connect_timeout 5s;\n    }|g' "$SITE"
fi

if nginx -t; then
  systemctl reload nginx
  echo "готово: https://shtil.tech/"
else
  echo "nginx -t не прошёл, возвращаю прежнюю конфигурацию" >&2
  cp "$SITE.bak-$STAMP" "$SITE"
  [ -f "$LIMITS.bak-$STAMP" ] && cp "$LIMITS.bak-$STAMP" "$LIMITS"
  exit 1
fi
