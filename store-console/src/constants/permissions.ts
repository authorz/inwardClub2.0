/**
 * 门店后台权限码（与总后台权限码完全独立）。
 *
 * 权限最终由服务端强制；前端仅用于隐藏无权入口和禁用按钮，属于体验层收敛。
 * 门店后台不得访问任何总后台权限码。
 */

export const PERM = {
  profileRead: 'store.profile.read',
  profileWrite: 'store.profile.write',
  catalogRead: 'store.catalog.read',
  catalogWrite: 'store.catalog.write',
  activityRead: 'store.activity.read',
  activityWrite: 'store.activity.write',
  couponRead: 'store.coupon.read',
  couponWrite: 'store.coupon.write',
  orderRead: 'store.order.read',
  orderStatusWrite: 'store.order.status_write',
  collectionCreate: 'store.collection.create',
  collectionRead: 'store.collection.read',
  refundRequest: 'store.refund.request',
  memberRead: 'store.member.read',
  memberReadLimited: 'store.member.read_limited',
  memberWalletAdjustRequest: 'store.member.wallet_adjust_request',
  reservationWrite: 'store.reservation.write',
  ticketVerify: 'store.ticket.verify',
  pointReview: 'store.point.review',
  staffWrite: 'store.staff.write',
  printerWrite: 'store.printer.write',
  reportRead: 'store.report.read',
  auditRead: 'store.audit.read',
} as const

export type PermissionCode = (typeof PERM)[keyof typeof PERM]

/** 门店后台角色。 */
export type StoreRole = 'store_admin' | 'cashier' | 'store_operator'

/** token subject_type 白名单：只有这些主体类型允许进入门店后台。 */
export const ALLOWED_SUBJECT_TYPES = ['store_admin', 'cashier', 'store_operator'] as const
export type AllowedSubjectType = (typeof ALLOWED_SUBJECT_TYPES)[number]
