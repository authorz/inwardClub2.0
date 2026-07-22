/**
 * 线下聚合收款服务。
 *
 * 创建收款码为高风险写操作，带 Idempotency-Key。
 * 前端不传 storeId（门店范围来自 token scope）；memberPhone 仅用于本次匹配，
 * 由服务端返回掩码会员信息，前端不留存原始手机号。
 */

import { get, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { CollectionOrder } from '@/types/models'

export interface CreateCollectionPayload {
  amountCent: number
  subject: string
  businessType: string
  expiresInSeconds: number
  /** 可选：用于精确匹配会员，仅本次使用。 */
  memberPhone?: string
}

export const collectionService = {
  create(payload: CreateCollectionPayload) {
    return post<CollectionOrder>(API_PATHS.collection.orders, payload, { idempotent: true })
  },
  detail(id: string | number) {
    return get<CollectionOrder>(API_PATHS.collection.order(id))
  },
  cancel(id: string | number) {
    return post<CollectionOrder>(API_PATHS.collection.cancel(id), undefined, { idempotent: true })
  },
  // 收款单列表（GET /store/offline-collection-orders）服务端未实现，故不提供 records 方法；记录页退化为空状态。
}
