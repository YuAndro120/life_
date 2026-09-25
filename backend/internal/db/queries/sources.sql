-- name: ListActiveSources :many
SELECT id, kind, handle, url, title, topic_hint, last_fetched_at, consecutive_failures
FROM sources
WHERE active AND legal_status = 'ok' AND legal_checked_at IS NOT NULL
ORDER BY id;

-- name: UpsertSource :exec
-- Каталог из seeds/sources.yaml. Активность ограничена CHECK-ограничением (нужна дата проверки по реестрам).
INSERT INTO sources (kind, handle, url, title, topic_hint, legal_status, legal_checked_at, active)
VALUES (@kind, @handle, @url, @title, sqlc.narg('topic_hint'), @legal_status, sqlc.narg('legal_checked_at')::date, @active)
ON CONFLICT (kind, handle) DO UPDATE SET
    url = EXCLUDED.url,
    title = EXCLUDED.title,
    topic_hint = EXCLUDED.topic_hint,
    legal_status = EXCLUDED.legal_status,
    legal_checked_at = EXCLUDED.legal_checked_at,
    active = EXCLUDED.active;

-- name: RecordFetchSuccess :exec
UPDATE sources
SET last_fetched_at = @fetched_at::timestamptz, last_error = NULL, consecutive_failures = 0
WHERE id = @id;

-- name: RecordFetchFailure :exec
UPDATE sources
SET last_fetched_at = @fetched_at::timestamptz, last_error = @error::text, consecutive_failures = consecutive_failures + 1
WHERE id = @id;

-- name: InsertPost :execrows
-- Дубликаты по (source_id, external_id) молча пропускаются.
INSERT INTO posts (source_id, external_id, url, published_at, text, lang, is_ad, ad_suspected)
VALUES (@source_id, @external_id, @url, @published_at, @text, 'ru', @is_ad, @ad_suspected)
ON CONFLICT (source_id, external_id) DO NOTHING;
