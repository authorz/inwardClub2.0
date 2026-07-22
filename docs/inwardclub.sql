SET FOREIGN_KEY_CHECKS = 0;
SET NAMES utf8mb4;
-- activities DDL
CREATE TABLE `activities` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '活动ID - 主键，自增',
`store_id` BIGINT(20) UNSIGNED NOT NULL Comment '门店ID - 活动所属门店，关联stores表',
`title` VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '活动标题 - 活动名称，最长100字符',
`image` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '活动图片 - 活动宣传图片URL，可为空',
`description` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '活动描述 - 活动详细说明，可为空',
`content` LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '活动详情 - 活动详细内容和规则',
`price` DECIMAL(8,2) NOT NULL Comment '活动价格 - 活动商品或套餐价格',
`start_time` DATETIME NOT NULL Comment '开始时间 - 活动开始时间',
`end_time` DATETIME NOT NULL Comment '结束时间 - 活动结束时间',
`status` ENUM("active","inactive","expired") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active' Comment '状态',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `activities_start_time_end_time_index`(`start_time` ASC,`end_time` ASC) USING BTREE,
INDEX `activities_store_id_status_index`(`store_id` ASC,`status` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 78 ROW_FORMAT = Dynamic COMMENT = '活动表 - 存储营销活动信息和规则';
-- activity_orders DDL
CREATE TABLE `activity_orders` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '活动订单ID - 主键，自增',
`order_no` VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '订单号 - 活动订单编号，最长32字符，唯一',
`transaction_no` VARCHAR(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '微信支付交易号',
`user_id` BIGINT(20) UNSIGNED NOT NULL Comment '用户ID - 参与用户，关联users表',
`activity_id` BIGINT(20) UNSIGNED NOT NULL Comment '活动ID - 参与的活动，关联activities表',
`amount` DECIMAL(8,2) NOT NULL Comment '订单金额 - 活动订单金额',
`payment_status` ENUM("pending","paid","failed","refunded") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' Comment '支付状态 - pending:待支付, paid:已支付, failed:支付失败, refunded:已退款',
`usage_status` ENUM("unused","used") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'unused' Comment '使用状态 - unused:未使用, used:已使用',
`qr_code` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '核销二维码 - 用于活动核销的唯一二维码',
`verification_code` VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '核销码',
`used_at` TIMESTAMP NULL Comment '使用时间 - 活动核销使用的时间，未使用时为空',
`payment_type` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '支付方式：1微信支付，2金币',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `activity_orders_activity_id_foreign`(`activity_id` ASC) USING BTREE,
INDEX `activity_orders_order_no_index`(`order_no` ASC) USING BTREE,
UNIQUE INDEX `activity_orders_order_no_unique`(`order_no` ASC) USING BTREE,
INDEX `activity_orders_user_id_index`(`user_id` ASC) USING BTREE,
UNIQUE INDEX `verification_code`(`verification_code` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 1151 ROW_FORMAT = Dynamic COMMENT = '活动订单表 - 存储用户参与活动的订单信息';
-- admins DDL
CREATE TABLE `admins` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '管理员ID - 主键，自增',
`phone` VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '手机号码 - 管理员登录手机号，唯一标识',
`password` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '密码 - 加密后的登录密码',
`name` VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '姓名 - 管理员真实姓名，最长50字符',
`role` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '管理员角色 - super_admin:超级管理员, cashier:收银员',
`store_id` BIGINT(20) UNSIGNED NULL Comment '所属门店ID - 关联stores表，超级管理员可为空，收银员必须指定',
`status` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '状态 - 1:启用, 2:禁用',
`last_login_at` TIMESTAMP NULL Comment '最后登录时间 - 管理员最后一次登录时间，可为空',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `admins_phone_index`(`phone` ASC) USING BTREE,
UNIQUE INDEX `admins_phone_unique`(`phone` ASC) USING BTREE,
INDEX `admins_store_id_role_index`(`store_id` ASC,`role` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 7 ROW_FORMAT = Dynamic COMMENT = '管理员表 - 存储系统管理员账号和权限信息';
-- balance_consumption_records DDL
CREATE TABLE `balance_consumption_records` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
`user_id` BIGINT(20) UNSIGNED NOT NULL Comment '用户ID',
`amount_consumed` DECIMAL(10,2) NOT NULL Comment '消费金额',
`balance_before` DECIMAL(10,2) NOT NULL Comment '消费前余额',
`balance_after` DECIMAL(10,2) NOT NULL Comment '消费后余额',
`consumption_type` ENUM("order","recharge","refund","transfer","other","activity") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '消费类型：order-订单消费，recharge-充值，refund-退款，transfer-转账，other-其他',
`related_type` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '关联类型（如：FoodOrder, Recharge等）',
`related_id` BIGINT(20) UNSIGNED NULL Comment '关联ID',
`description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '消费描述',
`extra_data` JSON NULL Comment '额外数据（JSON格式）',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
INDEX `balance_consumption_records_consumption_type_index`(`consumption_type` ASC) USING BTREE,
INDEX `balance_consumption_records_related_type_related_id_index`(`related_type` ASC,`related_id` ASC) USING BTREE,
INDEX `balance_consumption_records_user_id_created_at_index`(`user_id` ASC,`created_at` ASC) USING BTREE,
INDEX `balance_consumption_records_user_id_index`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 5973 ROW_FORMAT = Dynamic;
-- banner DDL
CREATE TABLE `banner` (`id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
`store_id` INT(11) NOT NULL Comment '门店ID',
`image` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL Comment 'banner图片',
`is_active` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '是否活动',
`activity_id` INT(10) NULL Comment '活动ID',
`sort` INT(10) UNSIGNED NOT NULL DEFAULT 1 Comment '排序',
`status` INT(1) UNSIGNED NULL DEFAULT 1 Comment '是否显示：1显示，2关闭',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 25 ROW_FORMAT = Dynamic COMMENT = 'banner表';
-- cache DDL
CREATE TABLE `cache` (`key` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`value` MEDIUMTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`expiration` INT(11) NOT NULL,
PRIMARY KEY (`key`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;
-- cache_locks DDL
CREATE TABLE `cache_locks` (`key` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`owner` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`expiration` INT(11) NOT NULL,
PRIMARY KEY (`key`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;
-- categories DDL
CREATE TABLE `categories` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '分类ID - 主键，自增',
`store_id` BIGINT(20) UNSIGNED NULL Comment '门店ID - 所属门店，关联stores表，NULL表示全局分类',
`parent_id` BIGINT(20) UNSIGNED NULL Comment '父分类ID - 上级分类，关联categories表，NULL表示顶级分类',
`level` TINYINT(3) UNSIGNED NOT NULL DEFAULT 1 Comment '分类层级 - 分类所在层级，1为顶级，2为二级，以此类推',
`name` VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '分类名称 - 分类显示名称，最长100字符',
`image` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '分类图片 - 分类展示图片URL，可为空',
`sort_order` INT(10) UNSIGNED NOT NULL DEFAULT 0 Comment '排序权重 - 分类显示顺序，数值越小越靠前，默认0',
`status` ENUM("active","inactive") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active' Comment '分类状态 - active:启用, inactive:禁用',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `categories_parent_id_level_index`(`parent_id` ASC,`level` ASC) USING BTREE,
INDEX `categories_store_id_sort_order_index`(`store_id` ASC,`sort_order` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 27 ROW_FORMAT = Dynamic COMMENT = '商品分类表 - 存储菜品分类信息和层级关系';
-- coin_product DDL
CREATE TABLE `coin_product` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment 'id',
`total_fee` DECIMAL(10,2) NULL Comment '金额',
`coin` DECIMAL(10,0) NULL Comment '金币',
`points` DECIMAL(10,0) NULL Comment '积分',
`status` INT(1) UNSIGNED NULL DEFAULT 1 Comment '状态：1启用，2关闭',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 6 ROW_FORMAT = Dynamic COMMENT = '金币充值产品';
-- coupon_order_items DDL
CREATE TABLE `coupon_order_items` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '订单项ID - 主键，自增',
`order_id` BIGINT(20) UNSIGNED NOT NULL Comment '订单ID - 所属订单，关联orders表',
`product_id` BIGINT(20) UNSIGNED NOT NULL Comment '商品ID - 订购商品，关联products表',
`product_type` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '产品类型-1普通商品，2券',
`product_name` VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '商品名称 - 下单时的商品名称快照，防止商品名称变更影响历史订单',
`product_price` DECIMAL(8,2) NOT NULL Comment '商品单价 - 下单时的商品价格快照，防止价格变更影响历史订单',
`quantity` INT(10) UNSIGNED NOT NULL Comment '购买数量 - 该商品的购买数量',
`subtotal` DECIMAL(10,2) NOT NULL Comment '小计金额 - 该商品的总价（单价×数量）',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `order_items_order_id_index`(`order_id` ASC) USING BTREE,
INDEX `order_items_product_id_foreign`(`product_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 1 ROW_FORMAT = Dynamic COMMENT = '券兑换订单项表 - 存储订单中的商品明细和数量';
-- failed_jobs DDL
CREATE TABLE `failed_jobs` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
`uuid` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`connection` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`queue` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`payload` LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`exception` LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`failed_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
UNIQUE INDEX `failed_jobs_uuid_unique`(`uuid` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 1 ROW_FORMAT = Dynamic;
-- food_order_items DDL
CREATE TABLE `food_order_items` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '订单项ID - 主键，自增',
`food_order_id` BIGINT(20) UNSIGNED NOT NULL Comment '订单ID - 所属订单，关联orders表',
`product_id` BIGINT(20) UNSIGNED NOT NULL Comment '商品ID - 订购商品，关联products表',
`product_type` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '产品类型-1普通商品，2券',
`product_name` VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '商品名称 - 下单时的商品名称快照，防止商品名称变更影响历史订单',
`product_price` DECIMAL(8,2) NOT NULL Comment '商品单价 - 下单时的商品价格快照，防止价格变更影响历史订单',
`quantity` INT(10) UNSIGNED NOT NULL Comment '购买数量 - 该商品的购买数量',
`subtotal` DECIMAL(10,2) NOT NULL Comment '小计金额 - 该商品的总价（单价×数量）',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `order_items_order_id_index`(`food_order_id` ASC) USING BTREE,
INDEX `order_items_product_id_foreign`(`product_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 8294 ROW_FORMAT = Dynamic COMMENT = '订单项表 - 存储订单中的商品明细和数量';
-- food_orders DDL
CREATE TABLE `food_orders` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '订单ID - 主键，自增',
`order_no` VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '订单号 - 唯一订单编号，最长32字符',
`user_id` BIGINT(20) UNSIGNED NOT NULL Comment '用户ID - 下单用户，关联users表',
`store_id` BIGINT(20) UNSIGNED NOT NULL Comment '门店ID - 下单门店，关联stores表',
`table_id` BIGINT(20) UNSIGNED NULL Comment '餐桌ID - 就餐餐桌，关联tables表，外卖订单可为空',
`total_amount` DECIMAL(10,2) NOT NULL Comment '订单总金额 - 订单总价，精确到分',
`payment_method` ENUM("wechat","cash","card","alipay","coin") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '支付方式 - wechat:微信支付, coin:金币支付,cash:现金, card:刷卡, alipay:支付宝',
`payment_status` ENUM("pending","paid","failed","refunded") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' Comment '支付状态 - pending:待支付, paid:已支付, failed:支付失败, refunded:已退款',
`order_status` ENUM("pending","confirmed","preparing","ready","completed","cancelled") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' Comment '订单状态 - pending:待确认, confirmed:已确认, preparing:制作中, ready:待取餐, completed:已完成, cancelled:已取消',
`points_earned` INT(10) UNSIGNED NOT NULL DEFAULT 0 Comment '获得积分 - 订单完成后用户获得的积分，默认0',
`transaction_id` VARCHAR(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '微信支付交易号',
`paid_at` TIMESTAMP NULL Comment '支付时间',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `orders_order_no_index`(`order_no` ASC) USING BTREE,
UNIQUE INDEX `orders_order_no_unique`(`order_no` ASC) USING BTREE,
INDEX `orders_store_id_created_at_index`(`store_id` ASC,`created_at` ASC) USING BTREE,
INDEX `orders_table_id_foreign`(`table_id` ASC) USING BTREE,
INDEX `orders_user_id_index`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 7997 ROW_FORMAT = Dynamic COMMENT = '订单表 - 存储用户订单信息和状态';
-- invitations DDL
CREATE TABLE `invitations` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '邀请记录ID - 主键，自增',
`inviter_id` BIGINT(20) UNSIGNED NOT NULL Comment '邀请人ID - 发起邀请的用户，关联users表',
`invitee_id` BIGINT(20) UNSIGNED NOT NULL Comment '被邀请人ID - 被邀请的用户，关联users表',
`inviter_points` INT(10) UNSIGNED NOT NULL DEFAULT 0 Comment '邀请人获得积分 - 邀请成功后邀请人获得的积分数量',
`invitee_points` INT(10) UNSIGNED NOT NULL DEFAULT 0 Comment '被邀请人获得积分 - 注册成功后被邀请人获得的积分数量',
`status` ENUM("pending","completed","expired") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending' Comment '邀请状态 - pending:待完成, completed:已完成, expired:已过期',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `invitations_invitee_id_index`(`invitee_id` ASC) USING BTREE,
INDEX `invitations_inviter_id_index`(`inviter_id` ASC) USING BTREE,
UNIQUE INDEX `invitations_inviter_id_invitee_id_unique`(`inviter_id` ASC,`invitee_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 1 ROW_FORMAT = Dynamic COMMENT = '邀请记录表 - 存储用户邀请关系和奖励记录';
-- job_batches DDL
CREATE TABLE `job_batches` (`id` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`name` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`total_jobs` INT(11) NOT NULL,
`pending_jobs` INT(11) NOT NULL,
`failed_jobs` INT(11) NOT NULL,
`failed_job_ids` LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`options` MEDIUMTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
`cancelled_at` INT(11) NULL,
`created_at` INT(11) NOT NULL,
`finished_at` INT(11) NULL,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;
-- jobs DDL
CREATE TABLE `jobs` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
`queue` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`payload` LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`attempts` TINYINT(3) UNSIGNED NOT NULL,
`reserved_at` INT(10) UNSIGNED NULL,
`available_at` INT(10) UNSIGNED NOT NULL,
`created_at` INT(10) UNSIGNED NOT NULL,
INDEX `jobs_queue_index`(`queue` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 8661 ROW_FORMAT = Dynamic;
-- migrations DDL
CREATE TABLE `migrations` (`id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
`migration` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`batch` INT(11) NOT NULL,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 30 ROW_FORMAT = Dynamic;
-- password_reset_tokens DDL
CREATE TABLE `password_reset_tokens` (`email` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`token` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`created_at` TIMESTAMP NULL,
PRIMARY KEY (`email`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;
-- points_consumption_records DDL
CREATE TABLE `points_consumption_records` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
`user_id` BIGINT(20) UNSIGNED NOT NULL Comment '用户ID',
`points_consumed` INT(10) UNSIGNED NOT NULL Comment '消费积分数量',
`points_before` INT(10) UNSIGNED NOT NULL Comment '消费前积分余额',
`points_after` INT(10) UNSIGNED NOT NULL Comment '消费后积分余额',
`consumption_type` ENUM("order","exchange","refund","other") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '消费类型：order-订单消费，exchange-兑换商品，refund-退款，other-其他',
`related_type` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '关联类型（如：FoodOrder, Product等）',
`related_id` BIGINT(20) UNSIGNED NULL Comment '关联ID',
`description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '消费描述',
`extra_data` JSON NULL Comment '额外数据（JSON格式）',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
INDEX `points_consumption_records_consumption_type_index`(`consumption_type` ASC) USING BTREE,
INDEX `points_consumption_records_related_type_related_id_index`(`related_type` ASC,`related_id` ASC) USING BTREE,
INDEX `points_consumption_records_user_id_created_at_index`(`user_id` ASC,`created_at` ASC) USING BTREE,
INDEX `points_consumption_records_user_id_index`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 1 ROW_FORMAT = Dynamic;
-- points_withdrawal DDL
CREATE TABLE `points_withdrawal` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '记录ID - 主键，自增',
`user_id` BIGINT(20) UNSIGNED NOT NULL Comment '用户ID - 提取积分的用户',
`store_id` BIGINT(20) UNSIGNED NOT NULL Comment '门店ID - 提取积分的门店',
`points` INT(11) NOT NULL Comment '提取积分数量 - 用户提取的积分数量',
`withdrawal_date` DATE NOT NULL Comment '提取日期 - 积分提取的日期（YYYY-MM-DD）',
`withdrawal_time` TIME NOT NULL Comment '提取时间 - 积分提取的具体时间（HH:MM:SS）',
`description` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '描述信息 - 提取积分的详细描述，可为空',
`audit_status` ENUM("pending","approved","rejected") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'approved' Comment '审核状态：pending-待审核，approved-已通过，rejected-已拒绝',
`auditor_id` BIGINT(20) UNSIGNED NULL Comment '审核员ID',
`audited_at` TIMESTAMP NULL Comment '审核时间',
`audit_remark` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '审核备注',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
INDEX `idx_audit_status`(`audit_status` ASC) USING BTREE,
INDEX `idx_auditor_id`(`auditor_id` ASC) USING BTREE,
INDEX `idx_store_id`(`store_id` ASC) USING BTREE,
INDEX `idx_user_audit_status`(`user_id` ASC,`audit_status` ASC) USING BTREE,
INDEX `idx_user_date`(`user_id` ASC,`withdrawal_date` ASC) USING BTREE,
INDEX `idx_user_id`(`user_id` ASC) USING BTREE,
INDEX `idx_withdrawal_date`(`withdrawal_date` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 9837 ROW_FORMAT = Dynamic;
-- printer_device DDL
CREATE TABLE `printer_device` (`id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT Comment 'id',
`store_id` INT(10) UNSIGNED NOT NULL Comment '门店ID',
`device_sn` VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL Comment '设备ID',
`name` VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL Comment '放置位置',
`voice` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '声音播报：1关闭，2开启',
`status` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '状态：1正常，2禁用',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
UNIQUE INDEX `sn`(`device_sn` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 3 ROW_FORMAT = Dynamic;
-- products DDL
CREATE TABLE `products` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '商品ID - 主键，自增',
`store_id` BIGINT(20) UNSIGNED NULL Comment '门店ID - 所属门店，关联stores表，NULL表示全局产品',
`category_id` BIGINT(20) UNSIGNED NOT NULL Comment '分类ID - 所属分类，关联categories表',
`name` VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '商品名称 - 菜品名称，最长100字符',
`image` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '商品图片 - 菜品展示图片URL，可为空',
`description` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '商品描述 - 菜品详细介绍，可为空',
`price` DECIMAL(8,2) NOT NULL Comment '商品价格 - 菜品单价，精确到分',
`stock` INT(10) UNSIGNED NOT NULL DEFAULT 0 Comment '库存数量 - 当前可售库存，0表示无库存',
`type` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '商品类型：1普通商品，2券',
`points` DECIMAL(10,0) NULL DEFAULT 0 Comment '赠送积分',
`sort_order` INT(10) NULL Comment '排序',
`status` ENUM("active","inactive","sold_out") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active' Comment '商品状态 - active:在售, inactive:下架, sold_out:售罄',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `products_category_id_foreign`(`category_id` ASC) USING BTREE,
INDEX `products_status_index`(`status` ASC) USING BTREE,
INDEX `products_store_id_category_id_index`(`store_id` ASC,`category_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 241 ROW_FORMAT = Dynamic COMMENT = '商品表 - 存储菜品商品信息和价格';
-- recharge DDL
CREATE TABLE `recharge` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment 'id',
`user_id` INT(10) UNSIGNED NOT NULL Comment '用户ID',
`total_fee` DECIMAL(10,2) NOT NULL DEFAULT 0.00 Comment '交易金额',
`old_total_fee` DECIMAL(10,2) NOT NULL DEFAULT 0.00 Comment '历史金额',
`coin` DECIMAL(10,0) UNSIGNED NULL DEFAULT 0 Comment '获得金币',
`points` DECIMAL(10,0) NULL Comment '获得积分',
`order_no` VARCHAR(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL Comment '订单号',
`transaction_no` VARCHAR(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL Comment '微信支付交易号',
`payment_status` INT(10) UNSIGNED NOT NULL DEFAULT 1 Comment '支付状态',
`status` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '订单状态',
`paid_at` TIMESTAMP NULL,
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
UNIQUE INDEX `order_no`(`order_no` ASC) USING BTREE,
INDEX `status`(`status` ASC) USING BTREE,
INDEX `user_id`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 1561 ROW_FORMAT = Dynamic COMMENT = '充值记录表';
-- reservations DDL
CREATE TABLE `reservations` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '预约ID - 主键，自增',
`user_id` BIGINT(20) UNSIGNED NOT NULL Comment '用户ID - 预约用户，关联users表',
`store_id` BIGINT(20) UNSIGNED NOT NULL Comment '门店ID - 预约门店，关联stores表',
`table_id` BIGINT(20) UNSIGNED NOT NULL Comment '餐桌ID - 预约餐桌，关联tables表',
`seat_id` INT(10) UNSIGNED NOT NULL Comment '座位ID - 预约座位，关联seats表',
`username` VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '用户昵称，入库，减少查询次数',
`user_avatar` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '用户头像',
`user_sex` INT(1) UNSIGNED NULL Comment '用户性别',
`status` INT(10) UNSIGNED NOT NULL DEFAULT 1 Comment '预约状态 - 1:待入座, 2:已入座,取消直接删记录',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
INDEX `reservations_store_id_reservation_time_index`(`store_id` ASC) USING BTREE,
INDEX `reservations_table_id_reservation_time_index`(`table_id` ASC) USING BTREE,
INDEX `reservations_user_id_index`(`user_id` ASC) USING BTREE,
INDEX `seat_id`(`seat_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 18 ROW_FORMAT = Dynamic COMMENT = '预约表 - 存储用户餐桌预约信息和状态';
-- save_points DDL
CREATE TABLE `save_points` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment 'id',
`store_id` BIGINT(20) NOT NULL Comment '门店ID',
`user_id` BIGINT(20) NOT NULL Comment '用户ID',
`save_points` DECIMAL(20,0) NOT NULL Comment '存入积分',
`points` DECIMAL(20,0) UNSIGNED NOT NULL Comment '实际获得积分',
`verify_staff` INT(11) NULL Comment '验证员工',
`verify_time` DATETIME NULL Comment '验证时间',
`remark` VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL Comment '备注',
`status` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '订单状态：1待审核，2确认，3拒绝',
`created_at` DATETIME NULL Comment '提交时间',
`updated_at` DATETIME NULL Comment '更新时间',
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
INDEX `store_id`(`store_id` ASC) USING BTREE,
INDEX `user_id`(`user_id` ASC) USING BTREE,
INDEX `verify_staff`(`verify_staff` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 1195 ROW_FORMAT = Dynamic COMMENT = '存入积分记录';
-- seats DDL
CREATE TABLE `seats` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '座位ID - 主键，自增',
`store_id` INT(11) NULL Comment '门店ID',
`table_id` BIGINT(20) UNSIGNED NOT NULL Comment '餐桌ID - 关联tables表',
`seat_number` VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '座位编号 - 座位在餐桌中的编号，如A1、A2等',
`status` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '座位状态 - 1:可用, 2:已预约, 3:游戏中, 4:维护中',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
INDEX `seats_status_index`(`status` ASC) USING BTREE,
INDEX `seats_table_id_index`(`table_id` ASC) USING BTREE,
UNIQUE INDEX `seats_table_seat_unique`(`table_id` ASC,`seat_number` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 28 ROW_FORMAT = Dynamic COMMENT = '座位表 - 存储餐桌座位详细信息和状态';
-- sessions DDL
CREATE TABLE `sessions` (`id` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`user_id` BIGINT(20) UNSIGNED NULL,
`ip_address` VARCHAR(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
`user_agent` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
`payload` LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
`last_activity` INT(11) NOT NULL,
INDEX `sessions_last_activity_index`(`last_activity` ASC) USING BTREE,
INDEX `sessions_user_id_index`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;
-- settings DDL
CREATE TABLE `settings` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment 'id',
`desc` VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL Comment '设置名称',
`key` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
`value` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
UNIQUE INDEX `key`(`key` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 7 ROW_FORMAT = Dynamic COMMENT = '全局设置表';
-- sign_in DDL
CREATE TABLE `sign_in` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment 'id',
`user_id` INT(11) UNSIGNED NOT NULL Comment '用户id',
`points` DECIMAL(10,0) NULL Comment '获得积分',
`sign_date` DATE NULL Comment '签到日期',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
INDEX `user_id`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 1 ROW_FORMAT = Dynamic COMMENT = '用户签到表';
-- staff DDL
CREATE TABLE `staff` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '员工ID - 主键，自增',
`store_id` BIGINT(20) UNSIGNED NOT NULL Comment '门店ID - 所属门店，关联stores表',
`user_id` BIGINT(20) UNSIGNED NOT NULL Comment '用户ID - 关联的用户账号，关联users表',
`name` VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '员工姓名 - 员工真实姓名，最长50字符',
`phone` VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '联系电话 - 员工联系电话，最长20位',
`status` ENUM("active","inactive","resigned") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active' Comment '员工状态 - active:在职, inactive:停职, resigned:离职',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `staff_store_id_index`(`store_id` ASC) USING BTREE,
UNIQUE INDEX `staff_store_id_user_id_unique`(`store_id` ASC,`user_id` ASC) USING BTREE,
INDEX `staff_user_id_foreign`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 1 ROW_FORMAT = Dynamic COMMENT = '员工表 - 存储员工基本信息和权限';
-- stores DDL
CREATE TABLE `stores` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '门店ID - 主键，自增',
`name` VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '门店名称 - 门店显示名称，最长100字符',
`logo` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' Comment '门店Logo - Logo图片URL地址，默认为空字符串',
`description` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '门店描述 - 门店详细介绍，可为空',
`latitude` DECIMAL(10,7) NULL Comment '纬度 - 门店地理位置纬度坐标，精度7位小数',
`longitude` DECIMAL(10,7) NULL Comment '经度 - 门店地理位置经度坐标，精度7位小数',
`address` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '门店地址 - 门店详细地址信息',
`phone` VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' Comment '联系电话 - 门店联系电话，最长20位，默认为空',
`status` ENUM("active","inactive") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active' Comment '门店状态 - active:营业中, inactive:暂停营业',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `stores_latitude_longitude_index`(`latitude` ASC,`longitude` ASC) USING BTREE,
INDEX `stores_status_index`(`status` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 2 ROW_FORMAT = Dynamic COMMENT = '门店表 - 存储餐厅门店基本信息和营业状态';
-- tables DDL
CREATE TABLE `tables` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '餐桌ID - 主键，自增',
`store_id` BIGINT(20) UNSIGNED NOT NULL Comment '门店ID - 所属门店，关联stores表',
`table_name` VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '桌子名称',
`table_number` VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '桌号 - 餐桌编号，最长20字符，如A01、B02等',
`basic_coin` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '基础积分',
`seat_count` TINYINT(3) UNSIGNED NOT NULL DEFAULT 1 Comment '座位数 - 该餐桌可容纳的人数，默认1人',
`status` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '餐桌状态 - 1:可预约, 2:暂停预约, 3:座位已满, 4:维护中',
`qr_code` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '桌码 - 餐桌二维码标识，用于扫码点餐，唯一',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
UNIQUE INDEX `tables_qr_code_unique`(`qr_code` ASC) USING BTREE,
INDEX `tables_store_id_status_index`(`store_id` ASC,`status` ASC) USING BTREE,
UNIQUE INDEX `tables_store_id_table_number_unique`(`store_id` ASC,`table_number` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 5 ROW_FORMAT = Dynamic COMMENT = '餐桌表 - 存储门店内餐桌信息和状态';
-- transaction_records DDL
CREATE TABLE `transaction_records` (`id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
`user_id` INT(10) UNSIGNED NOT NULL Comment '用户ID',
`amount_type` ENUM("balance","points","all_balance") CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL Comment '金额类型：balance：金币，points:积分',
`trans_type` INT(1) NOT NULL Comment '交易类型：1收入，2支出',
`amount` DECIMAL(10,2) NOT NULL Comment '交易金额',
`balance_before` DECIMAL(10,2) NOT NULL DEFAULT 0.00 Comment '交易前余额',
`balance_after` DECIMAL(10,2) NOT NULL Comment '交易后余额',
`description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL Comment '交易描述',
`related_id` INT(10) UNSIGNED NULL Comment '关联ID',
`remark` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL,
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
INDEX `amount_type`(`amount_type` ASC) USING BTREE,
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
INDEX `related_id`(`related_id` ASC) USING BTREE,
INDEX `trans_type`(`trans_type` ASC) USING BTREE,
INDEX `user_id`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 24406 ROW_FORMAT = Dynamic COMMENT = '交易记录表';
-- user_coupon DDL
CREATE TABLE `user_coupon` (`id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
`user_id` INT(10) UNSIGNED NOT NULL Comment '用户ID',
`product_id` INT(10) UNSIGNED NOT NULL Comment '产品ID',
`order_no` VARCHAR(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL Comment '订单号',
`name` VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL Comment '产品名称',
`description` VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL Comment '产品简介',
`price` DECIMAL(10,2) NOT NULL DEFAULT 0.00 Comment '产品金额',
`points` DECIMAL(10,0) NULL DEFAULT 0 Comment '赠送积分',
`expired_time` DATETIME NULL Comment '过期时间',
`status` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '券状态：1未使用，2已使用',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
UNIQUE INDEX `id`(`id` ASC) USING BTREE,
INDEX `status`(`status` ASC) USING BTREE,
INDEX `user_id`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 8649 ROW_FORMAT = Dynamic COMMENT = '用户券';
-- user_points DDL
CREATE TABLE `user_points` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '记录ID - 主键，自增',
`user_id` BIGINT(20) UNSIGNED NOT NULL Comment '用户ID - 积分变动的用户，关联users表',
`points` INT(11) NOT NULL Comment '积分变动 - 正数为获得积分，负数为消费积分',
`type` ENUM("earn","spend","expire","adjust") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '变动类型 - earn:获得, spend:消费, expire:过期, adjust:调整',
`source` VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '积分来源 - 积分变动的来源说明，如订单、活动、调整等',
`source_id` BIGINT(20) UNSIGNED NULL Comment '来源ID - 关联的业务记录ID，如订单ID、活动ID等',
`description` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '变动说明 - 积分变动的详细描述，可为空',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL,
INDEX `user_points_user_id_created_at_index`(`user_id` ASC,`created_at` ASC) USING BTREE,
INDEX `user_points_user_id_index`(`user_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 1117 ROW_FORMAT = Dynamic COMMENT = '用户积分记录表 - 存储用户积分变动的详细记录';
-- users DDL
CREATE TABLE `users` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment '用户ID - 主键，自增',
`openid` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '微信OpenID - 微信用户唯一标识，用于微信登录',
`nickname` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '用户昵称 - 用户显示名称，可为空',
`avatar` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '用户头像 - 头像图片URL地址，可为空',
`phone` VARCHAR(11) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL Comment '手机号码 - 用户联系电话，最长11位，可为空',
`user_type` ENUM("customer","staff") CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'customer' Comment '用户类型 - customer:顾客, staff:员工',
`store_id` BIGINT(20) UNSIGNED NULL Comment '所属门店ID - 员工所属门店，顾客为空，关联stores表',
`total_points` INT(10) UNSIGNED NOT NULL DEFAULT 0 Comment '总积分 - 用户累计获得的总积分数',
`used_points` INT(10) UNSIGNED NOT NULL DEFAULT 0 Comment '已使用积分 - 用户已消费的积分数',
`invite_code` VARCHAR(8) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL Comment '邀请码 - 用户专属邀请码，唯一标识',
`invited_by` BIGINT(20) UNSIGNED NULL Comment '邀请人ID - 邀请该用户的用户ID，关联users表',
`balance` DECIMAL(10,2) NULL DEFAULT 0.00 Comment '用户余额',
`all_balance` DECIMAL(20,2) NULL DEFAULT 0.00 Comment '用户所有的充值金额',
`points` DECIMAL(10,0) NULL DEFAULT 0 Comment '用户积分',
`sex` INT(1) NULL Comment '用户性别：1男，2女',
`level` INT(2) UNSIGNED NOT NULL DEFAULT 1 Comment '会员等级',
`created_at` TIMESTAMP NULL Comment '创建时间 - 记录创建时间',
`updated_at` TIMESTAMP NULL Comment '更新时间 - 记录最后更新时间',
INDEX `users_invite_code_index`(`invite_code` ASC) USING BTREE,
UNIQUE INDEX `users_invite_code_unique`(`invite_code` ASC) USING BTREE,
INDEX `users_invited_by_index`(`invited_by` ASC) USING BTREE,
INDEX `users_openid_index`(`openid` ASC) USING BTREE,
UNIQUE INDEX `users_openid_unique`(`openid` ASC) USING BTREE,
INDEX `users_store_id_index`(`store_id` ASC) USING BTREE,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci AUTO_INCREMENT = 3602 ROW_FORMAT = Dynamic COMMENT = '用户表 - 存储系统用户信息，包括顾客和员工';
-- vip_level DDL
CREATE TABLE `vip_level` (`id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT Comment 'id',
`level` INT(2) NOT NULL Comment '等级标识',
`name` VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL Comment '等级名称',
`points` DECIMAL(10,0) NOT NULL DEFAULT 0 Comment '所需积分',
`image` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL Comment '等级图片',
`status` INT(1) UNSIGNED NOT NULL DEFAULT 1 Comment '状态：1显示，2隐藏',
`created_at` TIMESTAMP NULL,
`updated_at` TIMESTAMP NULL,
PRIMARY KEY (`id`)) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_general_ci AUTO_INCREMENT = 9 ROW_FORMAT = Dynamic COMMENT = '会员等级表';
-- admins Constraints
ALTER TABLE `admins` 
 ADD CONSTRAINT `admins_store_id_foreign` FOREIGN KEY (`store_id`) REFERENCES `stores` (`id`) ON DELETE SET NULL ON UPDATE RESTRICT;
-- balance_consumption_records Constraints
ALTER TABLE `balance_consumption_records` 
 ADD CONSTRAINT `balance_consumption_records_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT;
-- categories Constraints
ALTER TABLE `categories` 
 ADD CONSTRAINT `categories_parent_id_foreign` FOREIGN KEY (`parent_id`) REFERENCES `categories` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT,
ADD CONSTRAINT `categories_store_id_foreign` FOREIGN KEY (`store_id`) REFERENCES `stores` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
-- coupon_order_items Constraints
ALTER TABLE `coupon_order_items` 
 ADD CONSTRAINT `coupon_order_items_ibfk_1` FOREIGN KEY (`order_id`) REFERENCES `food_orders` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
ADD CONSTRAINT `coupon_order_items_ibfk_2` FOREIGN KEY (`product_id`) REFERENCES `products` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
-- food_order_items Constraints
ALTER TABLE `food_order_items` 
 ADD CONSTRAINT `order_items_order_id_foreign` FOREIGN KEY (`food_order_id`) REFERENCES `food_orders` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
ADD CONSTRAINT `order_items_product_id_foreign` FOREIGN KEY (`product_id`) REFERENCES `products` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
-- food_orders Constraints
ALTER TABLE `food_orders` 
 ADD CONSTRAINT `orders_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
ADD CONSTRAINT `orders_store_id_foreign` FOREIGN KEY (`store_id`) REFERENCES `stores` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
ADD CONSTRAINT `orders_table_id_foreign` FOREIGN KEY (`table_id`) REFERENCES `tables` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
-- invitations Constraints
ALTER TABLE `invitations` 
 ADD CONSTRAINT `invitations_invitee_id_foreign` FOREIGN KEY (`invitee_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
ADD CONSTRAINT `invitations_inviter_id_foreign` FOREIGN KEY (`inviter_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
-- points_consumption_records Constraints
ALTER TABLE `points_consumption_records` 
 ADD CONSTRAINT `points_consumption_records_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT;
-- points_withdrawal Constraints
ALTER TABLE `points_withdrawal` 
 ADD CONSTRAINT `points_withdrawal_auditor_id_foreign` FOREIGN KEY (`auditor_id`) REFERENCES `users` (`id`) ON DELETE SET NULL ON UPDATE RESTRICT,
ADD CONSTRAINT `points_withdrawal_store_id_foreign` FOREIGN KEY (`store_id`) REFERENCES `stores` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT,
ADD CONSTRAINT `points_withdrawal_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT;
-- reservations Constraints
ALTER TABLE `reservations` 
 ADD CONSTRAINT `reservations_table_id_foreign` FOREIGN KEY (`table_id`) REFERENCES `tables` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
ADD CONSTRAINT `reservations_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
ADD CONSTRAINT `reservations_store_id_foreign` FOREIGN KEY (`store_id`) REFERENCES `stores` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
-- staff Constraints
ALTER TABLE `staff` 
 ADD CONSTRAINT `staff_store_id_foreign` FOREIGN KEY (`store_id`) REFERENCES `stores` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
ADD CONSTRAINT `staff_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
-- tables Constraints
ALTER TABLE `tables` 
 ADD CONSTRAINT `tables_store_id_foreign` FOREIGN KEY (`store_id`) REFERENCES `stores` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
-- user_points Constraints
ALTER TABLE `user_points` 
 ADD CONSTRAINT `user_points_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION;
SET FOREIGN_KEY_CHECKS = 1;
