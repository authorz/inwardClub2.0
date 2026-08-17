/**
 * 集中定义的业务枚举、状态机常量与展示映射。
 *
 * 设计规则要求：状态枚举、支付方式、订单类型、资产类型、核销状态必须集中定义，
 * 不允许在页面里散落字符串。所有 StatusTag / 筛选器 / 表格都从这里取标签与色调。
 */

/** StatusTag 可用的语义色调（映射到 Naive UI n-tag type）。 */
export type StatusTone = 'default' | 'success' | 'warning' | 'error' | 'info'

export interface EnumOption<T extends string = string> {
  value: T
  label: string
  tone?: StatusTone
}

/** 把枚举字典转换成筛选器/下拉可用的 options 列表。 */
export function toOptions<T extends string>(dict: Record<T, EnumOption<T>>): EnumOption<T>[] {
  return Object.values(dict) as EnumOption<T>[]
}

/** 根据字典取展示项，未知值回退为原字符串（避免界面出现空白）。 */
export function resolveEnum<T extends string>(
  dict: Record<string, EnumOption>,
  value: T | null | undefined,
): EnumOption {
  if (value == null || value === '') return { value: '', label: '-', tone: 'default' }
  return dict[value] ?? { value, label: value, tone: 'default' }
}

/* ------------------------------------------------------------------ */
/* 订单类型                                                            */
/* ------------------------------------------------------------------ */

export type OrderType = 'food' | 'activity' | 'recharge' | 'coupon' | 'offline_collection'

export const ORDER_TYPE: Record<OrderType, EnumOption<OrderType>> = {
  food: { value: 'food', label: '点餐', tone: 'default' },
  activity: { value: 'activity', label: '活动', tone: 'info' },
  recharge: { value: 'recharge', label: '充值', tone: 'default' },
  coupon: { value: 'coupon', label: '券兑换', tone: 'default' },
  offline_collection: { value: 'offline_collection', label: '线下收款', tone: 'default' },
}

/* ------------------------------------------------------------------ */
/* 支付渠道                                                            */
/* ------------------------------------------------------------------ */

export type PayChannel = 'wechat' | 'alipay' | 'coin' | 'offline'

export const PAY_CHANNEL: Record<PayChannel, EnumOption<PayChannel>> = {
  wechat: { value: 'wechat', label: '微信', tone: 'success' },
  alipay: { value: 'alipay', label: '支付宝', tone: 'info' },
  coin: { value: 'coin', label: '金币', tone: 'warning' },
  offline: { value: 'offline', label: '线下收款', tone: 'default' },
}

/** 线下收款当前使用已接入的微信支付 Native 渠道。 */
export const COLLECTION_PAY_CHANNELS: PayChannel[] = ['wechat']

/* ------------------------------------------------------------------ */
/* 支付状态                                                            */
/* ------------------------------------------------------------------ */

export type PaymentStatus = 'unpaid' | 'paid' | 'refunding' | 'refunded' | 'partially_refunded' | 'closed' | 'failed'

export const PAYMENT_STATUS: Record<PaymentStatus, EnumOption<PaymentStatus>> = {
  unpaid: { value: 'unpaid', label: '未支付', tone: 'warning' },
  paid: { value: 'paid', label: '已支付', tone: 'success' },
  refunding: { value: 'refunding', label: '退款中', tone: 'info' },
  refunded: { value: 'refunded', label: '已退款', tone: 'default' },
  partially_refunded: { value: 'partially_refunded', label: '部分退款', tone: 'info' },
  closed: { value: 'closed', label: '已关闭', tone: 'default' },
  failed: { value: 'failed', label: '支付失败', tone: 'error' },
}

/** 统一订单读模型的支付状态（与总后台订单中心一致）。 */
export const ORDER_PAYMENT_STATUS: Record<string, EnumOption> = {
  unpaid: { value: 'unpaid', label: '未支付', tone: 'warning' },
  paid: { value: 'paid', label: '已支付', tone: 'success' },
  expired: { value: 'expired', label: '已过期', tone: 'default' },
  partially_refunded: { value: 'partially_refunded', label: '部分退款', tone: 'warning' },
  refunded: { value: 'refunded', label: '已退款', tone: 'info' },
}

