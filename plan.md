# «Суть» — план MVP для iOS

Рабочее название: **Суть**. Нативное iOS-приложение: спокойный новостной дайджест и персональный блок изменений в законах.
Этот документ — единый источник правды для разработки в Claude Code. Работаем по фазам сверху вниз, каждая фаза заканчивается проверяемым результатом.

Макет дизайна (все экраны, темы, онбординг): https://claude.ai/artifact/7Kb1uD4cYoFEbiXjGm5nd1
Копия макетов лежит в `design/` (файлы `.dc.html`: обычный HTML с inline-стилями, шаблоны `{{...}}` и логика в `<script type="text/x-dc">`).

---

## 1. Продукт в одном абзаце

Человек один раз указывает жизненные ситуации (работа, жильё, транспорт, возраст, регион) и что ему не показывать (стоп-темы, тяжёлые новости, слухи). Дважды в день он получает **выпуск**, который можно дочитать до конца. Сверху блок **«Касается тебя»**: подписанные изменения в законах, подобранные под его профиль, с датой вступления в силу и тем, что нужно сделать. Ниже 3–7 сюжетов за сутки: повторы из разных каналов склеены, реклама убрана, заголовки нейтральные. Тяжёлые темы свёрнуты в одну сводку.

## 2. Ключевые решения (не пересматривать без причины)

| Решение | Суть |
|---|---|
| Платформа | Только iOS, нативно. Swift 6 + SwiftUI, iOS 17+ |
| Распространение | Без App Store. Этап 1: установка через Xcode с бесплатным Apple ID (приложение живёт 7 дней, push недоступен). Этап 2: платный аккаунт ($99) → TestFlight/Ad Hoc для друзей |
| Приватность | **Профиль и фильтры хранятся только на устройстве.** Сервер отдаёт одинаковую ленту всем, фильтрация и подбор законов происходят на телефоне. Без регистрации и аккаунтов |
| MVP-ядро | Юридический блок + спокойный дайджест из общего каталога источников |
| Источники | Общий каталог на сервере: только юридически чистые источники (официальные сайты, RSS СМИ без статусов, публичные TG-каналы без статусов). Нежелательные организации и иноагенты в каталог **не входят**. Личные источники, которые телефон забирает сам, — после MVP |
| Уведомления в MVP | Локальные уведомления по расписанию + `BGAppRefreshTask`. APNs после покупки платного аккаунта |
| Бэкенд | Go, PostgreSQL + pgvector, очередь River, эмбеддинги bge-m3, LLM через API (GigaChat/YandexGPT) или локально |
| iOS-зависимости | **Ноль сторонних библиотек.** Только системные фреймворки |
| Контент | Пересказ + ссылка на первоисточник. Без перепубликации полных текстов СМИ. Официальные документы можно цитировать (п. 6 ст. 1259 ГК) |

## 3. Что входит в MVP и что нет

**Входит:**
- Онбординг: приветствие → профиль → «что не показывать» → тема и расписание → анимация «собираем выпуск».
- Экран «Сегодня» (выпуск): блок «Касается тебя», сюжеты за сутки, свёрнутые тяжёлые темы, «Конец выпуска».
- Карточка изменения закона: статус, отсчёт «Д–N», что/кого/почему тебе/что сделать, первоисточники, дисклеймер.
- Календарь изменений: «Про меня / Все», напоминания.
- Фильтры: спокойный режим, тип информации, тяжёлые темы, стоп-темы, реклама, расписание.
- Три темы: Бумага (по умолчанию), Шалфей, Сумерки + автоматические Сумерки после 19:00.
- Локальные уведомления «Выпуск готов» в 08:00 и 19:00 и напоминания о законах за 7 дней.
- Бэкенд: сбор 20–40 источников, склейка, классификация, саммари; юридический конвейер с ручной проверкой.

**Не входит (после MVP):**
- Личные источники (RSS, добавляемый пользователем).
- APNs и мгновенные уведомления.
- Виджет на экран «Домой».
- Экран отдельного сюжета с раскрытием всех постов.
- Android, веб.

