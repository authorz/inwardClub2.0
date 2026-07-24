-- +goose Up
ALTER TABLE members
  ADD COLUMN avatar_url VARCHAR(512) NULL
  AFTER nickname;

-- +goose Down
ALTER TABLE members DROP COLUMN avatar_url;
