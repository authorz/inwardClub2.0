# InwardClub v1 → v2 数据库字段级迁移映射

> 生成时间：2026-07-22  
> 数据源：`docs/inwardclub.sql`（1.0 Laravel 导出，37 张表）vs `server/db/migrations/00001~00021`（2.0 goose 迁移）  
> 排除框架表：cache / cache_locks / failed_jobs / job_batches / jobs / migrations / password_reset_tokens / sessions / settings（KV）

---

## 一、表级对应总表

| # | 1.0 表 | 2.0 表 | 关系 | 可迁移性 |
|---|--------|--------|------|----------|
| 1 | `users` | `members` + `wallet_accounts` + `wallet_ledger_entries` | 拆分 | **有损**（钱字段拆账本+造期初分录；sex 字段无处放） |
| 2 | `user_points` | `wallet_ledger_entries` | 合并 | 有损（枚举/单位/来源语义变化；balance_after 需重算） |
| 3 | `save_points` | `point_savings` | 重命名 | 有损（丢 save_points/points 双值、verify_staff） |
| 4 | `points_withdrawal` | `point_withdrawals` | 重命名 | 有损（丢 withdrawal_date/time、audit_remark） |
| 5 | `points_consumption_records` | `wallet_ledger_entries` | 合并 | 有损（并入账本，字段收窄） |
| 6 | `balance_consumption_records` | `wallet_ledger_entries` | 合并 | 有损（同上） |
| 7 | `transaction_records` | `wallet_ledger_entries` | 合并 | 有损（多账本混合表拆分到按 asset_type 分账） |
| 8 | `sign_in` | `sign_in_records` | 重命名 | 完整（streak_days 需补算） |
| 9 | `recharge` | `business_orders` + `payment_orders` + `payment_transactions` | 拆分 | **有损**（元→分；old_total_fee 无处放；一行拆三表） |
| 10 | `coin_product` | `recharge_products` | 重命名 | **有损**（coin+points 双资产→单 asset_type；元→分） |
| 11 | `vip_level` | `membership_tiers` | 重命名 | 完整（image URL→asset；数据量小） |
| 12 | `activities` | `activities` | 同名重构 | 有损（price 移到 ticket_types；status 枚举不一致） |
| 13 | `activity_orders` | `activity_orders` + `tickets` + `business_orders` + `payment_orders` | 拆分 | **有损**（一行拆四表；需造 ticket_type） |
| 14 | `products` | `catalog_items` (+`catalog_variants`, `store_item_overrides`) | 拆分 | 有损（单位、type、status 枚举；image URL→asset） |
| 15 | `categories` | `catalog_categories` | 重命名 | 完整（level 可由 parent 推导） |
| 16 | `food_orders` | `food_orders` + `business_orders` + `payment_orders` | 拆分 | **有损**（payment_method 枚举收窄；一行拆三表） |
| 17 | `food_order_items` | `food_order_items` | 同名重构 | 有损（product_type 丢失；元→分） |
| 18 | `coupon_order_items` | *(无独立表)* → `coupon_redemptions.item_snapshot_json` | 降级 | **有损**（结构化→JSON 快照；数据量≈0） |
| 19 | `user_coupon` | `coupon_entitlements` + `coupon_templates` | 拆分 | **有损**（需反向造 coupon_templates；entitlement_no 需生成） |
| 20 | `stores` | `stores` | 同名重构 | 完整（logo URL→asset；description 丢失） |
| 21 | `tables` | `tables` | 同名重构 | **有损**（丢 table_number/basic_coin/qr_code） |
| 22 | `seats` | `seats` | 同名重构 | 有损（seat_number 唯一约束放松；status 枚举变化） |
| 23 | `reservations` | `reservations` | 同名重构 | 有损（丢用户快照字段；status 枚举变化） |
| 24 | `invitations` | *(无独立表)* → `members.invited_by_member_id` + `benefit_grants` | 降级 | **有损**（无邀请记录表；数据量≈0） |
| 25 | `staff` | `staff_accounts` | 重命名 | 有损（丢 phone；多店→一人一店 UNIQUE 收紧） |
| 26 | `admins` | `admin_accounts` | 重命名 | 有损（phone→username；role int→string；丢 last_login_at） |
| 27 | `banner` | `banners` | 重命名 | **有损**（丢 activity_id/is_active；image URL→asset） |
| 28 | `printer_device` | `printer_devices` | 重命名 | 有损（丢 voice 语音播报开关） |

---

## 二、迁移阻塞点清单（必须在 ETL 中处理）

### A. 全库金额单位：DECIMAL(元) → BIGINT(分)
- 影响表：`recharge`、`food_orders`、`food_order_items`、`activity_orders`、`products`、`coin_product`、`user_coupon`、`users.balance`
- 处理：ETL 层统一 ×100，逐字段对账，避免浮点误差

### B. 用户余额/积分 → 账本期初分录
- `users.balance/all_balance/points/total_points/used_points` 五个字段语义重叠
- 2.0 `wallet_ledger_entries.balance_after` NOT NULL，需重算每行快照
- 处理：先对账取真值 → 建 `wallet_accounts` → 造期初 credit 分录