## 4. Архитектура

```
┌─────────────── сервер (один VPS в MVP) ───────────────┐
│                                                        │
│  collector ──► posts ──► worker ──► stories            │
│  (RSS, t.me/s,     (эмбеддинг, склейка,               │
│   сайты ведомств)   классификация, саммари)            │
│                                                        │
│  legal-collector ──► law_drafts ──► ручная проверка    │
│  (pravo.gov.ru,       (LLM извлекает     ──► law_changes│
│   sozd.duma.gov.ru)    структуру)                      │
│                                                        │
│  api (Go) ──► GET /v1/feed, /v1/laws, /v1/catalog      │
│  Caddy (TLS 1.3, rate limit) — единственный вход       │
└────────────────────────────────────────────────────────┘
                         │ HTTPS, одинаковый ответ для всех
                         ▼
┌─────────────────── iPhone ────────────────────────────┐
│  Профиль + фильтры (SwiftData, только локально)        │
│  FilterEngine: stories + laws × профиль → выпуск       │
│  Локальные уведомления, BGAppRefresh, тема             │
└────────────────────────────────────────────────────────┘
```

Сервер не хранит персональных данных пользователей. В MVP нет регистрации, поэтому один VPS (в том числе вне РФ, чтобы стабильно читать источники) допустим. Если появятся аккаунты или push-токены, пересмотреть локализацию данных по 152-ФЗ.

## 5. Структура репозитория

```
sut/
├── CLAUDE.md                 # правила для Claude Code
├── plan.md                   # этот файл
├── design/                   # копии макетов .dc.html
├── ios/
│   └── Life/
│       ├── Life.xcodeproj
│       ├── App/              # LifeApp.swift, корневая навигация, DI
│       ├── Core/
│       │   ├── Theme/        # токены трёх тем, шрифты, модификаторы
│       │   ├── Networking/   # APIClient (URLSession), модели DTO, pinning
│       │   ├── Persistence/  # SwiftData: Profile, FilterSettings, CachedFeed, Reminders
│       │   ├── Filtering/    # FilterEngine, AudienceMatcher, EditionBuilder
│       │   ├── Notifications/# локальные уведомления, BGTask
│       │   └── Security/     # ключ устройства в Secure Enclave, подпись запросов
│       ├── Features/
│       │   ├── Onboarding/   # Welcome, Profile, Calm, Theme, Building
│       │   ├── Today/
│       │   ├── LawDetail/
│       │   ├── Calendar/
│       │   └── Filters/
│       ├── Resources/Fonts/  # Onest, IBM Plex Mono, Spectral, Golos Text (OFL)
│       └── Tests/
├── backend/
│   ├── go.mod
│   ├── cmd/
│   │   ├── api/
│   │   ├── collector/
│   │   ├── worker/
│   │   └── lawtool/          # CLI для ручной проверки законов
│   ├── internal/
│   │   ├── config/
│   │   ├── db/               # миграции (goose/atlas), запросы (sqlc)
│   │   ├── sources/          # rss, tgpreview, gov
│   │   ├── pipeline/         # embed, cluster, classify, summarize
│   │   ├── legal/
│   │   ├── llm/              # интерфейс Summarizer/Classifier: gigachat, yandexgpt, ollama
│   │   ├── embed/            # интерфейс Embedder: ollama (dev), tei (prod)
│   │   └── httpapi/
│   └── seeds/                # sources.yaml, тестовые данные
└── infra/
    ├── docker-compose.dev.yml
    ├── docker-compose.prod.yml
    ├── Caddyfile
    └── README.md
```

## 6. Модель данных (PostgreSQL)

