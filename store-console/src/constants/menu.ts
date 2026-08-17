/**
 * 门店后台菜单配置，作为侧边菜单与路由的单一数据源。
 *
 * 每一项声明 route name、路径、展示标题、图标名与所需权限码。
 * 侧边栏据此渲染，路由守卫据此做权限判断，避免菜单与路由/权限三处各写一套。
 */

import { PERM, type PermissionCode } from './permissions'

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
}

export interface MenuGroup {
  /** 分组标题；为空表示无分组标题（如工作台）。 */
  title?: string
  items: MenuItem[]
}

export const MENU: MenuGroup[] = [
  {
    items: [
      { name: 'dashboard', path: '/dashboard', title: '概览', icon: 'dashboard' },
    ],
  },
  {
    title: '订单与履约',
    items: [
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
    ],
  },
  {
    title: '核销与审核',
    items: [
      {
        name: 'activity-verify',
        path: '/activity-verify',
        title: '活动核销',
        icon: 'ticket',
        permissions: [PERM.ticketVerify],
      },
      {
        name: 'ticket-verify',
        path: '/ticket-verify',
        title: '票券核销',
        icon: 'coupon',
        permissions: [PERM.ticketVerify],
      },
      {
        name: 'point-review',
        path: '/point-review',
        title: '积分审核',
        icon: 'review',
        permissions: [PERM.pointReview],
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
    title: '经营',
    items: [
      {
        name: 'catalog-categories',
        path: '/catalog-categories',
        title: '商品分类',
        icon: 'catalog',
        permissions: [PERM.catalogRead, PERM.catalogWrite],
      },
      {
        name: 'catalog',
        path: '/catalog',
        title: '本店商品',
        icon: 'catalog',
        permissions: [PERM.catalogRead, PERM.catalogWrite],
      },
      {
        name: 'activities',
        path: '/activities',
        title: '本店活动',
        icon: 'activity',
        permissions: [PERM.activityRead, PERM.activityWrite],
      },
      {
        name: 'coupons',
        path: '/coupons',
        title: '本店优惠券',
        icon: 'coupon',
        permissions: [PERM.couponRead, PERM.couponWrite],
      },
      {
        name: 'banners',
        path: '/banners',
        title: '本店广告',
        icon: 'activity',
        permissions: [PERM.activityRead, PERM.activityWrite],
      },
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
    title: '会员',
    items: [
      {
        name: 'members',
        path: '/members',
        title: '会员管理',
        icon: 'member',
        permissions: [PERM.memberRead, PERM.memberReadLimited],
      },
    ],
  },
  {
    title: '收款',
    items: [
      {
        name: 'collection',
        path: '/collection',
        title: '线下聚合收款',
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
        name: 'payments',
        path: '/payments',
        title: '支付与退款',
        icon: 'payment',
        permissions: [PERM.orderRead],
      },
    ],
  },
  {
    title: '运营配置',
    items: [
      {
        name: 'cashiers',
        path: '/cashiers',
        title: '收银员管理',
        icon: 'staff',
        permissions: [PERM.staffWrite],
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
      {
        name: 'reports',
        path: '/reports',
        title: '本店报表',
        icon: 'report',
        permissions: [PERM.reportRead],
      },
      { name: 'settings', path: '/settings', title: '设置', icon: 'settings' },
    ],
  },
]

/** 扁平化菜单，便于路由生成与权限查找。 */
export const FLAT_MENU: MenuItem[] = MENU.flatMap((group) => group.items)
