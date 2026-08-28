/**
 * 集中定义的业务枚举与状态常量。
 *
 * 设计规则要求：状态枚举、支付方式、订单类型、资产类型、核销状态必须集中定义，
 * 不允许在页面里散落字符串。所有枚举同时导出「值常量」与「展示映射」，
 * 供 StatusTag、筛选下拉、表格列复用。
 *
 * 注意：这些是「展示层」枚举，最终业务状态机以服务端为准，前端不得据此计算关键状态。
 */

export interface OptionItem<T = string> {
  label: string
  value: T
  /** StatusTag 使用的语义类型；不依赖颜色单独表达状态 */
  tone?: 'default' | 'success' | 'warning' | 'error' | 'info'
}

/** 通用启用/禁用状态（账号、模板、门店等） */
export const RESOURCE_STATUS = {
  ACTIVE: 'active',
  DISABLED: 'disabled',
  DRAFT: 'draft',
  PUBLISHED: 'published',
  ARCHIVED: 'archived',
} as const
export type ResourceStatus = (typeof RESOURCE_STATUS)[keyof typeof RESOURCE_STATUS]

export const RESOURCE_STATUS_OPTIONS: OptionItem[] = [
  { label: '草稿', value: RESOURCE_STATUS.DRAFT, tone: 'default' },
  { label: '已发布', value: RESOURCE_STATUS.PUBLISHED, tone: 'success' },
  { label: '启用', value: RESOURCE_STATUS.ACTIVE, tone: 'success' },
  { label: '已禁用', value: RESOURCE_STATUS.DISABLED, tone: 'error' },
  { label: '已归档', value: RESOURCE_STATUS.ARCHIVED, tone: 'default' },
]

/** 门店营业状态（后端取值 active/inactive，不同于通用启用/禁用） */
export const STORE_STATUS_OPTIONS: OptionItem[] = [
  { label: '营业中', value: 'active', tone: 'success' },
  { label: '休息中', value: 'inactive', tone: 'default' },
]

/** 云打印机启用状态。 */
export const PRINTER_STATUS_OPTIONS: OptionItem[] = [
  { label: '启用', value: 'active', tone: 'success' },
  { label: '已停用', value: 'disabled', tone: 'error' },
]

/** Banner 展示状态（后端仅在 active 时向小程序返回）。 */
export const BANNER_STATUS_OPTIONS: OptionItem[] = [
  { label: '显示', value: 'active', tone: 'success' },
  { label: '不显示', value: 'inactive', tone: 'default' },
]

/** 桌位预约状态（桌子和座位共用）。 */
export const TABLE_SEAT_STATUS_OPTIONS: OptionItem[] = [
  { label: '可预约', value: 'available', tone: 'success' },
  { label: '暂停预约', value: 'paused', tone: 'warning' },
  { label: '座位已满', value: 'full', tone: 'error' },
  { label: '维护中', value: 'maintenance', tone: 'default' },
]

/** 加盟咨询处理状态。 */
export const FRANCHISE_INQUIRY_STATUS_OPTIONS: OptionItem[] = [
  { label: '未处理', value: 'unprocessed', tone: 'warning' },
  { label: '已处理', value: 'processed', tone: 'success' },
]

/** 门店状态值 → 中文标签；未知值原样返回。 */
export function storeStatusLabel(status?: string): string {
  return STORE_STATUS_OPTIONS.find((o) => o.value === status)?.label ?? status ?? ''
}

/** 资源归属范围：全局模板 vs 门店自有 */
export const SCOPE_TYPE = {
  GLOBAL: 'global',
  STORE: 'store',
} as const
export type ScopeType = (typeof SCOPE_TYPE)[keyof typeof SCOPE_TYPE]

export const SCOPE_TYPE_OPTIONS: OptionItem[] = [
  { label: '全局', value: SCOPE_TYPE.GLOBAL, tone: 'info' },
  { label: '门店', value: SCOPE_TYPE.STORE, tone: 'default' },
]

