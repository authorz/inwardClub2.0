/**
 * 路由定义。
 *
 * 业务页面全部作为 DefaultLayout 的子路由，懒加载。
 * 每条路由的 title / permission / breadcrumb 从 MENU 派生（flattenMenu），
 * 避免菜单与路由两处各写一套权限/标题。
 */
import type { RouteRecordRaw } from 'vue-router'
import { flattenMenu } from '@/layouts/menu'
import type { PermissionCode } from '@/constants/permissions'

// 带索引签名以兼容 vue-router 的 RouteMeta（其要求可赋值给 Record<PropertyKey, unknown>）
export interface AppRouteMeta {
  title: string
  permission?: PermissionCode
  breadcrumb: string[]
  requiresAuth: boolean
  [key: string]: unknown
  [key: symbol]: unknown
}

const flat = flattenMenu()
function metaFor(path: string): AppRouteMeta {
  const item = flat.find((f) => f.path === path)
  const label = item?.label ?? ''
  const breadcrumb = item?.parentLabel ? [item.parentLabel, label] : [label]
  return {
    title: label,
    permission: item?.permission,
    breadcrumb,
    requiresAuth: true,
  }
}

/** path -> 懒加载组件 的业务路由表 */
const businessRoutes: { path: string; component: NonNullable<RouteRecordRaw['component']> }[] = [
  { path: '/dashboard', component: () => import('@/pages/DashboardView.vue') },
  { path: '/stores', component: () => import('@/pages/stores/StoreListView.vue') },
  { path: '/stores/printers', component: () => import('@/pages/stores/PrinterListView.vue') },
  {
    path: '/franchise-inquiries',
    component: () => import('@/pages/franchise/FranchiseInquiryListView.vue'),
  },
  { path: '/tables', component: () => import('@/pages/tables/TableListView.vue') },
  { path: '/seats', component: () => import('@/pages/seats/SeatListView.vue') },
  { path: '/accounts/admins', component: () => import('@/pages/accounts/AdminAccountsView.vue') },
  {
    path: '/accounts/store-admins',
    component: () => import('@/pages/accounts/StoreAdminAccountsView.vue'),
  },
  { path: '/accounts/staff', component: () => import('@/pages/accounts/StaffAccountsView.vue') },
  { path: '/catalog/categories', component: () => import('@/pages/catalog/CategoryListView.vue') },
  { path: '/catalog/items', component: () => import('@/pages/catalog/ItemListView.vue') },
  { path: '/activities', component: () => import('@/pages/activities/ActivityListView.vue') },
  { path: '/coupons', component: () => import('@/pages/coupons/CouponTypeListView.vue') },
  {
    path: '/recharge-products',
    component: () => import('@/pages/recharge/RechargeProductListView.vue'),
  },
  {
    path: '/rules/membership-tiers',
    component: () => import('@/pages/rules/MembershipTierListView.vue'),
  },
  {
    path: '/rules/sign-in',
    component: () => import('@/pages/rules/SignInRuleView.vue'),
  },
  {
    path: '/rules/invitations',
    component: () => import('@/pages/rules/InvitationRewardRuleView.vue'),
  },
  {
    path: '/rules/store-low-spend',
    component: () => import('@/pages/rules/StoreLowSpendRuleListView.vue'),
  },
  { path: '/orders', component: () => import('@/pages/orders/OrderListView.vue') },
  { path: '/payments/refunds', component: () => import('@/pages/payments/RefundListView.vue') },
  { path: '/members', component: () => import('@/pages/members/MemberListView.vue') },
  {
    path: '/members/wallet-ledger',
    component: () => import('@/pages/members/WalletLedgerView.vue'),
  },
  { path: '/reports', component: () => import('@/pages/reports/ReportsView.vue') },
  { path: '/audit/logs', component: () => import('@/pages/audit/AuditLogView.vue') },
  { path: '/audit/print-jobs', component: () => import('@/pages/audit/PrintJobLogView.vue') },
  { path: '/audit/error-events', component: () => import('@/pages/audit/ErrorEventView.vue') },
  {
    path: '/system/global-settings',
    component: () => import('@/pages/system/GlobalSettingsView.vue'),
  },
  {
    path: '/system/payment-settings',
    component: () => import('@/pages/system/PaymentSettingsView.vue'),
  },
  {
    path: '/system/point-review-settings',
    component: () => import('@/pages/system/PointReviewSettingsView.vue'),
  },
]

export const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/pages/LoginView.vue'),
    meta: { requiresAuth: false, title: '登录' },
  },
  {
    path: '/',
    component: () => import('@/layouts/DefaultLayout.vue'),
    redirect: '/dashboard',
    children: businessRoutes.map(
      (r): RouteRecordRaw => ({
        path: r.path,
        component: r.component,
        meta: metaFor(r.path),
      }),
    ),
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/pages/NotFoundView.vue'),
    meta: { requiresAuth: false, title: '页面不存在' },
  },
]
