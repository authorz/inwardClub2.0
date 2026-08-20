-- +goose Up
-- Empty phone values are represented as NULL so MySQL can keep multiple
-- OpenID-only members while enforcing uniqueness for every bound phone.
UPDATE members SET phone = NULL WHERE phone IS NOT NULL AND TRIM(phone) = '';

ALTER TABLE members
  DROP INDEX idx_members_phone,
  ADD UNIQUE KEY uq_members_phone (phone);

-- +goose Down
ALTER TABLE members
  DROP INDEX uq_members_phone,
  ADD KEY idx_members_phone (phone);
