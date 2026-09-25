-- name: ListUnclusteredPosts :many
-- Свежие нерекламные посты, ещё не отнесённые к сюжету.
SELECT p.id, p.source_id, p.url, p.published_at, p.text, s.title AS source_title, s.kind AS source_kind
FROM posts p
JOIN sources s ON s.id = p.source_id
WHERE p.story_id IS NULL AND NOT p.is_ad AND p.published_at > @since::timestamptz
ORDER BY p.published_at, p.id;

-- name: ListOpenStoryPosts :many
-- Посты открытых сюжетов за окно склейки (для расчёта центров сюжетов).
SELECT p.story_id, p.id, p.source_id, p.text, p.published_at
FROM posts p
JOIN stories st ON st.id = p.story_id
WHERE st.status IN ('draft', 'published') AND st.first_seen_at > @since::timestamptz
ORDER BY p.story_id, p.published_at;

-- name: CreateStory :one
INSERT INTO stories (first_seen_at, updated_at, last_post_at, post_count, source_count, has_official_source, status)
VALUES (@first_seen_at, @updated_at, @updated_at, 0, 0, false, 'draft')
RETURNING id;

-- name: AttachPost :exec
UPDATE posts SET story_id = @story_id WHERE id = @post_id;

-- name: AttachStoryPost :exec
INSERT INTO story_posts (story_id, post_id) VALUES (@story_id, @post_id) ON CONFLICT DO NOTHING;

-- name: RecountStory :exec
-- Пересчёт счётчиков сюжета по его постам. Время обновления сдвигается только вперёд.
UPDATE stories s SET
    post_count = c.n,
    source_count = c.sources,
    has_official_source = c.official,
    last_post_at = c.last_at,
    updated_at = GREATEST(s.updated_at, c.last_at)
FROM (
    SELECT count(*)::int AS n, count(DISTINCT p.source_id)::int AS sources,
           bool_or(src.kind = 'gov') AS official, max(p.published_at) AS last_at
    FROM posts p JOIN sources src ON src.id = p.source_id
    WHERE p.story_id = @story_id
) c
WHERE s.id = @story_id;

-- name: ListStoriesForDigest :many
-- Сюжеты, которым пора делать пересказ: новых постов нет уже @quiet_for (дебаунс), попыток немного.
SELECT id, post_count, source_count, has_official_source, digest_attempts
FROM stories
WHERE status IN ('draft', 'published')
  AND last_post_at <= @quiet_before::timestamptz
  AND (title_neutral IS NULL OR post_count > digest_post_count)
  AND digest_attempts < @max_attempts::int
ORDER BY last_post_at
LIMIT @batch::int;

-- name: ListStoryPostsForDigest :many
SELECT p.id, p.url, p.published_at, p.text, src.title AS source_title, src.kind AS source_kind
FROM posts p JOIN sources src ON src.id = p.source_id
WHERE p.story_id = @story_id
ORDER BY p.published_at
LIMIT @lim::int;

-- name: SaveStoryDigest :exec
UPDATE stories SET
    topic = @topic, info_type = @info_type, heaviness = @heaviness,
    title_neutral = @title, summary = @summary, meaning = sqlc.narg('meaning'),
    region_code = sqlc.narg('region_code'),
    digest_post_count = post_count, digest_attempts = 0, digest_error = NULL, digest_at = @at::timestamptz,
    status = CASE WHEN @newsworthy::boolean THEN status ELSE 'skipped' END
WHERE id = @id;

-- name: RecordDigestFailure :exec
UPDATE stories SET digest_attempts = digest_attempts + 1, digest_error = @error::text WHERE id = @id;

-- name: PublishReadyStories :execrows
-- Правило публикации (раздел 9): не меньше двух разных источников или официальный источник.
UPDATE stories SET status = 'published'
WHERE status = 'draft' AND title_neutral IS NOT NULL AND (source_count >= 2 OR has_official_source);

-- name: UnpublishWeakStories :execrows
-- Если сюжет перестал удовлетворять правилу (например, после ручной правки), возвращаем в черновики.
UPDATE stories SET status = 'draft'
WHERE status = 'published' AND NOT (source_count >= 2 OR has_official_source);

-- name: ListRecentPosts :many
-- Для отладки склейки: все нерекламные посты за окно вместе с текущим сюжетом.
SELECT p.id, p.source_id, p.text, p.published_at, p.story_id, src.title AS source_title, src.kind AS source_kind
FROM posts p JOIN sources src ON src.id = p.source_id
WHERE p.published_at > @since::timestamptz AND NOT p.is_ad
ORDER BY p.published_at, p.id;
