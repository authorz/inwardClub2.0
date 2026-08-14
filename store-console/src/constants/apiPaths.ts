/**
 * 集中管理的门店后台接口路径。
 *
 * 所有服务只能从这里取路径，禁止在服务/页面里硬编码字符串路径。
 * 全部以 /store 开头（base url 已包含 /api/v2），确保门店后台只触达 /api/v2/store/*。
 */

export const API_PATHS = {
  assets: {
    uploadCredentials: '/store/assets/upload-credentials',
  },
  auth: {
    login: '/store/auth/login',
    refresh: '/store/auth/refresh',
    me: '/store/auth/me',
    logout: '/store/auth/logout',
  },

  profile: {
    get: '/store/profile',
    update: '/store/profile',
    status: '/store/profile/status',
  },

  lowSpendRule: {
    get: '/store/low-spend-reward-rule',
    update: '/store/low-spend-reward-rule',
  },

  reports: {
    overview: '/store/reports/overview',
    revenue: '/store/reports/revenue',
    catalogItems: '/store/reports/catalog-items',
    activities: '/store/reports/activities',
    coupons: '/store/reports/coupons',
  },

  orders: {
    list: '/store/orders',
    detail: (id: string | number) => `/store/orders/${id}`,
    foodOrders: '/store/food-orders',
    paymentOrders: '/store/payment-orders',
    paymentOrderDetail: (id: string | number) => `/store/payment-orders/${id}`,
    paymentTransactions: '/store/payment-transactions',
    refundOrders: '/store/refund-orders',
    refunds: '/store/refunds',
    foodAction: (id: string | number, action: string) => `/store/food-orders/${id}/${action}`,
  },

  catalog: {
    categories: '/store/catalog/categories',
    category: (id: string | number) => `/store/catalog/categories/${id}`,
    globalItems: '/store/catalog/global-items',
    adoptGlobalItem: (id: string | number) => `/store/catalog/global-items/${id}/adopt`,
    items: '/store/catalog/items',
    item: (id: string | number) => `/store/catalog/items/${id}`,
    publishItem: (id: string | number) => `/store/catalog/items/${id}/publish`,
    unpublishItem: (id: string | number) => `/store/catalog/items/${id}/unpublish`,
    itemStock: (id: string | number) => `/store/catalog/items/${id}/stock`,
    itemPaymentRules: (id: string | number) => `/store/catalog/items/${id}/payment-rules`,
    itemVariants: (id: string | number) => `/store/catalog/variants/${id}`,
    variant: (itemId: string | number, id: string | number) => `/store/catalog/variants/${itemId}/${id}`,
  },

  activities: {
    globalActivities: '/store/activities/global-activities',
    adoptGlobal: (id: string | number) => `/store/activities/global-activities/${id}/adopt`,
    list: '/store/activities',
    detail: (id: string | number) => `/store/activities/${id}`,
    publish: (id: string | number) => `/store/activities/${id}/publish`,
    unpublish: (id: string | number) => `/store/activities/${id}/unpublish`,
    generateShareAssets: (id: string | number) => `/store/activities/${id}/generate-share-assets`,
    sessions: (id: string | number) => `/store/activities/${id}/sessions`,
    session: (id: string | number) => `/store/activity-sessions/${id}`,
    ticketTypes: (id: string | number) => `/store/activities/${id}/ticket-types`,
    ticketType: (activityId: string | number, id: string | number) =>
      `/store/activities/${activityId}/ticket-types/${id}`,
    orders: '/store/activity-orders',
  },

  tickets: {
    list: '/store/tickets',
    verify: '/store/tickets/verify',
  },

  verifications: {
    list: '/store/verifications',
  },

  coupons: {
    templates: '/store/coupon-templates',
    template: (id: string | number) => `/store/coupon-templates/${id}`,
    publishTemplate: (id: string | number) => `/store/coupon-templates/${id}/publish`,
    disableTemplate: (id: string | number) => `/store/coupon-templates/${id}/disable`,
    applicableItems: (id: string | number) => `/store/coupon-templates/${id}/applicable-items`,
    grant: '/store/coupon-grants',
    entitlements: '/store/coupon-entitlements',
    redemptions: '/store/coupon-redemptions',
    // 券核销/作废为独立扁平写端点（idempotent），非 /coupon-entitlements/:id/... 子路由。
    void: '/store/coupon-voids',
    verify: '/store/coupon-verifications',
  },

  pointSavings: {
    list: '/store/point-savings',
    review: (id: string | number) => `/store/point-savings/${id}/review`,
  },

  reservations: {
    list: '/store/reservations',
    detail: (id: string | number) => `/store/reservations/${id}`,
    cancel: (id: string | number) => `/store/reservations/${id}/cancel`,
  },

  tables: {
    list: '/store/tables',
    table: (id: string | number) => `/store/tables/${id}`,
    seats: '/store/seats',
    seat: (id: string | number) => `/store/seats/${id}`,
  },

  collection: {
    orders: '/store/offline-collection-orders',
    order: (id: string | number) => `/store/offline-collection-orders/${id}`,
    cancel: (id: string | number) => `/store/offline-collection-orders/${id}/cancel`,
  },

  members: {
    list: '/store/members',
    lookup: '/store/member-lookup',
    detail: (id: string | number) => `/store/members/${id}`,
    walletAdjustments: (id: string | number) => `/store/members/${id}/wallet-adjustments`,
    walletLedger: '/store/wallet-ledger',
  },

  staff: {
    cashiers: '/store/cashiers',
    cashier: (id: string | number) => `/store/cashiers/${id}`,
    cashierDisable: (id: string | number) => `/store/cashiers/${id}/disable`,
    cashierPasswordReset: (id: string | number) => `/store/cashiers/${id}/password-reset`,
    staffAccounts: '/store/staff-accounts',
    staffAccount: (id: string | number) => `/store/staff-accounts/${id}`,
    staffDisable: (id: string | number) => `/store/staff-accounts/${id}/disable`,
    staffBinding: (id: string | number) => `/store/staff-accounts/${id}/binding`,
  },

  banners: {
    list: '/store/banners',
    banner: (id: string | number) => `/store/banners/${id}`,
  },

  printers: {
    list: '/store/printer-devices',
    device: (id: string | number) => `/store/printer-devices/${id}`,
  },
} as const
