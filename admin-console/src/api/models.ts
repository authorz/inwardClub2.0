/**
 * 后台领域实体类型（骨架阶段的最小字段集）。
 *
 * 这些类型对齐 docs/CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md 的后台 DTO 约定：
 * 详情包含 id/scopeType/storeId/status/createdAt/updatedAt/createdBy/updatedBy，
 * 金额字段以 Cent 结尾，资产字段只用 assetId。
 *
 * 服务端接口尚未落地，字段以规格为准，后续按 OpenAPI 收敛。
 */
import type { AuditableEntity } from './types'

export interface AdminUser {
  id: string
  username: string
  displayName: string
  role: string
  permissions: string[]
  subjectType: string
  audience: string
  storeId: string | null
  lastLoginAt?: string
}

export interface Store extends AuditableEntity {
  name: string
  phone?: string
  address?: string
  businessHours?: string
  logoAssetId?: string
}

export interface AccountEntity extends AuditableEntity {
  /** 后台/门店管理员账号用；员工账号无此字段 */
  username?: string
  displayName?: string
  /** 员工账号（staff_accounts）用；只有姓名 */
  name?: string
  role?: string
  /** 绑定门店名称（门店管理员 / 员工列表） */
  storeName?: string | null
  /** 门店管理员创建时一次性返回的初始密码；列表等其它场景不含此字段 */
  initialPassword?: string
}

export interface CatalogCategory extends AuditableEntity {
  name: string
  parentId?: string | null
  sortOrder?: number
  assetId?: string
}

export interface CatalogItem extends AuditableEntity {
  name: string
  itemType: string
  priceCent?: number
  stockQuantity?: number
  payChannels?: string[]
  assetId?: string
  categoryId?: string
}

export interface Activity extends AuditableEntity {
  // server admin.ActivityView 用 name（非 title）。
  name: string
  type?: string
  startAt?: string
  endAt?: string
  coverAssetId?: string
}

export interface CouponTemplate extends AuditableEntity {
  name: string
  couponType?: string
  // server admin.CouponTemplateView 提供 valueCent/totalStock/issuedCount（无 validDays）。
  valueCent?: number
  totalStock?: number
  issuedCount?: number
}

export interface Banner extends AuditableEntity {
  title: string
  assetId?: string
  linkType?: string
  linkTarget?: string
  sortOrder?: number
}

export interface RechargeProduct extends AuditableEntity {
  name: string
  /** 充值面额（整数分，展示时转元） */
  amount: number
  /** 赠送数量（金币 / 积分个数，非金额） */
  bonusAmount?: number
  /** 赠送资产类型：coin / point */
  assetType?: string
  sortOrder?: number
}

export interface MembershipTier extends AuditableEntity {
  name: string
  level: number
  /** 达到该等级所需门槛（积分 / 成长值） */
  threshold?: number
  benefits?: string
  /** VIP 海报（横幅）对象路径 objectKey：写入用，对齐后端 bannerPath */
  bannerPath?: string | null
  /** VIP 海报（横幅）展示地址：读取用，服务端由 bannerPath 解析（对齐后端 bannerUrl） */
  bannerUrl?: string
}

export interface RuleDefinition extends AuditableEntity {
  ruleKey: string
  scopeType?: string
  version?: number
  /** 规则配置（服务端为不透明 JSON） */
  configJson?: unknown
  enabled?: boolean
}

export interface BusinessOrder extends AuditableEntity {
  orderNo: string
  orderType: string
  paymentStatus: string
  orderStatus?: string
  payChannel?: string
  amountCent: number
  memberPhone?: string
  storeName?: string
}

export interface PaymentOrder extends AuditableEntity {
  paymentOrderNo: string
  businessOrderNo?: string
  storeName?: string
  orderType?: string
  amountCent: number
  payMethod?: string
  paymentStatus: string
}

