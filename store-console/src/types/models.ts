/**
 * 门店后台领域模型类型。
 *
 * 字段以接口文档/规格为准；第一阶段部分为骨架，标注 TODO 待服务端最终 DTO 对齐。
 * 金额字段统一整数分，字段名以 Cent 结尾。
 */

import type {
  ActiveStatus,
  CollectionOrderStatus,
  FoodOrderStatus,
  PayChannel,
  PaymentStatus,
  PublishStatus,
  RefundStatus,
  ReservationStatus,
  ReviewStatus,
  ScopeType,
  VerificationKind,
  VerificationStatus,
} from '@/constants/enums'

/** 所有后台详情响应的公共审计字段。 */
export interface AuditFields {
  createdAt?: string
  updatedAt?: string
  createdBy?: string
  updatedBy?: string
}

export interface StoreOrder extends AuditFields {
  id: number | string
  businessOrderId: number | string
  orderNo: string
  orderType: string
  paymentStatus: PaymentStatus
  orderStatus?: string
  payChannel?: PayChannel
  amountCent: number
  refundableCent?: number
  memberPhoneMasked?: string
  memberNickname?: string
  memberPhone?: string
  memberAvatarUrl?: string
  storeId?: number | string
  storeName?: string
  paymentOrderId?: number | string
  refundStatus?: string
}

export interface PaymentOrder extends AuditFields {
  id: number | string
  paymentOrderNo: string
  businessOrderId: number | string
  businessOrderNo: string
  orderType: string
  businessStatus?: string
  paymentStatus?: PaymentStatus
  amountCent: number
  payMethod?: PayChannel
  status: string
  paidAt?: string
}

export interface PaymentTransactionRecord extends AuditFields {
  id: number | string
  paymentOrderNo: string
  businessOrderId: number | string
  businessOrderNo: string
  orderType: string
  amountCent: number
  payMethod?: PayChannel
  status: string
  paidAt?: string
}

export interface RefundOrder extends AuditFields {
  id: number | string
  refundOrderNo: string
  paymentOrderId: number | string
  businessOrderId: number | string
  amountCent: number
  channel?: PayChannel
  status: RefundStatus
  reason?: string
}

export interface FoodOrder extends AuditFields {
  id: number | string
  businessOrderId: string
  status: FoodOrderStatus
  paymentStatus: PaymentStatus
  payChannel?: PayChannel
  amountCent: number
  paidAmountCent: number
  paymentOrderId: number | string
  pointsEarned: number
  tableName?: string
  itemsSummary?: string
  items: Array<{
    id: number | string
    name: string
    unitPriceCent: number
    quantity: number
    subtotalCent: number
    pointsReward: number
    assetId?: number | string
    imageUrl?: string
  }>
  memberNickname?: string
  memberPhone?: string
  memberAvatarUrl?: string
}

export interface CatalogItem extends AuditFields {
  id: number | string
  name: string
  scopeType: ScopeType
  sourceItemId?: number | string
  isInherited?: boolean
  overrideFields?: string[]
  priceCent: number
  stockQuantity: number
  payChannels: PayChannel[]
  couponTemplateIds?: Array<number | string>
  status: PublishStatus
  categoryName?: string
  categoryId?: number | string
  description?: string
  assetId?: number | string
  imageUrl?: string
  itemType?: string
  pointsReward?: number
  sortOrder?: number
}

export interface CatalogCategory extends AuditFields {
  id: number | string
  name: string
  scopeType: ScopeType
  sortOrder?: number
  status?: string
  parentId?: number | string | null
  assetId?: number | string | null
  imageUrl?: string
}

export interface StoreActivity extends AuditFields {
  id: number | string
  title: string
  scopeType: ScopeType
  status: PublishStatus
  startAt?: string
  endAt?: string
  soldCount?: number
  verifiedCount?: number
  imageUrl?: string
  description?: string
  content?: string
  assetId?: number | string
  payChannels?: PayChannel[]
  purchaseLimitPerMember?: number
}

export interface ActivityTicketType extends AuditFields {
  id: number | string
  activityId: number | string
  name: string
  priceCent: number
  stockQuantity: number
  soldQuantity?: number
  saleStartAt?: string
  saleEndAt?: string
  payChannels: PayChannel[]
  maxTicketsPerOrder: number
  status: string
}

export interface Ticket extends AuditFields {
  id: number | string
  code: string
  activityTitle?: string
  ticketTypeName?: string
  status: VerificationStatus
  memberNickname?: string
  memberPhoneMasked?: string
}

export interface VerificationRecord extends AuditFields {
  id: number | string
  kind: VerificationKind
  refId?: number | string
  code?: string
  activityTitle?: string
  memberName?: string
  status: VerificationStatus
  result?: string
  verifiedBy?: number | string
  verifiedAt?: string
  at?: string
}

