-- +goose Up
ALTER TABLE members
  ADD COLUMN gender VARCHAR(16) NULL COMMENT 'male|female|other'
  AFTER avatar_url;

-- +goose Down
ALTER TABLE members DROP COLUMN gender;