/** 统一订单读模型的履约状态（与总后台订单中心一致）。 */
export const ORDER_STATUS: Record<string, EnumOption> = {
  created: { value: 'created', label: '已创建', tone: 'default' },
  confirmed: { value: 'confirmed', label: '已确认', tone: 'info' },
  preparing: { value: 'preparing', label: '备餐中', tone: 'warning' },
  ready: { value: 'ready', label: '待取', tone: 'info' },
  completed: { value: 'completed', label: '已完成', tone: 'success' },
  cancelled: { value: 'cancelled', label: '已取消', tone: 'error' },
  partially_refunded: { value: 'partially_refunded', label: '部分退款', tone: 'warning' },
  refunded: { value: 'refunded', label: '已退款', tone: 'info' },
}

/* ------------------------------------------------------------------ */
/* 退款单状态                                                          */
/* ------------------------------------------------------------------ */

export type RefundStatus = 'pending' | 'success' | 'failed'

export const REFUND_STATUS: Record<RefundStatus, EnumOption<RefundStatus>> = {
  pending: { value: 'pending', label: '处理中', tone: 'warning' },
  success: { value: 'success', label: '已退款', tone: 'success' },
  failed: { value: 'failed', label: '退款失败', tone: 'error' },
}

/* ------------------------------------------------------------------ */
/* 点餐订单履约状态机                                                  */
/* ------------------------------------------------------------------ */

export type FoodOrderStatus =
  | 'pending'
  | 'confirmed'
  | 'preparing'
  | 'ready'
  | 'completed'
  | 'cancelled'

export const FOOD_ORDER_STATUS: Record<FoodOrderStatus, EnumOption<FoodOrderStatus>> = {
  pending: { value: 'pending', label: '待确认', tone: 'warning' },
  confirmed: { value: 'confirmed', label: '已确认', tone: 'info' },
  preparing: { value: 'preparing', label: '备餐中', tone: 'info' },
  ready: { value: 'ready', label: '待取餐', tone: 'success' },
  completed: { value: 'completed', label: '已完成', tone: 'default' },
  cancelled: { value: 'cancelled', label: '已取消', tone: 'error' },
}

/**
 * 点餐订单状态机可执行动作。集中定义避免每个页面各写一套流转判断。
 * key 为当前状态，value 为可推进的动作。
 */
export interface FoodOrderAction {
  action: 'confirm' | 'prepare' | 'ready' | 'complete' | 'cancel'
  label: string
  next: FoodOrderStatus
  danger?: boolean
}

export const FOOD_ORDER_TRANSITIONS: Record<FoodOrderStatus, FoodOrderAction[]> = {
  pending: [
    { action: 'confirm', label: '确认接单', next: 'confirmed' },
    { action: 'cancel', label: '取消订单', next: 'cancelled', danger: true },
  ],
  confirmed: [
    { action: 'prepare', label: '开始备餐', next: 'preparing' },
    { action: 'cancel', label: '取消订单', next: 'cancelled', danger: true },
  ],
  preparing: [{ action: 'ready', label: '标记待取', next: 'ready' }],
  ready: [{ action: 'complete', label: '完成订单', next: 'completed' }],
  completed: [],
  cancelled: [],
}

/* ------------------------------------------------------------------ */
/* 核销状态（票/券共用）                                               */
/* ------------------------------------------------------------------ */

export type VerificationStatus = 'valid' | 'used' | 'expired' | 'void'

export const VERIFICATION_STATUS: Record<VerificationStatus, EnumOption<VerificationStatus>> = {
  valid: { value: 'valid', label: '未核销', tone: 'success' },
  used: { value: 'used', label: '已核销', tone: 'default' },
  expired: { value: 'expired', label: '已过期', tone: 'warning' },
  void: { value: 'void', label: '已作废', tone: 'error' },
}

/** 核销业务类型：活动票 / 券。 */
export type VerificationKind = 'ticket' | 'coupon'

export const VERIFICATION_KIND: Record<VerificationKind, EnumOption<VerificationKind>> = {
  ticket: { value: 'ticket', label: '活动票', tone: 'info' },
  coupon: { value: 'coupon', label: '券', tone: 'default' },
}

/* ------------------------------------------------------------------ */
/* 审核状态（积分存入审核等审批流）                                    */
/* ------------------------------------------------------------------ */

export type ReviewStatus = 'pending' | 'approved' | 'rejected'

export const REVIEW_STATUS: Record<ReviewStatus, EnumOption<ReviewStatus>> = {
  pending: { value: 'pending', label: '待审核', tone: 'warning' },
  approved: { value: 'approved', label: '已通过', tone: 'success' },
  rejected: { value: 'rejected', label: '已驳回', tone: 'error' },
}

/* ------------------------------------------------------------------ */
/* 通用上下架 / 启用状态                                               */
/* ------------------------------------------------------------------ */

export type PublishStatus = 'draft' | 'published' | 'unpublished' | 'archived'

