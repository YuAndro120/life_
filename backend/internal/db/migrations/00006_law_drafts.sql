-- +goose Up
-- Черновики законов: строки law_changes с verified = false. В API попадают только verified = true (lawtool review).
ALTER TABLE law_changes
    ADD COLUMN eo_number     text UNIQUE,                        -- номер опубликования на publication.pravo.gov.ru
    ADD COLUMN source_url    text,                               -- страница с полным текстом (kremlin.ru)
    ADD COLUMN quotes        jsonb   NOT NULL DEFAULT '{}'::jsonb, -- дословные цитаты из текста для проверки человеком
    ADD COLUMN rejected      boolean NOT NULL DEFAULT false,     -- отклонён (не касается граждан или ошибка извлечения)
    ADD COLUMN reject_reason text;

ALTER TABLE law_changes
    ADD CONSTRAINT law_not_both CHECK (NOT (verified AND rejected));

-- +goose Down
ALTER TABLE law_changes DROP CONSTRAINT law_not_both;
ALTER TABLE law_changes
    DROP COLUMN reject_reason, DROP COLUMN rejected, DROP COLUMN quotes, DROP COLUMN source_url, DROP COLUMN eo_number;
