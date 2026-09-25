-- +goose Up
-- Сюжет, который модель признала не новостью (технические страницы, таблицы, регламентные объявления), в ленту не попадает.
ALTER TABLE stories DROP CONSTRAINT stories_status_check;
ALTER TABLE stories ADD CONSTRAINT stories_status_check CHECK (status IN ('draft', 'published', 'skipped'));

-- +goose Down
UPDATE stories SET status = 'draft' WHERE status = 'skipped';
ALTER TABLE stories DROP CONSTRAINT stories_status_check;
ALTER TABLE stories ADD CONSTRAINT stories_status_check CHECK (status IN ('draft', 'published'));