```sql
sources(id, kind['rss','tg','gov','site'], handle, url, title, topic_hint,
        legal_status['ok','excluded'], active bool, created_at)

posts(id, source_id, external_id, url, published_at, text, lang,
      is_ad bool, embedding vector(1024), story_id null, created_at,
      UNIQUE(source_id, external_id))

stories(id, first_seen_at, updated_at, topic, info_type, heaviness,
        title_neutral, summary, meaning,          -- «Значит: …»
        post_count, source_count, centroid vector(1024),
        region_code null, status['draft','published'])

story_posts(story_id, post_id)

law_changes(id, title, what_changed, who_affected, actions jsonb,
            audience_tags text[], region_code null,
            status['introduced','passed','signed','in_force'],
            introduced_at, passed_at, signed_at, effective_at,
            official_url, bill_url, act_number,
            verified bool default false, verified_at, created_at)
```

Индексы: `posts(published_at)`, `stories(updated_at)`, `law_changes(effective_at)`, HNSW по `posts.embedding` и `stories.centroid`.

## 7. Таксономии (общие для сервера и iOS)

**Темы (topic):** `economy, finance, law, tech_ai, city, health, education, transport, housing, science, culture, sport, showbiz, crypto, politics, crime, incidents, disasters`.

**Тип информации (info_type):** `fact, official, opinion, forecast, rumor`.

**Накал (heaviness):** `neutral, tense, heavy`. В режиме «Сворачивать» `heavy` уходит в сводку.

**Теги аудитории законов (audience_tags)** соответствуют ответам онбординга:
```
gender:male, gender:female
age:u20, age:20_25, age:26_35, age:36_50, age:50p
work:employee, work:ip, work:ip_usn, work:selfemployed, work:student
housing:renter, housing:owner, housing:mortgage
transport:driver
military:registered        # выводится на устройстве: мужчина 18–30
all                        # касается всех
region:<код>               # региональные акты
```
Правило подбора на устройстве: закон попадает в «Касается тебя», если есть пересечение `audience_tags` с тегами профиля или тег `all` и регион совпадает (или закон федеральный).

## 8. API (v1)

Ответы одинаковые для всех, кэшируются (ETag, `Cache-Control: max-age=300`), gzip.

`GET /v1/feed?since=2026-09-24T19:00:00Z`
```json
{
  "generated_at": "2026-09-25T02:55:00Z",
  "stories": [{
    "id": "st_01H...",
    "topic": "economy",
    "info_type": "fact",
    "heaviness": "neutral",
    "title": "Банк России объявил решение по ключевой ставке",
    "meaning": "Условия по вкладам и кредитам в ближайшие недели заметно не изменятся.",
    "summary": "…",
    "post_count": 9,
    "source_count": 5,
    "sources": [{"title": "…", "url": "https://…"}],
    "region_code": null,
    "updated_at": "2026-09-25T01:10:00Z"
  }],
  "stats": {"posts_total": 38, "ads_hidden": 7}
}
```

`GET /v1/laws?from=2026-09-01&to=2027-06-30` — только `verified = true`:
```json
{
  "laws": [{
    "id": "lw_…",
    "title": "Меняется срок уведомлений об авансовых платежах",
    "what_changed": "…",
    "who_affected": "…",
    "actions": ["Перенести даты в календаре платежей"],
    "audience_tags": ["work:ip_usn"],
    "region_code": null,
    "status": "signed",
    "dates": {"introduced": "…", "passed": "…", "signed": "…", "effective": "2026-10-01"},
    "official_url": "https://publication.pravo.gov.ru/…",
    "bill_url": "https://sozd.duma.gov.ru/…",
    "act_number": "…",
    "verified_at": "…"
  }]
}
```

`GET /v1/catalog/sources` — список источников каталога (для экрана источников после MVP).

`GET /v1/health` — для мониторинга.

Подпись запросов (фаза 7): заголовки `X-Device-Key` (id публичного ключа), `X-Timestamp`, `X-Signature` (ECDSA P-256 по `method\npath\ntimestamp`). Ключ регистрируется через `POST /v1/devices` с публичным ключом. Используется только для rate limit и защиты от ботов, не связан с личностью.

