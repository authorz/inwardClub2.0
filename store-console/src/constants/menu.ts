/**
 * 门店后台菜单配置，作为侧边菜单与路由的单一数据源。
 *
 * 每一项声明 route name、路径、展示标题、图标名与所需权限码。
 * 侧边栏据此渲染，路由守卫据此做权限判断，避免菜单与路由/权限三处各写一套。
 */

import { PERM, type PermissionCode, type StoreRole } from './permissions'

export interface MenuItem {
  /** 路由 name，同时作为菜单 key。 */
  name: string
  /** 路由路径。 */
  path: string
  /** 展示标题。 */
  title: string
  /** 图标名（对应 AppIcon 中的键）。 */
  icon: string
  /**
   * 访问所需权限码。为空表示所有已登录门店账号可见。
   * 命中任一权限即可访问（OR 语义）。
   */
  permissions?: PermissionCode[]
  /** 仅指定门店角色可见；用于超级管理员专属入口。 */
  roles?: StoreRole[]
}

export interface MenuGroup {
  /** 分组唯一标识，用于可展开菜单的 key。 */
  name?: string
  /** 分组标题；为空时直接展示组内入口。 */
  title?: string
  /** 分组图标。 */
  icon?: string
  items: MenuItem[]
}

export const MENU: MenuGroup[] = [
  {
    items: [
      { name: 'dashboard', path: '/dashboard', title: '概览', icon: 'dashboard' },
    ],
  },
  {
    name: 'transactions',
    title: '收款与订单',
    icon: 'collection',
    items: [
      {
        name: 'collection',
        path: '/collection',
        title: '线下收款',
        icon: 'collection',
        permissions: [PERM.collectionCreate],
      },
      {
        name: 'collection-records',
        path: '/collection-records',
        title: '收款记录',
        icon: 'payment',
        permissions: [PERM.collectionRead],
      },
      {
        name: 'orders',
        path: '/orders',
        title: '本店订单',
        icon: 'orders',
        permissions: [PERM.orderRead],
      },
      {
        name: 'food-orders',
        path: '/food-orders',
        title: '点餐订单',
        icon: 'food',
        permissions: [PERM.orderRead],
      },
      {
        name: 'payments',
        path: '/payments',
        title: '支付与退款',
        icon: 'payment',
        permissions: [PERM.orderRead],
      },
    ],
  },
  {
    name: 'activity-management',
    title: '活动管理',
    icon: 'activity',
    items: [
      {
        name: 'activities',
        path: '/activities',
        title: '活动列表',
        icon: 'activity',
        permissions: [PERM.activityRead, PERM.activityWrite],
      },
      {
        name: 'activity-verify',
        path: '/activity-verify',
        title: '活动核销',
        icon: 'ticket',
        permissions: [PERM.ticketVerify],
      },
      {
        name: 'verifications',
        path: '/verifications',
        title: '核销记录',
        icon: 'records',
        permissions: [PERM.ticketVerify],
      },
    ],
  },
  {
    name: 'point-reviews',
    title: '积分审核',
    icon: 'review',
    items: [
      {
        name: 'point-review',
        path: '/point-review',
        title: '积分审核',
        icon: 'review',
        permissions: [PERM.pointReview],
      },
      {
        name: 'point-review-records',
        path: '/point-review-records',
        title: '审核记录',
        icon: 'records',
        permissions: [PERM.pointReview],
      },
    ],
  },
  {
    name: 'catalog',
    title: '商品管理',
    icon: 'catalog',
    items: [
      {
        name: 'catalog',
        path: '/catalog',
        title: '本店商品',
        icon: 'catalog',
        permissions: [PERM.catalogRead, PERM.catalogWrite],
      },
      {
        name: 'catalog-categories',
        path: '/catalog-categories',
        title: '商品分类',
        icon: 'catalog',
        permissions: [PERM.catalogRead, PERM.catalogWrite],
      },
      {
        name: 'coupons',
        path: '/coupons',
        title: '本店优惠券',
        icon: 'coupon',
        permissions: [PERM.couponRead, PERM.couponWrite],
      },
    ],
  },
  {
    name: 'reservations',
    title: '预约与桌位',
    icon: 'reservation',
    items: [
      {
        name: 'reservations',
        path: '/reservations',
        title: '预约记录',
        icon: 'reservation',
        permissions: [PERM.reservationWrite],
      },
      {
        name: 'venues',
        path: '/venues',
        title: '桌子与座位',
        icon: 'reservation',
        permissions: [PERM.reservationWrite],
      },
    ],
  },
  {
    name: 'members',
    title: '会员管理',
    icon: 'member',
    items: [
      {
        name: 'members',
        path: '/members',
        title: '会员列表',
        icon: 'member',
        permissions: [PERM.memberRead, PERM.memberReadLimited],
      },
      {
        name: 'wallet-ledger',
        path: '/members/wallet-ledger',
        title: '资产流水',
        icon: 'records',
        permissions: [PERM.memberRead, PERM.memberReadLimited],
      },
    ],
  },
  {
    name: 'store-management',
    title: '门店管理',
    icon: 'store',
    items: [
      {
        name: 'reports',
        path: '/reports',
        title: '经营报表',
        icon: 'report',
        permissions: [PERM.reportRead],
      },
      {
        name: 'cashiers',
        path: '/cashiers',
        title: '管理员管理',
        icon: 'staff',
        permissions: [PERM.staffWrite],
        roles: ['store_admin'],
      },
      {
        name: 'staff-accounts',
        path: '/staff-accounts',
        title: '员工账号管理',
        icon: 'staff',
        permissions: [PERM.staffWrite],
      },
      {
        name: 'printers',
        path: '/printers',
        title: '打印机管理',
        icon: 'printer',
        permissions: [PERM.printerWrite],
      },
      { name: 'settings', path: '/settings', title: '门店设置', icon: 'settings' },
    ],
  },
]

/** 扁平化菜单，便于路由生成与权限查找。 */
export const FLAT_MENU: MenuItem[] = MENU.flatMap((group) => group.items)
