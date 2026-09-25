#!/usr/bin/env bash
# Разовая настройка сервера для Штиль. Запускать от root:
#   ssh root@<сервер> 'bash -s' < infra/root-setup.sh
# Что делает (ничего не трогает в чужих проектах):
#   1. Создаёт пользователя shtilapp (uid 1500) с одним SSH-ключом и доступом к docker; блокирует пользователя life,
#      созданного ранее с uid 1000 (этот uid совпадает с пользователем внутри контейнера n8n).
#   2. Добавляет swap 1 ГБ (на сервере его нет) — защита соседних проектов от нехватки памяти. Отключить: WITH_SWAP=0.
#   3. Добавляет в nginx отдельный сайт shtil.tech (только /v1/ -> 127.0.0.1:8081) и ограничение частоты запросов.
#      Перед перезагрузкой nginx проверяется конфигурация (nginx -t); при ошибке всё откатывается.
#   4. Выпускает сертификат Let's Encrypt для shtil.tech (certbot --nginx). Почта: CERT_EMAIL=you@example.com.
set -euo pipefail
[ "$(id -u)" = 0 ] || { echo "Нужен root"; exit 1; }
DOMAIN=shtil.tech
KEY='ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAG1t5pwkqcIajP6CZsDYAcue7kRob+pEoprlrY1OXxs life-deploy (Claude Code) 2026-09-25'

echo "== 1/4 пользователь shtilapp"
if ! id shtilapp >/dev/null 2>&1; then
  useradd -m -u 1500 -s /bin/bash shtilapp
fi
install -d -m 700 -o shtilapp -g shtilapp /home/shtilapp/.ssh
grep -qF "$KEY" /home/shtilapp/.ssh/authorized_keys 2>/dev/null || echo "$KEY" >> /home/shtilapp/.ssh/authorized_keys
chown shtilapp:shtilapp /home/shtilapp/.ssh/authorized_keys; chmod 600 /home/shtilapp/.ssh/authorized_keys
getent group docker >/dev/null && usermod -aG docker shtilapp && echo "shtilapp добавлен в группу docker"
if id life >/dev/null 2>&1; then
  rm -f /home/life/.ssh/authorized_keys
  usermod -s /usr/sbin/nologin life
  passwd -l life >/dev/null
  echo "пользователь life заблокирован (ключ удалён)"
fi

echo "== 2/4 swap"
if [ "${WITH_SWAP:-1}" = 1 ] && [ -z "$(swapon --show --noheadings)" ]; then
  fallocate -l 1G /swapfile && chmod 600 /swapfile && mkswap /swapfile >/dev/null && swapon /swapfile
  grep -q '^/swapfile ' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
  echo 'vm.swappiness=10' > /etc/sysctl.d/99-shtil-swappiness.conf && sysctl -q -p /etc/sysctl.d/99-shtil-swappiness.conf
  echo "swap 1 ГБ включён"
else
  echo "swap пропущен (уже есть или WITH_SWAP=0)"
fi

echo "== 3/4 nginx"
cat > /etc/nginx/conf.d/shtil-limits.conf <<'NGINX_LIMITS'
# Кладётся в /etc/nginx/conf.d/shtil-limits.conf (контекст http). Ограничение частоты запросов к API Штиль.
limit_req_zone $binary_remote_addr zone=shtil_api:10m rate=10r/s;
NGINX_LIMITS
cat > /etc/nginx/sites-available/$DOMAIN <<'NGINX_SITE'
# Кладётся в /etc/nginx/sites-available/shtil.tech и включается ссылкой в sites-enabled.
# HTTPS-часть добавит certbot (`certbot --nginx -d shtil.tech`), он же настроит редирект с 80 на 443.
server {
    listen 80;
    listen [::]:80;
    server_name shtil.tech;

    # Наружу отдаётся только API v1. Всё остальное — 404.
    location /v1/ {
        limit_req zone=shtil_api burst=30 nodelay;
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 20s;
        proxy_connect_timeout 5s;
    }

    location / {
        return 404;
    }

    access_log /var/log/nginx/shtil.tech.access.log;
    error_log  /var/log/nginx/shtil.tech.error.log warn;
}
NGINX_SITE
ln -sf /etc/nginx/sites-available/$DOMAIN /etc/nginx/sites-enabled/$DOMAIN
if nginx -t 2>/tmp/shtil-nginx-test.log; then
  systemctl reload nginx && echo "nginx перезагружен"
else
  echo "ОШИБКА конфигурации nginx, откатываю:"; cat /tmp/shtil-nginx-test.log
  rm -f /etc/nginx/sites-enabled/$DOMAIN /etc/nginx/sites-available/$DOMAIN /etc/nginx/conf.d/shtil-limits.conf
  nginx -t && echo "прежняя конфигурация цела"; exit 1
fi

echo "== 4/4 сертификат"
if [ -n "${CERT_EMAIL:-}" ]; then MAIL=(-m "$CERT_EMAIL"); else MAIL=(--register-unsafely-without-email); fi
certbot --nginx -d "$DOMAIN" --redirect --non-interactive --agree-tos "${MAIL[@]}"
nginx -t && systemctl reload nginx
echo "Готово. Проверка: curl -I https://$DOMAIN/v1/health (после запуска контейнеров вернёт 200)"
