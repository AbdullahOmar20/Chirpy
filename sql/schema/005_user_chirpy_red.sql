-- +goose up
ALTER TABLE users 
ADD COLUMN is_chirpy_red BOOlEAN DEFAULT FALSE;


-- +goose down
ALTER TABLE users 
DROP COLUMN is_chirpy_red;