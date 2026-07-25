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
  businessOrderId: string
  orderType: string
  paymentStatus: PaymentStatus
  orderStatus?: string
  payChannel?: PayChannel
  amountCent: number
  refundableCent?: number
  memberPhoneMasked?: string
  memberNickname?: string
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
  tableName?: string
  itemsSummary?: string
  memberNickname?: string
  memberPhoneMasked?: string
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
  status: PublishStatus
  categoryName?: string
  assetUrl?: string
}

export interface CatalogCategory extends AuditFields {
  id: number | string
  name: string
  scopeType: ScopeType
  sortOrder?: number
  status?: string
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
  assetUrl?: string
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

// 对齐 server activity.PointSavingView：memberName/phone/points/note/reviewedBy。
export interface PointSavingRequest extends AuditFields {
  id: number | string
  memberName?: string
  phone?: string
  direction?: string
  points: number
  status: ReviewStatus
  note?: string
  reviewedBy?: number
  submittedAt?: string
}

export interface Reservation extends AuditFields {
  id: number | string
  reservationNo?: string
  // memberNickname/memberPhoneMasked/tableName 依赖服务端 ReservationView 关联 members/tables，
  // 当前 /store/reservations 读模型未提供（见 blocked API gaps），到位前这些列为空。
  memberNickname?: string
  memberPhoneMasked?: string
  tableName?: string
  partySize?: number
  reservedAt?: string
  status: ReservationStatus
  remark?: string
}

export interface StoreTable extends AuditFields {
  id: number | string
  name: string
  seatCount: number
  area?: string
  status?: string
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

// 对齐 server reporting.OverviewView（GET /store/reports/overview，无入参）。
export interface ReportOverview {
  storeCount: number
  memberCount: number
  orderCount: number
  grossSalesCent: number
  couponsIssued: number
  couponsRedeemed: number
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
  pointsBalance?: number
  status?: string
  createdAt?: string
  /** 仅会员详情返回：各资产余额。 */
  wallet?: WalletAccount[]
}

export interface WalletLedgerEntry {
  id: number | string
  memberId: number | string
  assetType: string
  direction: 'credit' | 'debit'
  amount: number
  balanceAfter: number
  reason?: string
  sourceType?: string
  sourceId?: number | string
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
  /** 会员手机号（打码） */
  phone?: string
  storeId?: number | string
  storeName?: string
  status: ActiveStatus
}
