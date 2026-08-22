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
import { encryptPassword } from '@/utils/password-encryption'
import type {
  Activity,
  ActivityTicketType,
  ActivityReportRow,
  AccountEntity,
  AuditLog,
  BusinessOrder,
  CatalogCategory,
  CatalogItem,
  CatalogItemReportRow,
  CouponReportRow,
  CouponTemplate,
  ErrorEvent,
  FranchiseInquiry,
  GlobalSettings,
  Member,
  MemberCouponEntitlement,
  MemberReportRow,
  MembershipTier,
  PaymentChannelSetting,
  PointReviewSettings,
  PrintJobLog,
  PrinterDevice,
  PrinterDeviceInput,
  PrinterDevicePatch,
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
  StoreLowSpendRule,
  VenueSeat,
  VenueTable,
  StoreReportRow,
  WalletLedgerEntry,
} from '@/api/models'

const storeResource = createResource<Store>({ base: API_PATHS.stores.list })
export const storeService = {
  ...storeResource,
  remove: async (id: string, password: string) => {
    const encryptedPassword = await encryptPassword(password)
    return http.delete<void>(API_PATHS.stores.remove(id), {
      data: encryptedPassword,
      idempotent: true,
    })
  },
}
export const printerService = {
  list: async (query: Record<string, unknown> = {}) => {
    const devices = await http.get<PrinterDevice[]>(API_PATHS.printers.list, {
      ...(query.storeId ? { storeId: query.storeId } : {}),
    })
    const keyword = String(query.keyword ?? '').trim().toLowerCase()
    const status = String(query.status ?? '').trim()
    const filtered = devices.filter((device) => {
      if (status && device.status !== status) return false
      if (!keyword) return true
      return `${device.name} ${device.deviceSn} ${device.provider}`.toLowerCase().includes(keyword)
    })
    const page = Math.max(1, Number(query.page) || 1)
    const pageSize = Math.max(1, Number(query.pageSize) || 20)
    const start = (page - 1) * pageSize
    return {
      items: filtered.slice(start, start + pageSize),
      meta: { page, pageSize, total: filtered.length },
    }
  },
  create: (payload: PrinterDeviceInput) =>
    http.post<PrinterDevice>(API_PATHS.printers.list, payload, { idempotent: true }),
  update: (id: string, payload: PrinterDevicePatch) =>
    http.patch<PrinterDevice>(API_PATHS.printers.detail(id), payload, { idempotent: true }),
  testPrint: (id: string) =>
    http.post<{ id: string; printed: boolean }>(API_PATHS.printers.testPrint(id), {}, { idempotent: true }),
  remove: (id: string, reason: string) =>
    http.delete<void>(API_PATHS.printers.detail(id), { data: { reason }, idempotent: true }),
}
export const tableService = createResource<VenueTable>({ base: API_PATHS.tables.list })
export const seatService = createResource<VenueSeat>({ base: API_PATHS.seats.list })
const franchiseInquiryResource = createResource<FranchiseInquiry>({
  base: API_PATHS.franchiseInquiries.list,
})
export const franchiseInquiryService = {
  ...franchiseInquiryResource,
  updateStatus: (id: string, status: FranchiseInquiry['status']) =>
    http.patch<void>(API_PATHS.franchiseInquiries.status(id), { status }),
}

export const adminAccountService = createResource<AccountEntity>({
  base: API_PATHS.adminAccounts.list,
})
export const storeAdminAccountService = createResource<AccountEntity>({
  base: API_PATHS.storeAdminAccounts.list,
})
const staffAccountResource = createResource<AccountEntity>({
  base: API_PATHS.staffAccounts.list,
})
export const staffAccountService = {
  ...staffAccountResource,
  remove: (id: string) =>
    http.delete<void>(API_PATHS.staffAccounts.removeBinding(id), { idempotent: true }),
}

// 服务端 catalog/activity/coupon-template 的更新为 PUT（整体覆盖），非 PATCH。
const categoryResource = createResource<CatalogCategory>({
  base: API_PATHS.catalog.categories,
  updateMethod: 'put',
})
export const categoryService = {
  ...categoryResource,
  batchRemove: (ids: number[]) =>
    http.post<void>(API_PATHS.catalog.categoryBatchDelete, { ids }, { idempotent: true }),
}
const catalogItemResource = createResource<CatalogItem>({
  base: API_PATHS.catalog.items,
  idempotentWrites: true,
  updateMethod: 'put',
})
export const catalogItemService = {
  ...catalogItemResource,
  batchRemove: (ids: number[]) =>
    http.post<void>(API_PATHS.catalog.itemBatchDelete, { ids }, { idempotent: true }),
}

