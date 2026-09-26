-- +goose Up
-- Пост, почти дословно повторяющий более ранний пост другого источника в том же сюжете (пересказ, а не независимый источник).
ALTER TABLE posts ADD COLUMN derived_from bigint REFERENCES posts (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE posts DROP COLUMN derived_from;