## 9. Конвейер обработки

1. **Сбор** (collector, каждые 10 мин): RSS (`encoding/xml`), публичное веб-превью `https://t.me/s/<channel>` (парсинг HTML, `external_id` = id поста), сайты ведомств. Дедупликация по `(source_id, external_id)`.
2. **Реклама:** регулярка по маркировке (`Реклама`, `erid`, `ИНН`), плюс LLM для спорных случаев. `is_ad = true` → в сюжеты не попадает, считается в `stats.ads_hidden`.
3. **Эмбеддинг:** bge-m3 (1024), текст обрезать до ~512 токенов.
4. **Склейка:** для нового поста найти ближайший `stories.centroid` за последние 36 ч. Если косинусная близость ≥ 0.82 (подобрать на реальных данных), пост прикрепляется к сюжету и центроид пересчитывается. Иначе создаётся новый сюжет.
5. **Классификация сюжета** (LLM, строгий JSON): `topic, info_type, heaviness, region_code`.
6. **Саммари** (LLM, с дебаунсом 10 мин после последнего поста): `title_neutral` (без эмоций, кликбейта и восклицаний), `summary` (2–3 предложения), `meaning` («что это значит для обычного человека» или пусто).
7. **Публикация:** сюжет с ≥ 2 источниками или из официального источника → `published`.

Промпты хранить в `internal/llm/prompts/*.txt`, ответы проверять по JSON-схеме, при невалидном ответе повторять запрос.

## 10. Юридический конвейер

1. **Источники:** официальный портал опубликования (publication.pravo.gov.ru) и СОЗД Госдумы (sozd.duma.gov.ru). **Проверить** наличие API или RSS у каждого; у СОЗД есть API с ключом (api.duma.gov.ru). Если API нет, аккуратно парсить с кэшированием.
2. **Отбор:** в MVP только федеральные законы и постановления правительства, затрагивающие граждан и ИП. Фильтр по ключевым словам и LLM-оценке «касается обычных людей: да/нет».
3. **Извлечение:** LLM **только извлекает** структуру из официального текста (что изменилось, кого касается, теги аудитории, даты, действия). Не пересказывает «своими словами» и не придумывает. Ответ в JSON.
4. **Ручная проверка:** `lawtool review` (CLI) показывает черновик рядом с цитатами из текста, можно поправить поля и поставить `verified = true`. Без проверки закон в API не попадает.
5. **Статусы и даты** обновляются при каждом прогоне; для подписанных законов важна `effective_at`.

## 11. iOS: как устроено

**Навигация:** онбординг (если профиль пуст) → `TabView` с кастомным таб-баром: Сегодня / Календарь / Фильтры / Профиль. Карточка закона открывается через `NavigationStack`.

**Хранилище (SwiftData):**
- `Profile`: gender, ageBracket, work[], housing[], transport, regionCode.
- `FilterSettings`: calmMode, infoTypes (Set), heavyMode (hide/fold/show), maxHeavy, stopTopics (Set), hideAds, schedule (both/am/pm), morningTime, eveningTime, theme, autoDusk.
- `CachedStory`, `CachedLaw`: последний ответ API.
- `Reminder`: lawId, fireDate.
- `EditionCounter`: номер выпуска.

**FilterEngine** (чистые функции, покрыть тестами):
- `audienceTags(profile) -> Set<String>`, включая вывод `military:registered`.
- `relevantLaws(laws, tags, region) -> [Law]`, сортировка по `effective_at`.
- `edition(stories, settings, window) -> Edition`: убрать стоп-темы и отключённые типы информации, тяжёлые обработать по режиму, отсортировать (официальное > факт; больше источников — выше). Считать статистику для шапки (про тебя / сюжеты / минуты чтения ≈ символы / 1200 в минуту).

**Выпуск:** окно = (прошлый выпуск, текущий]. Номер выпуска — локальный счётчик.

