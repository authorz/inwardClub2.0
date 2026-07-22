/**
 * 预约与桌台服务。
 * 到店确认为状态流转写操作，带 Idempotency-Key。
 */

import { getPaged, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { Reservation } from '@/types/models'

// 桌台列表（GET /store/tables）服务端未实现，故不提供 tables 方法；桌台页退化为空状态。
export const reservationService = {
  list(params?: PageQuery) {
    return getPaged<Reservation>(API_PATHS.reservations.list, params)
  },
  arrive(id: string | number) {
    return post<Reservation>(API_PATHS.reservations.arrive(id), undefined, { idempotent: true })
  },
}
