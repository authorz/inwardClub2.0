/**
 * 总后台 API 路径集中定义（单一事实来源）。
 *
 * 依据 docs/CLAUDE_GO_2_0_IMPLEMENTATION_SPEC.md 与 docs/ADMIN_CONSOLE_ARCHITECTURE.md。
 * 约束：
 *  - 总后台只允许 /admin/* 与必要公共资产接口。
 *  - 禁止出现 /store/*（http client 会在运行时再次拦截）。
 *  - 路径不带 /api/v2 前缀；前缀由 VITE_API_BASE_URL 提供。
 *
 * 所有页面/服务必须引用这里的路径，不允许在业务代码里手写字符串路径。
 */

/** 允许调用的路径前缀（http client 白名单校验用） */
export const ALLOWED_PATH_PREFIXES = ['/admin/', '/assets'] as const

/** 明确禁止的路径前缀（越界防护） */
export const FORBIDDEN_PATH_PREFIXES = ['/store/', '/mini/'] as const

export const API_PATHS = {
  auth: {
    login: '/admin/auth/login',
    refresh: '/admin/auth/refresh',
    me: '/admin/auth/me',
    logout: '/admin/auth/logout',
  },

  assets: {
    // 直传凭证：服务端签发七牛上传 token/objectKey，前端据此直传后回传 assetId。
    uploadCredentials: '/admin/assets/upload-credentials',
  },

  stores: {
    list: '/admin/stores',
    create: '/admin/stores',
    detail: (id: string) => `/admin/stores/${id}`,
    update: (id: string) => `/admin/stores/${id}`,
  },

  franchiseInquiries: {
    list: '/admin/franchise-inquiries',
    status: (id: string) => `/admin/franchise-inquiries/${id}/status`,
  },

  tables: {
    list: '/admin/tables',
    detail: (id: string) => `/admin/tables/${id}`,
  },

  seats: {
    list: '/admin/seats',
    detail: (id: string) => `/admin/seats/${id}`,
  },

  adminAccounts: {
    list: '/admin/admin-accounts',
    create: '/admin/admin-accounts',
    update: (id: string) => `/admin/admin-accounts/${id}`,
    disable: (id: string) => `/admin/admin-accounts/${id}/disable`,
  },

  storeAdminAccounts: {
    list: '/admin/store-admin-accounts',
    create: '/admin/store-admin-accounts',
    update: (id: string) => `/admin/store-admin-accounts/${id}`,
    disable: (id: string) => `/admin/store-admin-accounts/${id}/disable`,
  },

  staffAccounts: {
    list: '/admin/staff-accounts',
    create: '/admin/staff-accounts',
    update: (id: string) => `/admin/staff-accounts/${id}`,
    disable: (id: string) => `/admin/staff-accounts/${id}/disable`,
    removeBinding: (id: string) => `/admin/staff-accounts/${id}/binding`,
  },

  catalog: {
    categories: '/admin/catalog/categories',
    categoryDetail: (id: string) => `/admin/catalog/categories/${id}`,
    items: '/admin/catalog/items',
    itemDetail: (id: string) => `/admin/catalog/items/${id}`,
    itemAssignStores: (id: string) => `/admin/catalog/items/${id}/assign-stores`,
    itemStoreOverrides: (id: string) => `/admin/catalog/items/${id}/store-overrides`,
    itemStoreOverride: (id: string, storeId: string) =>
      `/admin/catalog/items/${id}/store-overrides/${storeId}`,
    itemVariants: (id: string) => `/admin/catalog/items/${id}/variants`,
    variantDetail: (variantId: string) => `/admin/catalog/variants/${variantId}`,
  },

  activities: {
    list: '/admin/activities',
    create: '/admin/activities',
    detail: (id: string) => `/admin/activities/${id}`,
    assignStores: (id: string) => `/admin/activities/${id}/assign-stores`,
    generateShareAssets: (id: string) => `/admin/activities/${id}/generate-share-assets`,
    sessions: (id: string) => `/admin/activities/${id}/sessions`,
    sessionDetail: (sessionId: string) => `/admin/activity-sessions/${sessionId}`,
    ticketTypes: (id: string) => `/admin/activities/${id}/ticket-types`,
    ticketTypeDetail: (activityId: string, ticketTypeId: string) =>
      `/admin/activities/${activityId}/ticket-types/${ticketTypeId}`,
    orders: '/admin/activity-orders',
    tickets: '/admin/tickets',
    verifications: '/admin/verifications',
  },

  tournamentEvents: {
    list: '/admin/tournament-events',
    detail: (id: string) => `/admin/tournament-events/${id}`,
  },

  coupons: {
    templates: '/admin/coupon-templates',
    templateDetail: (id: string) => `/admin/coupon-templates/${id}`,
    templateAssignStores: (id: string) => `/admin/coupon-templates/${id}/assign-stores`,
    templateApplicableItems: (id: string) => `/admin/coupon-templates/${id}/applicable-items`,
    publishTemplate: (id: string) => `/admin/coupon-templates/${id}/publish`,
    disableTemplate: (id: string) => `/admin/coupon-templates/${id}/disable`,
    entitlementsGrant: '/admin/coupon-entitlements/grant',
    entitlements: '/admin/coupon-entitlements',
    redemptions: '/admin/coupon-redemptions',
    entitlementVoid: (id: string) => `/admin/coupon-entitlements/${id}/void`,
  },

  banners: {
    list: '/admin/banners',
    create: '/admin/banners',
    update: (id: string) => `/admin/banners/${id}`,
    remove: (id: string) => `/admin/banners/${id}`,
  },

  rechargeProducts: {
    list: '/admin/recharge-products',
    create: '/admin/recharge-products',
    update: (id: string) => `/admin/recharge-products/${id}`,
    disable: (id: string) => `/admin/recharge-products/${id}/disable`,
  },

  membershipTiers: {
    list: '/admin/membership-tiers',
    create: '/admin/membership-tiers',
    update: (id: string) => `/admin/membership-tiers/${id}`,
    disable: (id: string) => `/admin/membership-tiers/${id}/disable`,
  },

  ruleDefinitions: {
    list: '/admin/rule-definitions',
    create: '/admin/rule-definitions',
    update: (id: string) => `/admin/rule-definitions/${id}`,
    publish: (id: string) => `/admin/rule-definitions/${id}/publish`,
    disable: (id: string) => `/admin/rule-definitions/${id}/disable`,
  },

  orders: {
    list: '/admin/orders',
    detail: (id: string) => `/admin/orders/${id}`,
  },

  payments: {
    orders: '/admin/payment-orders',
    transactions: '/admin/payment-transactions',
    refundOrders: '/admin/refund-orders',
    refunds: '/admin/refunds',
  },

  members: {
    list: '/admin/members',
    lookup: '/admin/member-lookup',
    detail: (id: string) => `/admin/members/${id}`,
    update: (id: string) => `/admin/members/${id}`,
    walletAdjustments: (id: string) => `/admin/members/${id}/wallet-adjustments`,
    walletLedger: '/admin/wallet-ledger',
  },

  reports: {
    overview: '/admin/reports/overview',
    revenue: '/admin/reports/revenue',
    catalogItems: '/admin/reports/catalog-items',
    activities: '/admin/reports/activities',
    coupons: '/admin/reports/coupons',
    records: '/admin/reports/records',
    members: '/admin/reports/members',
    reservations: '/admin/reports/reservations',
    stores: '/admin/reports/stores',
  },

  audit: {
    logs: '/admin/audit-logs',
    errorEvents: '/admin/error-events',
    maintenanceCleanup: '/admin/audit-log-maintenance/cleanup',
  },

  system: {
    globalSettings: '/admin/global-settings',
    storeLowSpendRules: '/admin/store-low-spend-rules',
    storeLowSpendRule: (storeId: string | number) => `/admin/store-low-spend-rules/${storeId}`,
    paymentChannelSettings: '/admin/payment-channel-settings',
    pointReviewSettings: '/admin/point-review-settings',
  },
} as const