// 对齐 server admin.PaymentTransactionView（支付单投影）：paymentOrderNo/payMethod/status。
export interface PaymentTransaction extends AuditableEntity {
  paymentOrderNo: string
  businessOrderNo?: string
  orderType?: string
  amountCent: number
  payMethod?: string
}

export interface RefundOrder extends AuditableEntity {
  refundOrderNo: string
  paymentOrderId?: number
  businessOrderId?: number
  storeName?: string
  amountCent: number
  channel?: string
  reason?: string
}

export interface Member {
  id: string
  nickname?: string
  phone?: string
  pointsBalance?: number
  status: string
  createdAt: string
}

/** 会员钱包分账户余额（会员详情接口返回） */
export interface WalletAccount {
  assetType: string
  availableAmount: number
  heldAmount: number
}

/** 会员详情：基础信息 + 钱包各资产余额 */
export interface MemberDetail extends Member {
  wallet: WalletAccount[]
}

export interface WalletLedgerEntry {
  id: string
  memberId: string
  assetType: string
  direction: string
  amount: number
  balanceAfter: number
  reason?: string
  sourceType?: string
  sourceId?: string
  createdAt: string
}

// 对齐 server admin.AuditLogView：actorType/actorId/targetType/targetId/storeId。
export interface AuditLog {
  id: string
  actorType: string
  actorId?: number
  storeId?: string | number | null
  action: string
  targetType?: string
  targetId?: string
  requestId?: string
  createdAt: string
}

export interface LoginEvent {
  id: string
  actor: string
  ip?: string
  userAgent?: string
  result: string
  createdAt: string
}

// 对齐 server diagnostics.ErrorEvent：method/path/status（无 code；已持久化 error_events 表）。
export interface ErrorEvent {
  id: string
  method?: string
  path?: string
  status?: number
  message: string
  requestId?: string
  createdAt: string
}

// 对齐 server payment.ChannelSetting：GET /admin/payment-channel-settings 返回渠道开关列表。
export interface PaymentChannelSetting {
  channel: string
  displayName: string
  enabled: boolean
}

/** 门店运营配置的业务字段。服务端将其整体存为不透明 JSON blob，schema 由前端约定。 */
export interface StoreSettingsData {
  businessHoursNote?: string
  autoAcceptOrders: boolean
  tableReservationEnabled: boolean
}

/** 门店运营配置接口返回：settings 为不透明配置对象，updatedAt 为服务端更新时间。 */
export interface StoreSettings {
  settings: StoreSettingsData
  updatedAt?: string
}

/** 报表：经营总览（/admin/reports/overview） */
export interface ReportOverview {
  storeCount: number
  memberCount: number
  orderCount: number
  grossSalesCent: number
  couponsIssued: number
  couponsRedeemed: number
}

/** 报表：收款趋势（/admin/reports/revenue） */
export interface RevenueReportRow {
  date: string
  orderCount: number
  grossCent: number
}

/** 报表：商品销量（/admin/reports/catalog-items） */
export interface CatalogItemReportRow {
  itemId: string
  itemName: string
  soldQty: number
  grossCent: number
}

/** 报表：活动销售（/admin/reports/activities） */
export interface ActivityReportRow {
  activityId: string
  activityName: string
  orderCount: number
  ticketCount: number
}

/** 报表：券核销（/admin/reports/coupons） */
export interface CouponReportRow {
  templateId: string
  name: string
  issued: number
  redeemed: number
}

/** 报表：核销记录（/admin/reports/records） */
export interface RecordReportRow {
  id: string
  kind: string
  createdAt: string
}

/** 报表：会员概况（/admin/reports/members） */
export interface MemberReportRow {
  memberId: string
  pointsBalance: number
  orderCount: number
}

/** 报表：预约概况（/admin/reports/reservations） */
export interface ReservationReportRow {
  date: string
  count: number
}

/** 报表：门店经营（/admin/reports/stores） */
export interface StoreReportRow {
  storeId: string
  storeName: string
  orderCount: number
  grossCent: number
}
