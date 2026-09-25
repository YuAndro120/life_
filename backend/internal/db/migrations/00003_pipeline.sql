-- +goose Up
-- Сюжет создаётся при склейке, а классификация и пересказ приходят позже (LLM), поэтому эти поля пустеют до обработки.
ALTER TABLE stories
    ALTER COLUMN topic DROP NOT NULL,
    ALTER COLUMN info_type DROP NOT NULL,
    ALTER COLUMN heaviness DROP NOT NULL,
    ALTER COLUMN title_neutral DROP NOT NULL,
    ALTER COLUMN summary DROP NOT NULL,
    ADD COLUMN last_post_at        timestamptz,
    -- Сколько постов было в сюжете при последнем пересказе: если стало больше, пересказ обновляется.
    ADD COLUMN digest_post_count   integer NOT NULL DEFAULT 0,
    ADD COLUMN digest_attempts     integer NOT NULL DEFAULT 0,
    ADD COLUMN digest_error        text,
    ADD COLUMN digest_at           timestamptz,
    ADD COLUMN has_official_source boolean NOT NULL DEFAULT false;

CREATE INDEX stories_status_idx ON stories (status, updated_at);
CREATE INDEX posts_unclustered_idx ON posts (published_at) WHERE story_id IS NULL AND NOT is_ad;

-- +goose Down
DROP INDEX posts_unclustered_idx;
DROP INDEX stories_status_idx;
ALTER TABLE stories
    DROP COLUMN has_official_source,
    DROP COLUMN digest_at,
    DROP COLUMN digest_error,
    DROP COLUMN digest_attempts,
    DROP COLUMN digest_post_count,
    DROP COLUMN last_post_at;
