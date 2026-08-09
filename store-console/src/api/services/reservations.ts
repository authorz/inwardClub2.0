/**
 * 当前门店仍在占座中的预约记录。
 */

import { getPaged, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { Reservation } from '@/types/models'

export const reservationService = {
  list(params?: PageQuery) {
    return getPaged<Reservation>(API_PATHS.reservations.list, params)
  },
  cancel(id: string | number) {
    return post<void>(API_PATHS.reservations.cancel(id), undefined, { idempotent: true })
  },
}
