-- +goose Up
ALTER TABLE sources
    ADD COLUMN legal_checked_at     date,
    ADD COLUMN last_fetched_at      timestamptz,
    ADD COLUMN last_error           text,
    ADD COLUMN consecutive_failures integer NOT NULL DEFAULT 0;

ALTER TABLE posts
    -- Пост похож на рекламу, но маркировка неполная: решает LLM (фаза 4).
    ADD COLUMN ad_suspected boolean NOT NULL DEFAULT false;

-- Жёсткое правило проекта: источник без даты проверки по реестрам нежелательных организаций
-- и иноагентов не может быть активным. Держим его в БД, чтобы не обойти ни кодом, ни вручную.
UPDATE sources SET active = false WHERE legal_checked_at IS NULL;
ALTER TABLE sources
    ADD CONSTRAINT sources_active_requires_legal_check
    CHECK (NOT active OR (legal_status = 'ok' AND legal_checked_at IS NOT NULL));

-- +goose Down
ALTER TABLE sources DROP CONSTRAINT sources_active_requires_legal_check;
ALTER TABLE posts DROP COLUMN ad_suspected;
ALTER TABLE sources
    DROP COLUMN legal_checked_at,
    DROP COLUMN last_fetched_at,
    DROP COLUMN last_error,
    DROP COLUMN consecutive_failures;
