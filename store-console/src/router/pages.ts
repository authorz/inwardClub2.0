/**
 * 路由 name -> 页面组件 的懒加载映射。
 * 与 FLAT_MENU 的 name 一一对应。
 */

import type { Component } from 'vue'

export const pageComponents: Record<string, () => Promise<Component>> = {
  dashboard: () => import('@/pages/DashboardView.vue'),
  orders: () => import('@/pages/orders/OrdersView.vue'),
  'food-orders': () => import('@/pages/orders/FoodOrdersView.vue'),
  'activity-verify': () => import('@/pages/verify/ActivityVerifyView.vue'),
  'ticket-verify': () => import('@/pages/verify/TicketVerifyView.vue'),
  'point-review': () => import('@/pages/review/PointReviewView.vue'),
  verifications: () => import('@/pages/records/VerificationsView.vue'),
  catalog: () => import('@/pages/catalog/CatalogView.vue'),
  'catalog-categories': () => import('@/pages/catalog/CategoryView.vue'),
  activities: () => import('@/pages/activities/ActivitiesView.vue'),
  coupons: () => import('@/pages/coupons/CouponsView.vue'),
  reservations: () => import('@/pages/reservations/ReservationsView.vue'),
  venues: () => import('@/pages/venues/VenuesView.vue'),
  collection: () => import('@/pages/collection/CollectionView.vue'),
  'collection-records': () => import('@/pages/collection/CollectionRecordsView.vue'),
  members: () => import('@/pages/members/MembersView.vue'),
  'wallet-ledger': () => import('@/pages/members/WalletLedgerView.vue'),
  payments: () => import('@/pages/payments/PaymentsView.vue'),
  cashiers: () => import('@/pages/staff/CashiersView.vue'),
  'staff-accounts': () => import('@/pages/staff/StaffAccountsView.vue'),
  printers: () => import('@/pages/printers/PrintersView.vue'),
  reports: () => import('@/pages/reports/ReportsView.vue'),
  settings: () => import('@/pages/settings/SettingsView.vue'),
}
