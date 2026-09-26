-- +goose Up
-- Региональные источники и законы. Код региона — двузначный номер субъекта РФ.
ALTER TABLE sources ADD COLUMN region_code text CHECK (region_code ~ '^[0-9]{2}$');
-- Регион сюжета по его источникам (заполняется при пересчёте); регион, названный моделью, остаётся в region_code.
ALTER TABLE stories ADD COLUMN source_region text;
-- kind: federal — разбор моделью с цитатами; regional_title — название и данные с портала опубликования, без пересказа.
ALTER TABLE law_changes ADD COLUMN kind text NOT NULL DEFAULT 'federal' CHECK (kind IN ('federal', 'regional_title'));

-- +goose Down
ALTER TABLE law_changes DROP COLUMN kind;
ALTER TABLE stories DROP COLUMN source_region;
ALTER TABLE sources DROP COLUMN region_code;
