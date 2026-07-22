/**
 * Centralized enums, status maps and business constants.
 * Rule (GLOBAL_DESIGN_RULES §7): status enums, payment methods, order/asset
 * types and verification states are defined here — never as loose strings
 * scattered across pages.
 */

/* ---- payment methods (mini program: ONLY wechat + coin — no alipay) ---- */
const PAY_METHOD = {
  WECHAT: 'wechat',
  COIN: 'coin',
};

const PAY_METHOD_LABEL = {
  [PAY_METHOD.WECHAT]: '微信支付',
  [PAY_METHOD.COIN]: '金币支付',
};

/* ---- asset (wallet account) types ---- */
const ASSET_TYPE = {
  COIN: 'coins',
  POINT: 'points',
  COUPON: 'coupons',
  RECHARGE: 'recharge',
};

const ASSET_LABEL = {
  [ASSET_TYPE.COIN]: '金币',
  [ASSET_TYPE.POINT]: '积分',
  [ASSET_TYPE.COUPON]: '券',
  [ASSET_TYPE.RECHARGE]: '充值',
};

/* ---- order types (order center segmented control) ---- */
const ORDER_TYPE = {
  FOOD: 'food',
  ACTIVITY: 'activity',
  RECHARGE: 'recharge',
  REDEMPTION: 'coupon',
};

const ORDER_TYPE_LABEL = {
  [ORDER_TYPE.FOOD]: '点餐',
  [ORDER_TYPE.ACTIVITY]: '活动',
  [ORDER_TYPE.RECHARGE]: '充值',
  [ORDER_TYPE.REDEMPTION]: '兑换',
};

/* ---- generic order status → display label + tone (text, never color-only) ---- */
const ORDER_STATUS = {
  PENDING: 'pending', // 待支付
  PAID: 'paid',
  PREPARING: 'preparing', // 备餐中
  READY: 'ready',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
  REFUNDED: 'refunded',
  PENDING_VERIFY: 'pending_verify', // 待核销
  USED: 'used',
  EXPIRED: 'expired',
};

const ORDER_STATUS_LABEL = {
  pending: '待支付',
  paid: '已支付',
  preparing: '备餐中',
  ready: '待取餐',
  completed: '已完成',
  cancelled: '已取消',
  refunded: '已退款',
  pending_verify: '待核销',
  used: '已使用',
  expired: '已过期',
};

/** tone: neutral | active | done | danger  (drives icon + weak color) */
const ORDER_STATUS_TONE = {
  pending: 'active',
  paid: 'active',
  preparing: 'active',
  ready: 'active',
  completed: 'done',
  cancelled: 'neutral',
  refunded: 'danger',
  pending_verify: 'active',
  used: 'done',
  expired: 'neutral',
};

/* ---- ticket / coupon lifecycle (used by ticket-row) ---- */
const TICKET_STATUS = {
  UNUSED: 'unused', // 待使用 / 可用
  USED: 'used',
  EXPIRED: 'expired',
  REFUNDED: 'refunded',
  PENDING_VERIFY: 'pending_verify', // 待核销
};

const TICKET_STATUS_LABEL = {
  unused: '待使用',
  used: '已使用',
  expired: '已过期',
  refunded: '已退款',
  pending_verify: '待核销',
};

const COUPON_STATUS_LABEL = {
  unused: '可用',
  used: '已使用',
  expired: '已过期',
  refunded: '已退款',
};

/* ---- reservation seat states (never color-only: always paired with text) ---- */
const SEAT_STATE = {
  FREE: 'free', // 空闲
  RESERVED: 'reserved', // 已预约
  SELECTED: 'selected', // 已选择
  PLAYING: 'playing', // 游戏中
  MAINTENANCE: 'maintenance', // 维护中
};

const SEAT_STATE_LABEL = {
  free: '空闲',
  reserved: '已预约',
  selected: '已选择',
  playing: '游戏中',
  maintenance: '维护中',
};

/* ---- reservation lifecycle ---- */
const RESERVATION_STATUS_LABEL = {
  active: '进行中',
  booked: '待到店',
  pending: '待到店',
  arrived: '已到店',
  cancelled: '已取消',
  expired: '已过期',
  completed: '已完成',
};

/* ---- staff verification results ---- */
const VERIFY_RESULT = {
  SUCCESS: 'success',
  FAILED: 'failed',
  VOID: 'void',
};

const VERIFY_RESULT_LABEL = {
  success: '核销成功',
  failed: '核销失败',
  void: '退款/作废',
};

/* ---- point saving (存取积分) direction ---- */
const POINT_SAVING = {
  DEPOSIT: 'deposit', // 存入
  WITHDRAW: 'withdraw', // 取出
};

const POINT_SAVING_LABEL = {
  deposit: '存入',
  withdraw: '取出',
};

/* ---- subject types (identity) ---- */
const SUBJECT_TYPE = {
  MEMBER: 'member',
  STAFF: 'staff',
};

module.exports = {
  PAY_METHOD,
  PAY_METHOD_LABEL,
  ASSET_TYPE,
  ASSET_LABEL,
  ORDER_TYPE,
  ORDER_TYPE_LABEL,
  ORDER_STATUS,
  ORDER_STATUS_LABEL,
  ORDER_STATUS_TONE,
  TICKET_STATUS,
  TICKET_STATUS_LABEL,
  COUPON_STATUS_LABEL,
  SEAT_STATE,
  SEAT_STATE_LABEL,
  RESERVATION_STATUS_LABEL,
  VERIFY_RESULT,
  VERIFY_RESULT_LABEL,
  POINT_SAVING,
  POINT_SAVING_LABEL,
  SUBJECT_TYPE,
};