**Уведомления:**
- `UNUserNotificationCenter`: «Выпуск готов» в выбранные часы (`UNCalendarNotificationTrigger`, repeats).
- Напоминания о законах за 7 дней до `effective_at`. Если осталось меньше 7 дней, напомнить накануне.
- `BGAppRefreshTask` (`com.sut.refresh`): подтянуть `/v1/feed` и `/v1/laws` заранее. Info.plist: `BGTaskSchedulerPermittedIdentifiers`, `UIBackgroundModes: fetch`.

**Безопасность:**
- ATS по умолчанию (только HTTPS), certificate/public-key pinning в `URLSessionDelegate` (pin на ключ промежуточного/листового сертификата с запасным пином).
- Ключ устройства: `SecureEnclave.P256.Signing.PrivateKey`, `dataRepresentation` в Keychain (`kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`).
- Data Protection для файлов SwiftData (`.complete` или `.completeUntilFirstUserAuthentication` из-за фоновых задач).

## 12. Дизайн-система (из макета)

**Шрифты** (все OFL, положить в `Resources/Fonts`, прописать в `UIAppFonts`):
- Onest 400/500/600/700 — основной (Бумага, Сумерки).
- IBM Plex Mono 400/500 — метаданные: заглавные, 11pt, трекинг +0.04em.
- Spectral 500/600 + Italic 400 — заголовки темы Шалфей.
- Golos Text 400/500/600 — текст темы Шалфей.

**Токены цветов:**

| Токен | Бумага | Шалфей | Сумерки |
|---|---|---|---|
| bg | #F2F1EC | #EDF0EA | #1C1D22 |
| ink | #22211E | #1E2621 | #ECEAE4 |
| body | #3D3C38 | #36403A | #CFCDC7 |
| muted | #66655F | #5B665E | #A3A19B |
| line | #D8D6CE | #D2DAD1 | #34363E |
| accent («про тебя») | #2432D0 | #2F6B4F | #A9B3FF |
| accentDeep | #1D2A9E | #245740 | #C6CCFF |
| plate (подложка «Касается тебя») | #E4E7F6 | #DAE7DD | #262937 |
| card | #FBFBFE | #FAFCF9 | #30344A |
| segment bg | #E6E4DD | #DFE6DE | #2A2C34 |
| chip border / bg | #C9C7BF / #FAF9F6 | — / #FAFCF9 | #34363E / #262937 |
| primary button bg / text | ink / bg | #2F6B4F / #FAFCF9 | #A9B3FF / #1C1D22 |

**Правило цвета:** акцент используется только для того, что касается пользователя лично. Красного в интерфейсе нет.

**Формы:** чипы и сегменты — капсулы (radius 999); карточки 18 (Шалфей 22); подложка 26 (Шалфей 30); главная кнопка 18, высота 58; минимальная зона нажатия 44×44.

**Типографика (Бумага/Сумерки):** логотип «Суть.» 76 bold, трекинг −0.065em; H1 44–48 bold, −0.055em; цифры-итоги 44 semibold tabular; заголовок закона 23–24 semibold, −0.03em; заголовок сюжета 21 semibold; текст 15–16; метаданные — моно 11 заглавными.
**Шалфей:** H1 Spectral 40 medium, вторая строка italic muted; вместо моно-меток обычный регистр 13pt; отсчёт словами («через 6 дней»).

**Экраны (ориентир — макет):** Welcome, Profile, Calm, Theme (живая перекраска при выборе), Building (анимация 38 → 19 → 3), Today, LawDetail, Calendar, Filters. Таб-бар: Бумага/Сумерки — текстовый с линией сверху у активного; Шалфей — плавающая капсула.

**Анимация «Собираем выпуск»:** около 5 с. Полоски-посты появляются → отфильтрованные гаснут → оставшиеся съезжаются в 3 стопки → стопки превращаются в карточки. Чек-лист из 4 шагов, в конце кнопка «Открыть выпуск». В приложении цифры настоящие: из `stats` и результата FilterEngine. SwiftUI: `matchedGeometryEffect` или явные позиции с `withAnimation(.spring)`, фазы через `PhaseAnimator` или `Task` с задержками.

