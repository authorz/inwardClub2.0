-- +goose Up
-- Identity: members, back-office accounts, staff bindings, roles.
-- legacy_* columns preserve v1 ids for migration traceability.

CREATE TABLE members (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  legacy_user_id BIGINT UNSIGNED NULL,
  wechat_openid VARCHAR(64) NULL,
  wechat_unionid VARCHAR(64) NULL,
  nickname VARCHAR(64) NOT NULL DEFAULT '',
  avatar_asset_id BIGINT UNSIGNED NULL,
  phone VARCHAR(20) NULL,
  invite_code VARCHAR(32) NULL,
  invited_by_member_id BIGINT UNSIGNED NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  token_version BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_members_openid (wechat_openid),
  UNIQUE KEY uq_members_invite_code (invite_code),
  UNIQUE KEY uq_members_legacy (legacy_user_id),
  KEY idx_members_phone (phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE admin_accounts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  legacy_admin_id BIGINT UNSIGNED NULL,
  username VARCHAR(64) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  display_name VARCHAR(64) NOT NULL DEFAULT '',
  -- role is one of the super_admin / store_admin / cashier codes.
  role VARCHAR(32) NOT NULL,
  -- store_id is NULL for global super_admin, required for store-scoped accounts.
  store_id BIGINT UNSIGNED NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  token_version BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_admin_username (username),
  KEY idx_admin_store (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE staff_accounts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  legacy_staff_id BIGINT UNSIGNED NULL,
  member_id BIGINT UNSIGNED NULL,
  wechat_openid VARCHAR(64) NULL,
  name VARCHAR(64) NOT NULL DEFAULT '',
  store_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  token_version BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  -- A WeChat user / member may bind to multiple stores, but only once per store.
  UNIQUE KEY uq_staff_openid_store (wechat_openid, store_id),
  UNIQUE KEY uq_staff_member_store (member_id, store_id),
  KEY idx_staff_store (store_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE roles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(32) NOT NULL,
  name VARCHAR(64) NOT NULL,
  audience VARCHAR(16) NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_roles_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE role_permissions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  role_code VARCHAR(32) NOT NULL,
  permission VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_role_permission (role_code, permission)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS staff_accounts;
DROP TABLE IF EXISTS admin_accounts;
DROP TABLE IF EXISTS members;
