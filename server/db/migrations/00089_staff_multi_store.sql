-- +goose Up
ALTER TABLE staff_accounts
  DROP INDEX uq_staff_openid,
  DROP INDEX uq_staff_member,
  ADD UNIQUE KEY uq_staff_openid_store (wechat_openid, store_id),
  ADD UNIQUE KEY uq_staff_member_store (member_id, store_id);

-- +goose Down
ALTER TABLE staff_accounts
  DROP INDEX uq_staff_openid_store,
  DROP INDEX uq_staff_member_store,
  ADD UNIQUE KEY uq_staff_openid (wechat_openid),
  ADD UNIQUE KEY uq_staff_member (member_id);