const activityResource = createResource<Activity>({
  base: API_PATHS.activities.list,
  idempotentWrites: true,
  updateMethod: 'put',
})
export const activityService = {
  ...activityResource,
  ticketTypes: (activityId: string) =>
    http.get<ActivityTicketType[]>(API_PATHS.activities.ticketTypes(activityId)),
  createTicketType: (activityId: string, payload: Partial<ActivityTicketType>) =>
    http.post<ActivityTicketType>(API_PATHS.activities.ticketTypes(activityId), payload, {
      idempotent: true,
    }),
  updateTicketType: (
    activityId: string,
    ticketTypeId: string,
    payload: Partial<ActivityTicketType>,
  ) =>
    http.put<ActivityTicketType>(
      API_PATHS.activities.ticketTypeDetail(activityId, ticketTypeId),
      payload,
      { idempotent: true },
    ),
  removeTicketType: (activityId: string, ticketTypeId: string) =>
    http.delete<void>(API_PATHS.activities.ticketTypeDetail(activityId, ticketTypeId), {
      idempotent: true,
    }),
}
export const couponTemplateService = createResource<CouponTemplate>({
  base: API_PATHS.coupons.templates,
  idempotentWrites: true,
  updateMethod: 'put',
})

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

const orderResource = createResource<BusinessOrder>({ base: API_PATHS.orders.list })
export const orderService = {
  ...orderResource,
  refund: (payload: {
    paymentOrderId: number
    amountCent: number
    reason: string
    password: string
  }) => {
    const { password, ...refund } = payload
    return encryptPassword(password).then((encryptedPassword) =>
      http.post(API_PATHS.payments.refunds, { ...refund, ...encryptedPassword }, { idempotent: true }),
    )
  },
}
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
  memberCouponEntitlements: (memberId: string, query?: Record<string, unknown>) =>
    http.getList<MemberCouponEntitlement>(API_PATHS.members.couponEntitlements(memberId), query),
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
  printJobs: (query?: Record<string, unknown>) =>
    http.getList<PrintJobLog>(API_PATHS.audit.printJobs, query),
  errorEvents: (query?: Record<string, unknown>) =>
    http.getList<ErrorEvent>(API_PATHS.audit.errorEvents, query),
}

/** 系统设置：支付渠道配置（读写，写操作高风险）。GET 返回渠道列表，PUT 提交 { channels } 开关集合。 */
export const systemService = {
  getGlobalSettings: () => http.get<GlobalSettings>(API_PATHS.system.globalSettings),
  updateGlobalSettings: (settings: {
    firstRechargeDoublePointsEnabled: boolean
    rechargeDoublePointsThresholdAmount: number
    rechargeNotice: string
    franchiseInquirySources: string[]
    franchiseHotline: string
    phoneChangeIntervalDays: number
    printerDeveloperAccount: string
    printerDeveloperKey?: string
    printerApiUrl: string
  }) =>
    http.put<GlobalSettings>(API_PATHS.system.globalSettings, settings, { idempotent: true }),
  getPaymentChannelSettings: () =>
    http.get<PaymentChannelSetting[]>(API_PATHS.system.paymentChannelSettings),
  updatePaymentChannelSettings: (channels: { channel: string; enabled: boolean }[]) =>
    http.put<PaymentChannelSetting[]>(
      API_PATHS.system.paymentChannelSettings,
      { channels },
      { idempotent: true },
    ),
  getPointReviewSettings: () =>
    http.get<PointReviewSettings>(API_PATHS.system.pointReviewSettings),
  updatePointReviewSettings: (settings: {
    pointsDivisor: number
    belowBasePointsDivisor: number
    coinPointsDivisor: number
  }) =>
    http.put<PointReviewSettings>(API_PATHS.system.pointReviewSettings, settings, {
      idempotent: true,
    }),
  listStoreLowSpendRules: (query?: Record<string, unknown>) =>
    http.getList<StoreLowSpendRule>(API_PATHS.system.storeLowSpendRules, query),
  updateStoreLowSpendRule: (
    storeId: number,
    settings: Omit<StoreLowSpendRule, 'storeId' | 'storeName' | 'configured' | 'updatedAt'>,
  ) =>
    http.put<StoreLowSpendRule>(API_PATHS.system.storeLowSpendRule(storeId), settings, {
      idempotent: true,
    }),
  deleteStoreLowSpendRule: (storeId: number) =>
    http.delete<void>(API_PATHS.system.storeLowSpendRule(storeId), { idempotent: true }),
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
