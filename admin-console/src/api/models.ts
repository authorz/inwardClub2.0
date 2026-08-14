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
  customerServiceQrAssetId?: string | number | null
  customerServiceQrUrl?: string
  address?: string
  businessHours?: string
  /** GPS 坐标：小程序据此计算门店距离并支持"导航前往"，缺失则两者都不可用 */
  latitude?: number | null
  longitude?: number | null
  logoAssetId?: string
}

export interface VenueTable extends AuditableEntity {
  storeName: string
  name: string
  code: string
  capacity: number
  seatCount: number
  basePoints: number
  layoutAssetId?: string | number | null
  layoutUrl?: string
}

export interface VenueSeat extends AuditableEntity {
  storeName: string
  tableId: string | number
  tableName: string
  name: string
}

export interface AccountEntity extends AuditableEntity {
  /** 后台/门店管理员账号用；员工账号无此字段 */
  username?: string
  displayName?: string
  /** 员工账号（staff_accounts）用；只有姓名 */
  name?: string
  role?: string
  /** 系统管理员不可删除或禁用，但可编辑姓名和密码 */
  isSystem?: boolean
  /** 员工账号：绑定的小程序会员 id 与手机号（打码） */
  memberId?: number
  phone?: string
  /** 绑定门店名称（门店管理员 / 员工列表） */
  storeName?: string | null
}

export interface CatalogCategory extends AuditableEntity {
  name: string
  storeName?: string
  parentId?: string | null
  sortOrder?: number
  assetId?: string | number | null
  imageUrl?: string
}

export interface CatalogItem extends AuditableEntity {
  name: string
  itemType: string
  description?: string
  storeName?: string
  categoryName?: string
  imageUrl?: string
  priceCent?: number
  stockQuantity?: number
  payChannels?: string[]
  assetId?: string | number
  categoryId?: string | number
  pointsReward?: number
  sortOrder?: number
}

export interface Activity extends AuditableEntity {
  /** 列表接口使用 name，详情/写接口使用 title。 */
  name?: string
  title?: string
  type?: string
  description?: string
  content?: string
  assetId?: string | number | null
  imageUrl?: string
  startAt?: string
  endAt?: string
  payChannels?: string[]
  purchaseLimitPerMember?: number
}

export interface ActivityTicketType {
  id: string | number
  activityId: string | number
  name: string
  priceCent: number
  stockQuantity: number
  soldQuantity?: number
  saleStartAt?: string
  saleEndAt?: string
  payChannels: string[]
  maxTicketsPerOrder: number
  status: string
}

export interface CouponTemplate extends AuditableEntity {
  name: string
  description?: string
  couponType: string
  valueCent: number
  pointsPrice?: number
  stockQuantity?: number
  issuedQuantity?: number
  perMemberLimit?: number
  totalStock?: number
  issuedCount?: number
  scopeType?: string
  storeId?: string | number | null
}

export interface Banner extends AuditableEntity {
  title: string
  assetId?: string | number | null
  imageUrl?: string
  linkUrl?: string
  sortOrder?: number
}

export interface RechargeProduct extends AuditableEntity {
  /** 实际支付金额（整数分） */
  amountCent: number
  /** 支付成功后到账的金币总数 */
  coinAmount: number
  /** 支付成功后赠送的积分数 */
  pointsAmount: number
  /** 支付成功后赠送的优惠券模板；为空表示不赠券 */
  couponTemplateId?: string | number | null
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
  paymentOrderId?: number
  refundStatus?: string
  memberAvatarUrl?: string
  memberNickname?: string
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
  businessOrderNo: string
  storeName?: string
  orderAmountCent: number
  memberId?: number
  memberNickname?: string
  memberPhone?: string
  memberAvatarUrl?: string
  amountCent: number
  channel?: string
  reason?: string
  orderCreatedAt: string
  operatedAt: string
}

export interface Member {
  id: string
  nickname?: string
  phone?: string
  avatarUrl?: string
  gender?: string
  pointsBalance?: number
  coinsBalance?: number
  vipTierName?: string
  vipLevel?: number
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
  recordKey: string
  memberId: string
  memberNickname?: string
  memberPhone?: string
  memberAvatarUrl?: string
  storeId?: string
  storeName?: string
  assetType: string
  direction: string
  amount: number
  balanceAfter?: number
  status: string
  reason?: string
  sourceType?: string
  sourceId?: string
  relatedOrderNo?: string
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

/** 员工审核存积分时使用的总部统一比例配置。 */
export interface PointReviewSettings {
  pointsDivisor: number
  coinPointsDivisor: number
  version: number
  updatedAt?: string
}

/** 总后台统一展示配置。 */
export interface GlobalSettings {
  tableDefaultBackgroundUrl: string
  firstRechargeDoublePointsEnabled: boolean
  rechargeDoublePointsThresholdAmount: number
  franchiseInquirySources: string[]
  franchiseHotline: string
  phoneChangeIntervalDays: number
  updatedAt?: string
}

export interface StoreLowSpendRule {
  storeId: number
  storeName: string
  configured: boolean
  enabled: boolean
  reservationCutoff: string
  consumptionCutoff: string
  minimumAmount: number
  rewardPoints: number
  updatedAt?: string
}

/** 小程序提交的加盟咨询。 */
export interface FranchiseInquiry {
  id: number
  memberId?: number
  memberNickname?: string
  memberPhone?: string
  memberAvatarUrl?: string
  contactName: string
  phone: string
  expectedRegion: string
  source: string
  status: 'unprocessed' | 'processed'
  processedAt?: string
  createdAt: string
}

/** 报表：经营总览（/admin/reports/overview） */
export interface ReportOverviewBreakdown {
  total: number
  today: number
  recharge: number
  food: number
  activity: number
  todayRecharge: number
  todayFood: number
  todayActivity: number
}

export interface ReportOverviewTrendPoint {
  date: string
  wechatRevenueCent: number
  orderCount: number
}

export interface ReportOverview {
  storeCount: number
  memberCount: number
  orderCount: number
  grossSalesCent: number
  todayOrderCount: number
  todayGrossSalesCent: number
  todayNewMemberCount: number
  activityRevenueCent: number
  todayActivityRevenueCent: number
  couponsIssued: number
  couponsRedeemed: number
  wechatRevenue: ReportOverviewBreakdown
  coinConsumption: ReportOverviewBreakdown
  trend: ReportOverviewTrendPoint[]
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
