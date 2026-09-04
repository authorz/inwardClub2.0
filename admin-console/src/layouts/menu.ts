/**
 * 总后台菜单配置（单一事实来源）。
 *
 * 菜单结构依据 docs/ADMIN_CONSOLE_ARCHITECTURE.md 第 7 节。
 * 侧边菜单渲染、路由 meta（标题 + 权限）、面包屑都从这里派生，避免重复维护。
 * 每一项挂 permission，用于路由守卫与菜单显隐。
 */
import type { Component } from 'vue'
import {
  Armchair,
  BuildingStore,
  CalendarEvent,
  Crown,
  Dashboard,
  Database,
  MessageCircle,
  Package,
  Receipt,
  Settings,
  ShieldLock,
  Users,
} from '@vicons/tabler'
import { PERMISSIONS, type PermissionCode } from '@/constants/permissions'

export interface MenuNode {
  key: string
  label: string
  /** 路由 path（叶子节点必填） */
  path?: string
  /** 页面所需权限码 */
  permission?: PermissionCode
  /** 一级菜单使用的统一线性图标 */
  icon?: Component
  children?: MenuNode[]
}

export const MENU: MenuNode[] = [
  {
    key: 'dashboard',
    label: '概览',
    path: '/dashboard',
    permission: PERMISSIONS.REPORT_READ,
    icon: Dashboard,
  },
  {
    key: 'stores',
    label: '门店管理',
    icon: BuildingStore,
    children: [
      {
        key: 'store-list',
        label: '门店列表',
        path: '/stores',
        permission: PERMISSIONS.STORE_READ,
      },
      {
        key: 'printer-list',
        label: '打印机管理',
        path: '/stores/printers',
        permission: PERMISSIONS.STORE_READ,
      },
    ],
  },
  {
    key: 'franchise-inquiries',
    label: '加盟咨询',
    path: '/franchise-inquiries',
    permission: PERMISSIONS.STORE_READ,
    icon: MessageCircle,
  },
  {
    key: 'accounts',
    label: '管理员管理',
    icon: ShieldLock,
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
    key: 'activities',
    label: '活动管理',
    path: '/activities',
    permission: PERMISSIONS.ACTIVITY_READ,
    icon: CalendarEvent,
  },
  {
    key: 'catalog',
    label: '餐品管理',
    icon: Package,
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
      {
        key: 'coupon-items',
        label: '券商品',
        path: '/catalog/coupons',
        permission: PERMISSIONS.CATALOG_READ,
      },
    ],
  },
  {
    key: 'table-seats',
    label: '桌子管理',
    icon: Armchair,
    children: [
      {
        key: 'tables',
        label: '桌子列表',
        path: '/tables',
        permission: PERMISSIONS.STORE_READ,
      },
      {
        key: 'seats',
        label: '座位管理',
        path: '/seats',
        permission: PERMISSIONS.STORE_READ,
      },
    ],
  },
  {
    key: 'orders',
    label: '订单管理',
    icon: Receipt,
    children: [
      {
        key: 'order-list',
        label: '订单列表',
        path: '/orders',
        permission: PERMISSIONS.ORDER_READ,
      },
      {
        key: 'refunds',
        label: '退款记录',
        path: '/payments/refunds',
        permission: PERMISSIONS.PAYMENT_READ,
      },
    ],
  },
  {
    key: 'members',
    label: '用户管理',
    icon: Users,
    children: [
      {
        key: 'members-list',
        label: '会员列表',
        path: '/members',
        permission: PERMISSIONS.MEMBER_READ,
      },
      {
        key: 'wallet-ledger',
        label: '资产流水',
        path: '/members/wallet-ledger',
        permission: PERMISSIONS.MEMBER_READ,
      },
      {
        key: 'recharge-products',
        label: '快捷充值',
        path: '/recharge-products',
        permission: PERMISSIONS.RECHARGE_READ,
      },
    ],
  },
  {
    key: 'benefits',
    label: '权益规则',
    icon: Crown,
    children: [
      {
        key: 'membership-tiers',
        label: 'VIP 等级',
        path: '/rules/membership-tiers',
        permission: PERMISSIONS.RULE_READ,
      },
      {
        key: 'coupons',
        label: '券管理',
        path: '/coupons',
        permission: PERMISSIONS.COUPON_READ,
      },
      {
        key: 'sign-in-rule',
        label: '签到规则',
        path: '/rules/sign-in',
        permission: PERMISSIONS.RULE_READ,
      },
      {
        key: 'invitation-reward-rule',
        label: '邀请奖励',
        path: '/rules/invitations',
        permission: PERMISSIONS.RULE_READ,
      },
      {
        key: 'store-low-spend-rule',
        label: '预约低消奖励',
        path: '/rules/store-low-spend',
        permission: PERMISSIONS.RULE_READ,
      },
    ],
  },
  {
    key: 'data-logs',
    label: '数据与日志',
    icon: Database,
    children: [
      {
        key: 'reports',
        label: '经营报表',
        path: '/reports',
        permission: PERMISSIONS.REPORT_READ,
      },
      {
        key: 'audit-logs',
        label: '审计日志',
        path: '/audit/logs',
        permission: PERMISSIONS.AUDIT_READ,
      },
      {
        key: 'print-jobs',
        label: '打印日志',
        path: '/audit/print-jobs',
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
    icon: Settings,
    children: [
      {
        key: 'global-settings',
        label: '全局设置',
        path: '/system/global-settings',
        permission: PERMISSIONS.SYSTEM_PAYMENT_SETTINGS_WRITE,
      },
      {
        key: 'payment-settings',
        label: '支付渠道',
        path: '/system/payment-settings',
        permission: PERMISSIONS.SYSTEM_PAYMENT_SETTINGS_WRITE,
      },
      {
        key: 'point-review-settings',
        label: '积分审核配置',
        path: '/system/point-review-settings',
        permission: PERMISSIONS.SYSTEM_PAYMENT_SETTINGS_WRITE,
      },
    ],
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