## 13. Окружение на Mac M4

```bash
# 1. Xcode из App Store (последняя стабильная), затем:
xcode-select --install
sudo xcodebuild -license accept

# 2. Homebrew и инструменты
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install go sqlc goose caddy jq
brew install --cask orbstack        # Docker на Apple Silicon (легче Docker Desktop)
brew install ollama                 # локальные эмбеддинги и LLM на Metal

# 3. Модели для разработки
ollama pull bge-m3                  # эмбеддинги 1024, хороший русский
ollama pull qwen2.5:7b-instruct     # локальный LLM для классификации и саммари (или API)

# 4. База
docker compose -f infra/docker-compose.dev.yml up -d   # postgres 16 + pgvector
```

`.env.dev`: `DATABASE_URL`, `EMBEDDER=ollama`, `LLM_PROVIDER=ollama|gigachat|yandexgpt`, ключи API, `CACHE_DIR`.

**Установка на iPhone без App Store:**
1. Xcode → Settings → Accounts → добавить Apple ID (Personal Team).
2. Target → Signing & Capabilities → Team = Personal Team, уникальный Bundle ID (например `ru.andronov.life`).
3. На iPhone: Настройки → Конфиденциальность и безопасность → Режим разработчика → вкл., перезагрузка.
4. Подключить iPhone кабелем (потом можно по Wi-Fi: Window → Devices and Simulators → Connect via network), выбрать устройство и нажать Run.
5. На iPhone: Настройки → Основные → VPN и управление устройством → доверять разработчику.
6. Через 7 дней приложение перестанет запускаться — снова Run из Xcode. Бесплатная подпись не даёт Push Notifications; локальные уведомления и фоновое обновление работают.

## 14. Фазы и чек-листы

### Фаза 0 — Каркас (0.5 дня)
- [x] Монорепо по структуре из раздела 5, git, `.gitignore`, `CLAUDE.md`.
- [x] Xcode-проект SwiftUI, iOS 17, Swift 6 strict concurrency (собирается и тестируется на симуляторе; запуск на iPhone проверить вручную).
- [x] Go-модуль, `cmd/api` с `/v1/health`, docker-compose с Postgres + pgvector (compose написан, `docker compose up` не проверен).

### Фаза 1 — iOS на моках (3–5 дней) — результат: приложение на телефоне выглядит как макет
- [x] Theme: токены трёх тем, шрифты, `@Environment(\.theme)`, авто-Сумерки после 19:00 (до 06:00).
- [x] Модели `Story`, `Law`, `Edition` и JSON-фикстуры в формате API (раздел 8).
- [x] SwiftData-модели профиля и фильтров.
- [x] FilterEngine + unit-тесты (подбор законов по тегам, стоп-темы, режимы тяжёлых тем).
- [ ] Экраны: Today, LawDetail, Calendar, Filters, кастомный таб-бар.
- [ ] Онбординг: Welcome → Profile → Calm → Theme (живая перекраска) → Building (анимация).
- [ ] Локальные уведомления: выпуск по расписанию, напоминания о законах.

### Фаза 2 — Бэкенд-скелет (2–3 дня) — результат: приложение берёт данные с локального сервера
- [ ] Миграции по разделу 6, sqlc-запросы.
- [ ] `GET /v1/feed`, `GET /v1/laws` из сид-данных, ETag и gzip.
- [ ] iOS APIClient, кэш в SwiftData, pull-to-refresh, офлайн-режим.

### Фаза 3 — Сбор источников (2–3 дня)
- [ ] `seeds/sources.yaml`: 20–40 источников по темам (ИИ, экономика, право, город, технологии). Среди первых — @NeuralProfit, @simply_formula, если проходят проверку статуса.
- [ ] Коллекторы RSS и `t.me/s`, расписание, ретраи, уважительные интервалы, User-Agent.
- [ ] Фильтр рекламы по маркировке.

