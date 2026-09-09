DROP TABLE IF EXISTS encuestas_grupos;
DROP TABLE IF EXISTS circulares_grupos;
ALTER TABLE encuestas  DROP COLUMN IF EXISTS para_todos;
ALTER TABLE circulares DROP COLUMN IF EXISTS para_todos;