/** 用户支付方式：仅支持微信和金币。 */
export const PAY_CHANNEL = {
  WECHAT: 'wechat',
  COIN: 'coin',
} as const
export type PayChannel = (typeof PAY_CHANNEL)[keyof typeof PAY_CHANNEL]

export const PAY_CHANNEL_OPTIONS: OptionItem[] = [
  { label: '微信', value: PAY_CHANNEL.WECHAT, tone: 'success' },
  { label: '金币', value: PAY_CHANNEL.COIN, tone: 'warning' },
]

/** 订单列表还会返回优惠券兑换记录，仅用于订单筛选和展示。 */
export const ORDER_PAY_CHANNEL_OPTIONS: OptionItem[] = [
  ...PAY_CHANNEL_OPTIONS,
  { label: '优惠券', value: 'coupon', tone: 'default' },
]

/** 订单类型 */
export const ORDER_TYPE = {
  FOOD: 'food',
  ACTIVITY: 'activity',
  RECHARGE: 'recharge',
  COUPON: 'coupon',
  OFFLINE_COLLECTION: 'offline_collection',
} as const
export type OrderType = (typeof ORDER_TYPE)[keyof typeof ORDER_TYPE]

export const ORDER_TYPE_OPTIONS: OptionItem[] = [
  { label: '点餐', value: ORDER_TYPE.FOOD, tone: 'default' },
  { label: '活动', value: ORDER_TYPE.ACTIVITY, tone: 'default' },
  { label: '充值', value: ORDER_TYPE.RECHARGE, tone: 'default' },
  { label: '券兑换', value: ORDER_TYPE.COUPON, tone: 'default' },
  { label: '线下聚合收款', value: ORDER_TYPE.OFFLINE_COLLECTION, tone: 'default' },
]

/** 支付状态 */
export const PAYMENT_STATUS = {
  PENDING: 'pending',
  PAID: 'paid',
  FAILED: 'failed',
  REFUNDING: 'refunding',
  PARTIALLY_REFUNDED: 'partially_refunded',
  REFUNDED: 'refunded',
  CLOSED: 'closed',
} as const
export type PaymentStatus = (typeof PAYMENT_STATUS)[keyof typeof PAYMENT_STATUS]

export const PAYMENT_STATUS_OPTIONS: OptionItem[] = [
  { label: '待支付', value: PAYMENT_STATUS.PENDING, tone: 'warning' },
  { label: '已支付', value: PAYMENT_STATUS.PAID, tone: 'success' },
  { label: '支付失败', value: PAYMENT_STATUS.FAILED, tone: 'error' },
  { label: '退款中', value: PAYMENT_STATUS.REFUNDING, tone: 'warning' },
  { label: '部分退款', value: PAYMENT_STATUS.PARTIALLY_REFUNDED, tone: 'warning' },
  { label: '已退款', value: PAYMENT_STATUS.REFUNDED, tone: 'info' },
  { label: '已关闭', value: PAYMENT_STATUS.CLOSED, tone: 'default' },
]

/** 履约/订单状态（展示用；真实状态机在服务端） */
export const ORDER_STATUS = {
  CREATED: 'created',
  CONFIRMED: 'confirmed',
  PREPARING: 'preparing',
  READY: 'ready',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
} as const
export type OrderStatus = (typeof ORDER_STATUS)[keyof typeof ORDER_STATUS]

export const ORDER_STATUS_OPTIONS: OptionItem[] = [
  { label: '已创建', value: ORDER_STATUS.CREATED, tone: 'default' },
  { label: '已确认', value: ORDER_STATUS.CONFIRMED, tone: 'info' },
  { label: '备餐中', value: ORDER_STATUS.PREPARING, tone: 'warning' },
  { label: '待取', value: ORDER_STATUS.READY, tone: 'info' },
  { label: '已完成', value: ORDER_STATUS.COMPLETED, tone: 'success' },
  { label: '已取消', value: ORDER_STATUS.CANCELLED, tone: 'error' },
  { label: '部分退款', value: 'partially_refunded', tone: 'warning' },
  { label: '已退款', value: 'refunded', tone: 'info' },
]

