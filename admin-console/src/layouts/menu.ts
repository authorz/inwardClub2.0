/**
 * 总后台菜单配置（单一事实来源）。
 *
 * 菜单结构依据 docs/ADMIN_CONSOLE_ARCHITECTURE.md 第 7 节。
 * 侧边菜单渲染、路由 meta（标题 + 权限）、面包屑都从这里派生，避免重复维护。
 * 每一项挂 permission，用于路由守卫与菜单显隐。
 */
import { PERMISSIONS, type PermissionCode } from '@/constants/permissions'

export interface MenuNode {
  key: string
  label: string
  /** 路由 path（叶子节点必填） */
  path?: string
  /** 页面所需权限码 */
  permission?: PermissionCode
  children?: MenuNode[]
}

export const MENU: MenuNode[] = [
  {
    key: 'dashboard',
    label: '工作台',
    path: '/dashboard',
    permission: PERMISSIONS.REPORT_READ,
  },
  {
    key: 'stores',
    label: '门店管理',
    path: '/stores',
    permission: PERMISSIONS.STORE_READ,
  },
  {
    key: 'accounts',
    label: '账号与权限',
    children: [
      {
        key: 'admin-accounts',
        label: '总后台账号',
        path: '/accounts/admins',
        permission: PERMISSIONS.ACCOUNT_READ,
      },
      {
        key: 'store-admin-accounts',
        label: '门店管理员',
        path: '/accounts/store-admins',
        permission: PERMISSIONS.ACCOUNT_READ,
      },
      {
        key: 'staff-accounts',
        label: '员工',
        path: '/accounts/staff',
        permission: PERMISSIONS.ACCOUNT_READ,
      },
    ],
  },
  {
    key: 'catalog',
    label: '商品管理',
    children: [
      {
        key: 'categories',
        label: '商品分类',
        path: '/catalog/categories',
        permission: PERMISSIONS.CATALOG_READ,
      },
      {
        key: 'items',
        label: '商品管理',
        path: '/catalog/items',
        permission: PERMISSIONS.CATALOG_READ,
      },
    ],
  },
  {
    key: 'activities',
    label: '活动管理',
    path: '/activities',
    permission: PERMISSIONS.ACTIVITY_READ,
  },
  {
    key: 'coupons',
    label: '券管理',
    path: '/coupons',
    permission: PERMISSIONS.COUPON_READ,
  },
  {
    key: 'banners',
    label: '广告管理',
    path: '/banners',
    permission: PERMISSIONS.BANNER_READ,
  },
  {
    key: 'recharge-products',
    label: '快捷充值',
    path: '/recharge-products',
    permission: PERMISSIONS.RECHARGE_READ,
  },
  {
    key: 'rules',
    label: 'VIP / 权益规则',
    children: [
      {
        key: 'membership-tiers',
        label: 'VIP 等级',
        path: '/rules/membership-tiers',
        permission: PERMISSIONS.RULE_READ,
      },
      {
        key: 'sign-in-rule',
        label: '签到规则',
        path: '/rules/sign-in',
        permission: PERMISSIONS.RULE_READ,
      },
      {
        key: 'rule-definitions',
        label: '规则中心',
        path: '/rules/definitions',
        permission: PERMISSIONS.RULE_READ,
      },
    ],
  },
  {
    key: 'orders',
    label: '订单中心',
    path: '/orders',
    permission: PERMISSIONS.ORDER_READ,
  },
  {
    key: 'payments',
    label: '支付与退款',
    children: [
      {
        key: 'payment-orders',
        label: '支付单',
        path: '/payments/orders',
        permission: PERMISSIONS.PAYMENT_READ,
      },
      {
        key: 'payment-transactions',
        label: '支付流水',
        path: '/payments/transactions',
        permission: PERMISSIONS.PAYMENT_READ,
      },
      {
        key: 'refunds',
        label: '退款单',
        path: '/payments/refunds',
        permission: PERMISSIONS.PAYMENT_READ,
      },
    ],
  },
  {
    key: 'members',
    label: '用户 / 会员',
    children: [
      {
        key: 'members-list',
        label: '会员列表',
        path: '/members',
        permission: PERMISSIONS.MEMBER_READ,
      },
      {
        key: 'wallet-ledger',
        label: '钱包账本',
        path: '/members/wallet-ledger',
        permission: PERMISSIONS.MEMBER_READ,
      },
    ],
  },
  {
    key: 'reports',
    label: '报表',
    path: '/reports',
    permission: PERMISSIONS.REPORT_READ,
  },
  {
    key: 'audit',
    label: '审计与日志',
    children: [
      {
        key: 'audit-logs',
        label: '审计日志',
        path: '/audit/logs',
        permission: PERMISSIONS.AUDIT_READ,
      },
      {
        key: 'error-events',
        label: '错误事件',
        path: '/audit/error-events',
        permission: PERMISSIONS.AUDIT_READ,
      },
    ],
  },
  {
    key: 'system',
    label: '系统设置',
    path: '/system/payment-settings',
    permission: PERMISSIONS.SYSTEM_PAYMENT_SETTINGS_WRITE,
  },
]

/** 扁平化菜单叶子节点，用于路由生成与守卫查表 */
export interface FlatMenuItem {
  path: string
  label: string
  permission?: PermissionCode
  /** 面包屑父级标签 */
  parentLabel?: string
}

export function flattenMenu(nodes: MenuNode[] = MENU, parentLabel?: string): FlatMenuItem[] {
  const result: FlatMenuItem[] = []
  for (const node of nodes) {
    if (node.path) {
      result.push({
        path: node.path,
        label: node.label,
        permission: node.permission,
        parentLabel,
      })
    }
    if (node.children) {
      result.push(...flattenMenu(node.children, node.label))
    }
  }
  return result
}
