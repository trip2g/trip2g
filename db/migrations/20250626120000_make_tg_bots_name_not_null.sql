-- migrate:up
-- Make name field not null in tg_bots table
UPDATE tg_bots SET name = 'Unknown' WHERE name IS NULL;
ALTER TABLE tg_bots ADD COLUMN name_new text not null default '';
UPDATE tg_bots SET name_new = COALESCE(name, '');
ALTER TABLE tg_bots DROP COLUMN name;
ALTER TABLE tg_bots RENAME COLUMN name_new TO name;

-- migrate:down
-- Revert name field to nullable
ALTER TABLE tg_bots ADD COLUMN name_new text;
UPDATE tg_bots SET name_new = name;
ALTER TABLE tg_bots DROP COLUMN name;
ALTER TABLE tg_bots RENAME COLUMN name_new TO name;