### C. 订单三层化（business_order 中间实体缺失）
- 1.0 每笔订单 1 行 → 2.0 需合成 business_order + payment_order + payment_transaction
- `food_orders.business_order_id` / `activity_orders.business_order_id` 均 NOT NULL UNIQUE
- 处理：ETL 为每笔老订单生成三套单号

### D. 图片 URL → asset_id 外键
- 影响：`users.avatar`、`stores.logo`、`activities.image`、`products.image`、`banner.image`、`categories.image`、`vip_level.image`
- 2.0 `assets` 表需要 bucket/object_key/etag 等完整元数据
- 处理：批量建 assets 记录（元数据可能缺失），或丢弃历史图片

### E. status 默认值反向（draft vs active）
- `catalog_items.status DEFAULT 'draft'`、`activities.status DEFAULT 'draft'`
- 1.0 对应字段默认 'active'
- 处理：ETL 显式把老 active → published，否则历史商品/活动全变草稿

### F. coin_product 一档双资产 → 单 asset_type
- 1.0：`coin_product.coin` + `coin_product.points` 同行
- 2.0：`recharge_products.asset_type` 只能是 coin 或 point 之一
- 处理：拆两行，或业务确认丢弃其中一种

### G. staff 多店 → 一人一店 UNIQUE 收紧
- 1.0：UNIQUE(store_id, user_id)，同一 user 可在多店当员工
- 2.0：UNIQUE(wechat_openid) / UNIQUE(member_id)，全局唯一
- 处理：迁移前核验生产库有无一人多店记录

---

## 三、明确丢失字段（2.0 无列可放）

| 字段 | 所在 1.0 表 | 说明 |
|------|------------|------|
| `sex` | `users` | 性别，2.0 members 无此列 |
| `description` | `stores` | 门店描述 |
| `table_number` | `tables` | 桌号编号 |
| `basic_coin` | `tables` | 桌台基础积分 |
| `qr_code` | `tables` | 桌码 URL |
| `phone` | `staff` | 员工联系电话 |
| `old_total_fee` | `recharge` | 历史金额 |
| `product_type` | `food_order_items` | 券商品标记 |
| `withdrawal_date/time` | `points_withdrawal` | 取分日期时间 |
| `audit_remark` | `points_withdrawal` | 审核备注 |
| `save_points`(双值) | `save_points` | 存入 vs 实际获得两个数 |
| `verify_staff` | `save_points` | 审核员工 |
| `qr_code` | `activity_orders` | 核销二维码 URL |
| `voice` | `printer_device` | 语音播报开关 |
| `activity_id`/`is_active` | `banner` | Banner 关联活动、活动开关 |
| `inviter_points`/`invitee_points`/`status` | `invitations` | 邀请奖励明细和状态机 |

---

## 四、低风险/基本完整可迁移

- `vip_level → membership_tiers`：仅 image 需转 asset，配置数据量小（AUTO_INCREMENT=9）
- `categories → catalog_categories`：level 可由 parent 推导
- `sign_in → sign_in_records`：streak_days 需补算，UNIQUE(member_id,sign_date) 天然去重
- `admins → admin_accounts`：账号数极少（AUTO_INCREMENT=7）

---

## 五、2.0 完全没有对应表的 1.0 表

| 1.0 表 | 状态 | 说明 |
|--------|------|------|
| `coupon_order_items` | 无关系表 | 降级为 `coupon_redemptions.item_snapshot_json` JSON；数据量≈0 |
| `invitations` | 无独立表 | 降级为 `members.invited_by_member_id` + `benefit_grants`；数据量≈0 |
| `balance_consumption_records` / `points_consumption_records` / `transaction_records` / `user_points` | 合并进账本 | 四张流水表全部并入 `wallet_ledger_entries` |

---

## 六、ETL 实现路径（cmd/reconcile）

当前状态：`server/cmd/reconcile/main.go` 是空壳；`server/db/migrations/00013_migration.sql` 已建好支撑表（`legacy_id_maps`、`migration_runs`、`reconciliation_results`）；各 2.0 业务表已留 `legacy_*` 列（如 `members.legacy_user_id`、`stores.legacy_store_id`）。

建议迁移顺序：
1. **基础数据**：stores → catalog_categories → catalog_items → membership_tiers → printer_devices → banners
2. **身份**：users → members（含 wallet_accounts 期初分录）
3. **员工/管理员**：staff → staff_accounts；admins → admin_accounts
4. **积分流水**：user_points + save_points + points_withdrawal → wallet_ledger_entries + point_savings + point_withdrawals
5. **券**：user_coupon → coupon_templates（反向造）+ coupon_entitlements
6. **订单**：recharge / food_orders / activity_orders → business_orders + payment_orders + payment_transactions + 各业务订单表
7. **预约/签到**：reservations → reservations；sign_in → sign_in_records
8. **对账验证**：reconciliation_results 逐表写入差异报告
