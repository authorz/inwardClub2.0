/**
 * 积分存入审核服务。
 * 审核（通过/驳回）为高风险写操作，带 Idempotency-Key。
 */

import { get, getPaged, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { PointSavingRequest } from '@/types/models'
import type { ReviewStatus } from '@/constants/enums'

export const pointSavingService = {
  list(params?: PageQuery) {
    return getPaged<PointSavingRequest>(API_PATHS.pointSavings.list, params)
  },
  detail(id: string | number) {
    return get<PointSavingRequest>(API_PATHS.pointSavings.detail(id))
  },
  review(id: string | number, decision: Extract<ReviewStatus, 'approved' | 'rejected'>, reason?: string) {
    // 服务端 decision 取值为 approve/reject，备注字段名为 remark。
    return post<PointSavingRequest>(
      API_PATHS.pointSavings.review(id),
      { decision: decision === 'approved' ? 'approve' : 'reject', remark: reason },
      { idempotent: true },
    )
  },
}
