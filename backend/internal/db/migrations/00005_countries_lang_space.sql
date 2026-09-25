-- +goose Up
-- Страна и язык источника: выбор «откуда новости» в приложении и язык пересказа. Тема «космос».
ALTER TABLE sources
    ADD COLUMN country text,                       -- ISO 3166-1 alpha-2 (RU, US, GB) или EU; NULL — не указана
    ADD COLUMN lang    text NOT NULL DEFAULT 'ru';

ALTER TABLE stories
    ADD COLUMN country text,
    ADD COLUMN lang    text NOT NULL DEFAULT 'ru';

ALTER TABLE stories DROP CONSTRAINT stories_topic_check;
ALTER TABLE stories ADD CONSTRAINT stories_topic_check CHECK (topic IN (
    'economy', 'finance', 'law', 'tech_ai', 'city', 'health', 'education', 'transport',
    'housing', 'science', 'space', 'culture', 'sport', 'showbiz', 'crypto',
    'politics', 'crime', 'incidents', 'disasters'));

-- Учёт токенов модели по дням: дневной лимит защищает от неожиданных расходов.
CREATE TABLE llm_usage (
    day               date PRIMARY KEY,
    prompt_tokens     bigint NOT NULL DEFAULT 0,
    completion_tokens bigint NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE llm_usage;
UPDATE stories SET topic = 'science' WHERE topic = 'space';
ALTER TABLE stories DROP CONSTRAINT stories_topic_check;
ALTER TABLE stories ADD CONSTRAINT stories_topic_check CHECK (topic IN (
    'economy', 'finance', 'law', 'tech_ai', 'city', 'health', 'education', 'transport',
    'housing', 'science', 'culture', 'sport', 'showbiz', 'crypto',
    'politics', 'crime', 'incidents', 'disasters'));
ALTER TABLE stories DROP COLUMN lang, DROP COLUMN country;
ALTER TABLE sources DROP COLUMN lang, DROP COLUMN country;