/** 退款单状态 */
export const REFUND_STATUS = {
  SUCCEEDED: 'succeeded',
  FAILED: 'failed',
} as const
export type RefundStatus = (typeof REFUND_STATUS)[keyof typeof REFUND_STATUS]

export const REFUND_STATUS_OPTIONS: OptionItem[] = [
  { label: '退款成功', value: REFUND_STATUS.SUCCEEDED, tone: 'success' },
  { label: '退款失败', value: REFUND_STATUS.FAILED, tone: 'error' },
]

/** 商品/资产类型 */
export const ITEM_TYPE = {
  FOOD: 'food',
  COUPON: 'coupon',
  REDEEMABLE: 'redeemable',
  PHYSICAL: 'physical',
} as const
export type ItemType = (typeof ITEM_TYPE)[keyof typeof ITEM_TYPE]

export const ITEM_TYPE_OPTIONS: OptionItem[] = [
  { label: '餐品', value: ITEM_TYPE.FOOD, tone: 'default' },
  { label: '券商品', value: ITEM_TYPE.COUPON, tone: 'default' },
  { label: '积分兑换', value: ITEM_TYPE.REDEEMABLE, tone: 'default' },
  { label: '实物', value: ITEM_TYPE.PHYSICAL, tone: 'default' },
]

/** 钱包资产类型（分账户） */
export const ASSET_TYPE = {
  POINTS: 'points',
  COINS: 'coins',
  CASH_BALANCE: 'cash_balance',
  GROWTH_VALUE: 'growth_value',
} as const
export type AssetType = (typeof ASSET_TYPE)[keyof typeof ASSET_TYPE]

export const ASSET_TYPE_OPTIONS: OptionItem[] = [
  { label: '积分', value: ASSET_TYPE.POINTS, tone: 'default' },
  { label: '金币', value: ASSET_TYPE.COINS, tone: 'warning' },
  { label: '余额', value: ASSET_TYPE.CASH_BALANCE, tone: 'info' },
  { label: '成长值', value: ASSET_TYPE.GROWTH_VALUE, tone: 'default' },
]

/** 核销状态 */
export const VERIFICATION_STATUS = {
  UNUSED: 'unused',
  USED: 'used',
  EXPIRED: 'expired',
  VOID: 'void',
} as const
export type VerificationStatus = (typeof VERIFICATION_STATUS)[keyof typeof VERIFICATION_STATUS]

export const VERIFICATION_STATUS_OPTIONS: OptionItem[] = [
  { label: '未核销', value: VERIFICATION_STATUS.UNUSED, tone: 'info' },
  { label: '已核销', value: VERIFICATION_STATUS.USED, tone: 'success' },
  { label: '已过期', value: VERIFICATION_STATUS.EXPIRED, tone: 'default' },
  { label: '已作废', value: VERIFICATION_STATUS.VOID, tone: 'error' },
]

/** 用户持有优惠券状态。 */
export const COUPON_ENTITLEMENT_STATUS_OPTIONS: OptionItem[] = [
  { label: '可使用', value: 'active', tone: 'success' },
  { label: '已使用', value: 'used', tone: 'default' },
  { label: '已过期', value: 'expired', tone: 'warning' },
  { label: '已删除', value: 'void', tone: 'error' },
]

/**
 * 从任意 options 列表构造 value -> OptionItem 的查表，供 StatusTag / 展示复用。
 */
export function buildOptionMap<T extends string>(
  options: OptionItem<T>[],
): Record<string, OptionItem<T>> {
  return options.reduce<Record<string, OptionItem<T>>>((acc, opt) => {
    acc[opt.value] = opt
    return acc
  }, {})
}