### Фаза 4 — Обработка (4–6 дней) — результат: реальные склеенные сюжеты
- [ ] Интерфейсы `Embedder` (Ollama/TEI) и `LLM` (Ollama/GigaChat/YandexGPT).
- [ ] Склейка по центроидам; подобрать порог на реальных данных, сделать отчёт `worker debug-clusters`.
- [ ] Классификация и саммари со строгим JSON, валидацией и дебаунсом.
- [ ] Метрики: постов за сутки, сюжетов, доля рекламы, средний размер кластера.

### Фаза 5 — Юридический блок (4–6 дней)
- [ ] Исследовать API и форматы publication.pravo.gov.ru и sozd.duma.gov.ru, выбрать способ сбора.
- [ ] Отбор «касается граждан/ИП», извлечение структуры и тегов аудитории.
- [ ] `lawtool review` для ручной проверки; публикуются только проверенные.
- [ ] Первые 20–30 проверенных изменений со вступлением в силу в ближайшие 6 месяцев.

### Фаза 6 — Сборка выпуска на устройстве (2 дня)
- [ ] `BGAppRefreshTask` заранее подтягивает данные; выпуск собирается к времени уведомления.
- [ ] Шапка выпуска: реальные «про тебя / сюжеты / минуты».
- [ ] Анимация Building на реальных цифрах первого выпуска.

### Фаза 7 — Безопасность и деплой (2–3 дня)
- [ ] VPS, Docker-образы distroless, non-root, read-only FS, внутренняя сеть; наружу только Caddy:443.
- [ ] SSH только по ключам через WireGuard, файрвол, автообновления.
- [ ] Ключ устройства в Secure Enclave, подпись запросов, rate limit.
- [ ] Certificate pinning в приложении.
- [ ] Шифрованные бэкапы (restic), `govulncheck` и `gosec` в CI.

### Фаза 8 — Догфудинг (2 недели)
- [ ] Пользуюсь сам каждый день, веду заметки о шуме и пропусках.
- [ ] Подстраиваю пороги склейки, промпты, каталог источников.
- [ ] Платный Apple Developer → TestFlight для 10–20 знакомых.

## 15. Definition of Done для MVP

- Приложение установлено на iPhone, онбординг проходится за минуту.
- Дважды в день приходит уведомление, выпуск открывается офлайн и дочитывается до «Конца выпуска».
- В выпуске нет рекламы и стоп-тем, повторы склеены (одна новость = одна карточка).
- Блок «Касается тебя» показывает только проверенные законы, релевантные профилю, со ссылкой на официальный текст.
- Сервер не хранит ничего о пользователях, кроме технических логов без персональных данных.
- Три темы работают, Сумерки включаются вечером автоматически.

## 16. Юридические правила контента

- Каталог источников проверяется по реестрам нежелательных организаций и иноагентов **перед** добавлением и периодически после. Такие источники в каталог не входят.
- Любой сюжет содержит ссылки на первоисточники; сервис не делает собственных выводов «кто прав».
- Карточка закона сопровождается дисклеймером «Пересказ, не юридическая консультация» и датой сверки.
- Политика конфиденциальности: сервер не собирает персональных данных, профиль хранится на устройстве.

## 17. Открытые вопросы

- Финальное название: **пока «Life»** (решение 2026-09-25). В макетах и тексте интерфейса остаётся «Суть.» до отдельного решения о ребрендинге; Bundle ID — `ru.andronov.life`, целевое имя Xcode-проекта `Life`.
- Пункт «Срочные уведомления → Ключевые слова» в Фильтрах: показывается **неактивным** (APNs после MVP), решение 2026-09-25.
- Список регионов и региональных источников для MVP.
- Провайдер LLM для прода: GigaChat, YandexGPT или своя модель на GPU-сервере.
- Хостинг: один VPS или разделение на сборщик и API.
