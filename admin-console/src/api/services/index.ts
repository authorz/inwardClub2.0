/**
 * 领域服务集合（公共服务清单）。
 *
 * 绝大多数模块直接复用 createResource 生成的标准 CRUD 服务；
 * 只有需要特殊子动作（publish/assign-stores/refund 等）的模块在页面里通过
 * service.action(API_PATHS.xxx) 调用，无需为每个动作再写一个函数。
 *
 * 这样「接口路径集中在 api-paths.ts、请求逻辑集中在 http.ts、CRUD 形状集中在 resource.ts」，
 * 服务层本身几乎零重复。
 */
import { API_PATHS } from '@/constants/api-paths'
import { createResource } from '@/api/resource'
import { http } from '@/api/http'
import type {
  Activity,
  ActivityReportRow,
  AccountEntity,
  AuditLog,
  Banner,
  BusinessOrder,
  CatalogCategory,
  CatalogItem,
  CatalogItemReportRow,
  CouponReportRow,
  CouponTemplate,
  ErrorEvent,
  Member,
  MemberReportRow,
  MembershipTier,
  PaymentChannelSetting,
  PaymentOrder,
  PaymentTransaction,
  RechargeProduct,
  RecordReportRow,
  RefundOrder,
  ReportOverview,
  ReservationReportRow,
  RevenueReportRow,
  RuleDefinition,
  Store,
  StoreReportRow,
  StoreSettings,
  StoreSettingsData,
  WalletLedgerEntry,
} from '@/api/models'

export const storeService = createResource<Store>({ base: API_PATHS.stores.list })

/** 门店运营配置（读写，写操作高风险）。服务端约定为 { settings: {...} } 整体覆盖。 */
export const storeSettingsService = {
  get: (storeId: string) => http.get<StoreSettings>(API_PATHS.stores.settings(storeId)),
  update: (storeId: string, settings: Partial<StoreSettingsData>) =>
    http.put<StoreSettings>(API_PATHS.stores.settings(storeId), { settings }, { idempotent: true }),
}

export const adminAccountService = createResource<AccountEntity>({
  base: API_PATHS.adminAccounts.list,
})
export const storeAdminAccountService = createResource<AccountEntity>({
  base: API_PATHS.storeAdminAccounts.list,
})
export const staffAccountService = createResource<AccountEntity>({
  base: API_PATHS.staffAccounts.list,
})

// 服务端 catalog/activity/coupon-template 的更新为 PUT（整体覆盖），非 PATCH。
export const categoryService = createResource<CatalogCategory>({
  base: API_PATHS.catalog.categories,
  updateMethod: 'put',
})
export const catalogItemService = createResource<CatalogItem>({
  base: API_PATHS.catalog.items,
  idempotentWrites: true,
  updateMethod: 'put',
})

export const activityService = createResource<Activity>({
  base: API_PATHS.activities.list,
  idempotentWrites: true,
  updateMethod: 'put',
})
export const couponTemplateService = createResource<CouponTemplate>({
  base: API_PATHS.coupons.templates,
  idempotentWrites: true,
  updateMethod: 'put',
})

export const bannerService = createResource<Banner>({ base: API_PATHS.banners.list })
export const rechargeProductService = createResource<RechargeProduct>({
  base: API_PATHS.rechargeProducts.list,
})
export const membershipTierService = createResource<MembershipTier>({
  base: API_PATHS.membershipTiers.list,
})
export const ruleDefinitionService = createResource<RuleDefinition>({
  base: API_PATHS.ruleDefinitions.list,
  idempotentWrites: true,
})

export const orderService = createResource<BusinessOrder>({ base: API_PATHS.orders.list })
export const memberService = createResource<Member>({ base: API_PATHS.members.list })

