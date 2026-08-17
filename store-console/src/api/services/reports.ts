/**
 * 本店报表服务（仅本店范围，无跨店维度）。
 */

import { get } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { ReportOverview, RevenueReportQuery, RevenueReportRow } from '@/types/models'

export const reportService = {
  overview() {
    // 概览无入参（服务端忽略任何 query，如 range）。
    return get<ReportOverview>(API_PATHS.reports.overview)
  },
  revenue(params: RevenueReportQuery) {
    return get<RevenueReportRow[]>(API_PATHS.reports.revenue, { params })
  },
  catalogItems(params?: Record<string, unknown>) {
    return get<Record<string, unknown>[]>(API_PATHS.reports.catalogItems, { params })
  },
  activities(params?: Record<string, unknown>) {
    return get<Record<string, unknown>[]>(API_PATHS.reports.activities, { params })
  },
}