export interface PointSavingReviewer {
  type: 'staff' | 'store_admin' | 'cashier'
  id: number
  role?: string
  username?: string
  displayName?: string
  staffName?: string
  nickname?: string
  phone?: string
  avatarUrl?: string
  source?: string
}

// 对齐 server activity.PointSavingView：会员资料、审核者快照及审核时间。
export interface PointSavingRequest extends AuditFields {
  id: number | string
  memberId: number | string
  memberName?: string
  phone?: string
  memberAvatarUrl?: string
  direction?: string
  points: number
  basePoints: number
  excessPoints: number
  awardedPoints: number
  coinBasePoints: number
  awardedCoins: number
  ruleVersion: number
  pointsDivisor: number
  coinPointsDivisor: number
  calculationDescription?: string
  status: ReviewStatus
  note?: string
  reviewedBy?: number
  reviewedByType?: string
  reviewer?: PointSavingReviewer
  reviewedAt?: string
  submittedAt?: string
}

export interface Reservation extends AuditFields {
  id: number | string
  reservationNo?: string
  memberNickname?: string
  memberPhone?: string
  memberAvatarUrl?: string
  tableId?: number | string
  seatId?: number | string
  tableNo?: string
  seatNo?: string
  partySize?: number
  reservedAt?: string
  status: ReservationStatus
  remark?: string
}

export interface StoreTable extends AuditFields {
  id: number | string
  storeId: number | string
  storeName?: string
  name: string
  code: string
  capacity: number
  seatCount: number
  basePoints: number
  status: string
}

export interface StoreSeat extends AuditFields {
  id: number | string
  storeId: number | string
  storeName?: string
  tableId: number | string
  tableName: string
  name: string
  status: string
}

export interface CouponTemplate extends AuditFields {
  id: number | string
  storeId?: number | string
  scopeType: 'store'
  name: string
  description?: string
  couponType: string
  status: string
}

export interface CollectionOrder extends AuditFields {
  id: number | string
  collectionOrderNo: string
  amountCent: number
  subject: string
  businessType?: string
  payChannel?: PayChannel
  status: CollectionOrderStatus
  qrContent?: string
  qrDisplayUrl?: string
  expiresAt?: string
  memberNickname?: string
  memberPhoneMasked?: string
}

export interface PrinterDevice extends AuditFields {
  id: number | string
  storeId?: number | string
  name: string
  provider?: string
  deviceSn?: string
  deviceKey?: string
  status: ActiveStatus
}

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

// 对齐 server reporting.OverviewView（GET /store/reports/overview，无入参）。
export interface ReportOverview {
  storeCount: number
  memberCount: number
  orderCount: number
  grossSalesCent: number
  offlineCollectionRevenueCent: number
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

/** 门店收款趋势的单日汇总，金额单位为分。 */
export interface RevenueReportRow {
  date: string
  orderCount: number
  grossCent: number
}

export interface RevenueReportQuery {
  from: string
  to: string
  page: number
  pageSize: number
}

/** 会员单一资产余额（仅会员详情返回 wallet 数组）。 */
export interface WalletAccount {
  assetType: string
  availableAmount: number
  heldAmount: number
}

export interface Member {
  id: number | string
  nickname?: string
  phone?: string
  avatarUrl?: string
  pointsBalance?: number
  coinsBalance?: number
  vipTierName?: string
  vipLevel?: number
  status?: string
  createdAt?: string
  /** 仅会员详情返回：各资产余额。 */
  wallet?: WalletAccount[]
}

export interface WalletLedgerEntry {
  id: number | string
  recordKey: string
  memberId: number | string
  memberNickname?: string
  memberPhone?: string
  memberAvatarUrl?: string
  storeId?: number | string
  storeName?: string
  assetType: string
  direction: 'credit' | 'debit'
  amount: number
  balanceAfter?: number
  status: string
  reason?: string
  sourceType?: string
  sourceId?: number | string
  relatedOrderNo?: string
  createdAt?: string
}

export interface Cashier extends AuditFields {
  id: number | string
  username?: string
  displayName: string
  role?: string
  storeId?: number | string
  storeName?: string
  status: ActiveStatus
  /** 仅新增/重置密码响应返回：一次性明文初始密码。 */
  initialPassword?: string
}

export interface StaffAccount extends AuditFields {
  id: number | string
  /** 绑定的小程序会员 id */
  memberId?: number | string
  name: string
  /** 绑定会员的头像、昵称与完整手机号 */
  avatarUrl?: string
  nickname?: string
  phone?: string
  storeId?: number | string
  storeName?: string
  status: ActiveStatus
}