/** 只读列表类资源，直接用 http.getList，不需要完整 CRUD */
export const readonlyLists = {
  paymentOrders: (query?: Record<string, unknown>) =>
    http.getList<PaymentOrder>(API_PATHS.payments.orders, query),
  paymentTransactions: (query?: Record<string, unknown>) =>
    http.getList<PaymentTransaction>(API_PATHS.payments.transactions, query),
  refundOrders: (query?: Record<string, unknown>) =>
    http.getList<RefundOrder>(API_PATHS.payments.refundOrders, query),
  walletLedger: (query?: Record<string, unknown>) =>
    http.getList<WalletLedgerEntry>(API_PATHS.members.walletLedger, query),
  activityOrders: (query?: Record<string, unknown>) =>
    http.getList(API_PATHS.activities.orders, query),
  tickets: (query?: Record<string, unknown>) => http.getList(API_PATHS.activities.tickets, query),
  verifications: (query?: Record<string, unknown>) =>
    http.getList(API_PATHS.activities.verifications, query),
  couponEntitlements: (query?: Record<string, unknown>) =>
    http.getList(API_PATHS.coupons.entitlements, query),
  couponRedemptions: (query?: Record<string, unknown>) =>
    http.getList(API_PATHS.coupons.redemptions, query),
  auditLogs: (query?: Record<string, unknown>) =>
    http.getList<AuditLog>(API_PATHS.audit.logs, query),
  errorEvents: (query?: Record<string, unknown>) =>
    http.getList<ErrorEvent>(API_PATHS.audit.errorEvents, query),
}

/** 系统设置：支付渠道配置（读写，写操作高风险）。GET 返回渠道列表，PUT 提交 { channels } 开关集合。 */
export const systemService = {
  getPaymentChannelSettings: () =>
    http.get<PaymentChannelSetting[]>(API_PATHS.system.paymentChannelSettings),
  updatePaymentChannelSettings: (channels: { channel: string; enabled: boolean }[]) =>
    http.put<PaymentChannelSetting[]>(
      API_PATHS.system.paymentChannelSettings,
      { channels },
      { idempotent: true },
    ),
}

/** 报表列表接口统一用 created 筛选（daterange 写入 createdFrom/createdTo），转换为后端约定的 from/to */
function withReportRange(query?: Record<string, unknown>): Record<string, unknown> | undefined {
  if (!query) return query
  const { createdFrom, createdTo, ...rest } = query
  return {
    ...rest,
    ...(createdFrom ? { from: createdFrom } : {}),
    ...(createdTo ? { to: createdTo } : {}),
  }
}

/** 报表：只读汇总（总部经营报表，见 /admin/reports/*） */
export const reportService = {
  overview: (query?: Record<string, unknown>) =>
    http.get<ReportOverview>(API_PATHS.reports.overview, query),
  revenue: (query?: Record<string, unknown>) =>
    http.getList<RevenueReportRow>(API_PATHS.reports.revenue, withReportRange(query)),
  catalogItems: (query?: Record<string, unknown>) =>
    http.getList<CatalogItemReportRow>(API_PATHS.reports.catalogItems, withReportRange(query)),
  activities: (query?: Record<string, unknown>) =>
    http.getList<ActivityReportRow>(API_PATHS.reports.activities, withReportRange(query)),
  coupons: (query?: Record<string, unknown>) =>
    http.getList<CouponReportRow>(API_PATHS.reports.coupons, withReportRange(query)),
  records: (query?: Record<string, unknown>) =>
    http.getList<RecordReportRow>(API_PATHS.reports.records, withReportRange(query)),
  members: (query?: Record<string, unknown>) =>
    http.getList<MemberReportRow>(API_PATHS.reports.members, withReportRange(query)),
  reservations: (query?: Record<string, unknown>) =>
    http.getList<ReservationReportRow>(API_PATHS.reports.reservations, withReportRange(query)),
  stores: (query?: Record<string, unknown>) =>
    http.getList<StoreReportRow>(API_PATHS.reports.stores, withReportRange(query)),
}
