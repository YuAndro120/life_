-- name: ListPublishedStories :many
-- Опубликованные сюжеты за окно. Источники: по одной ссылке на источник (предпочтительно оригинал, а не пересказ; первоисточники идут первыми).
SELECT
    s.id,
    s.topic,
    s.info_type,
    s.heaviness,
    s.title_neutral,
    s.summary,
    s.meaning,
    s.post_count,
    s.source_count,
    COALESCE(s.source_region, s.region_code) AS region_code,
    s.country,
    s.lang,
    s.updated_at,
    COALESCE((
        SELECT jsonb_agg(jsonb_build_object('title', src.title, 'url', fp.url, 'reprint', fp.reprint) ORDER BY fp.reprint, fp.published_at, src.title)
        FROM (
            SELECT DISTINCT ON (p.source_id) p.source_id, p.url, p.published_at, (p.derived_from IS NOT NULL) AS reprint
            FROM posts p
            WHERE p.story_id = s.id
            ORDER BY p.source_id, (p.derived_from IS NOT NULL), p.published_at
        ) fp
        JOIN sources src ON src.id = fp.source_id
    ), '[]'::jsonb)::jsonb AS sources
FROM stories s
WHERE s.status = 'published' AND s.title_neutral IS NOT NULL AND s.summary IS NOT NULL
  AND s.topic IS NOT NULL AND s.info_type IS NOT NULL AND s.heaviness IS NOT NULL
  AND s.updated_at > @since::timestamptz
ORDER BY s.updated_at DESC, s.id DESC;

-- name: FeedStats :one
SELECT
    count(*)::int AS posts_total,
    (count(*) FILTER (WHERE is_ad))::int AS ads_hidden
FROM posts
WHERE published_at > @since::timestamptz;