export const PUBLISH_STATUS: Record<PublishStatus, EnumOption<PublishStatus>> = {
  draft: { value: 'draft', label: '草稿', tone: 'default' },
  published: { value: 'published', label: '已上架', tone: 'success' },
  unpublished: { value: 'unpublished', label: '已下架', tone: 'warning' },
  archived: { value: 'archived', label: '已归档', tone: 'default' },
}

export type AvailabilityStatus = 'available' | 'paused' | 'full' | 'maintenance'
export const AVAILABILITY_STATUS: Record<AvailabilityStatus, EnumOption<AvailabilityStatus>> = {
  available: { value: 'available', label: '可预约', tone: 'success' },
  paused: { value: 'paused', label: '暂停预约', tone: 'warning' },
  full: { value: 'full', label: '座位已满', tone: 'info' },
  maintenance: { value: 'maintenance', label: '维护中', tone: 'error' },
}

export type ActiveStatus = 'active' | 'disabled'

export const ACTIVE_STATUS: Record<ActiveStatus, EnumOption<ActiveStatus>> = {
  active: { value: 'active', label: '启用', tone: 'success' },
  disabled: { value: 'disabled', label: '停用', tone: 'default' },
}

/* ------------------------------------------------------------------ */
/* 商品来源（本店自建 / 采用全局）                                     */
/* ------------------------------------------------------------------ */

export type ScopeType = 'global' | 'store'

export const SCOPE_TYPE: Record<ScopeType, EnumOption<ScopeType>> = {
  global: { value: 'global', label: '全局商品', tone: 'info' },
  store: { value: 'store', label: '本店自建', tone: 'default' },
}

/* ------------------------------------------------------------------ */
/* 预约状态机                                                          */
/* ------------------------------------------------------------------ */

// 状态值对齐服务端 reservation.model（booked/arrived/cancelled/expired）。
export type ReservationStatus = 'booked' | 'arrived'

export const RESERVATION_STATUS: Record<ReservationStatus, EnumOption<ReservationStatus>> = {
  booked: { value: 'booked', label: '已预定', tone: 'warning' },
  arrived: { value: 'arrived', label: '已到店', tone: 'success' },
}

/* ------------------------------------------------------------------ */
/* 线下聚合收款单状态                                                  */
/* ------------------------------------------------------------------ */

export type CollectionOrderStatus = 'pending' | 'paid' | 'cancelled' | 'expired'

export const COLLECTION_ORDER_STATUS: Record<
  CollectionOrderStatus,
  EnumOption<CollectionOrderStatus>
> = {
  pending: { value: 'pending', label: '待支付', tone: 'warning' },
  paid: { value: 'paid', label: '已收款', tone: 'success' },
  cancelled: { value: 'cancelled', label: '已取消', tone: 'default' },
  expired: { value: 'expired', label: '已过期', tone: 'default' },
}

/* ------------------------------------------------------------------ */
/* 打印机 / 打印任务状态                                               */
/* ------------------------------------------------------------------ */

export type PrinterStatus = 'online' | 'offline' | 'error' | 'disabled'

export const PRINTER_STATUS: Record<PrinterStatus, EnumOption<PrinterStatus>> = {
  online: { value: 'online', label: '在线', tone: 'success' },
  offline: { value: 'offline', label: '离线', tone: 'default' },
  error: { value: 'error', label: '异常', tone: 'error' },
  disabled: { value: 'disabled', label: '已停用', tone: 'default' },
}

export type PrintJobStatus = 'queued' | 'printing' | 'succeeded' | 'failed'

export const PRINT_JOB_STATUS: Record<PrintJobStatus, EnumOption<PrintJobStatus>> = {
  queued: { value: 'queued', label: '排队中', tone: 'warning' },
  printing: { value: 'printing', label: '打印中', tone: 'info' },
  succeeded: { value: 'succeeded', label: '成功', tone: 'success' },
  failed: { value: 'failed', label: '失败', tone: 'error' },
}

/* ------------------------------------------------------------------ */
/* 资产上传 purpose（门店后台可用范围）                                */
/* ------------------------------------------------------------------ */

export type AssetPurpose =
  | 'store_logo'
  | 'banner'
  | 'category'
  | 'product'
  | 'activity'
  | 'table_layout'
  | 'seat_layout'
  | 'rich_content'

export const STORE_ASSET_PURPOSES: AssetPurpose[] = [
  'store_logo',
  'banner',
  'category',
  'product',
  'activity',
  'table_layout',
  'seat_layout',
  'rich_content',
]
