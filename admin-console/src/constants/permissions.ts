/**
 * 总后台权限码（集中定义）。
 *
 * 依据 docs/ADMIN_CONSOLE_ARCHITECTURE.md 第 5 节权限模型。
 * 前端仅用权限码做「菜单/路由/按钮」显隐；最终权限必须由服务端强制。
 *
 * 权限分层：
 *  1) 页面权限：菜单和路由（meta.permission）。
 *  2) 操作权限：新增/编辑/删除/发布/退款/调账/导出（PermissionButton / v-permission）。
 *  3) 数据权限：由服务端强制。
 */
export const PERMISSIONS = {
  STORE_READ: 'admin.store.read',
  STORE_WRITE: 'admin.store.write',

  ACCOUNT_READ: 'admin.account.read',
  ACCOUNT_WRITE: 'admin.account.write',

  CATALOG_READ: 'admin.catalog.read',
  CATALOG_GLOBAL_WRITE: 'admin.catalog.global.write',
  CATALOG_STORE_OVERRIDE_WRITE: 'admin.catalog.store_override.write',

  ACTIVITY_READ: 'admin.activity.read',
  ACTIVITY_GLOBAL_WRITE: 'admin.activity.global.write',

  COUPON_READ: 'admin.coupon.read',
  COUPON_GLOBAL_WRITE: 'admin.coupon.global.write',

  BANNER_READ: 'admin.banner.read',
  BANNER_WRITE: 'admin.banner.write',

  RECHARGE_READ: 'admin.recharge.read',
  RECHARGE_WRITE: 'admin.recharge.write',

  RULE_READ: 'admin.rule.read',
  RULE_PUBLISH: 'admin.rule.publish',

  ORDER_READ: 'admin.order.read',

  PAYMENT_READ: 'admin.payment.read',
  REFUND_APPROVE: 'admin.refund.approve',

  MEMBER_READ: 'admin.member.read',
  MEMBER_WALLET_ADJUST: 'admin.member.wallet_adjust',

  REPORT_READ: 'admin.report.read',

  AUDIT_READ: 'admin.audit.read',

  SYSTEM_PAYMENT_SETTINGS_WRITE: 'admin.system.payment_settings.write',
} as const

export type PermissionCode = (typeof PERMISSIONS)[keyof typeof PERMISSIONS]
