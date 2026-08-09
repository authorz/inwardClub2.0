/**
 * 本店活动服务（列表、发布/下架、采用全局活动、今日核销概览）。
 */

import { del, get, getPaged, post, put } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { ActivityTicketType, StoreActivity } from '@/types/models'

export const activityService = {
  list(params?: PageQuery) {
    return getPaged<StoreActivity>(API_PATHS.activities.list, params)
  },
  detail(id: string | number) {
    return get<StoreActivity>(API_PATHS.activities.detail(id))
  },
  create(body: Record<string, unknown>) {
    return post<StoreActivity>(API_PATHS.activities.list, body, { idempotent: true })
  },
  update(id: string | number, body: Record<string, unknown>) {
    return put<StoreActivity>(API_PATHS.activities.detail(id), body, { idempotent: true })
  },
  remove(id: string | number) {
    return del<void>(API_PATHS.activities.detail(id), { idempotent: true })
  },
  ticketTypes(activityId: string | number) {
    return get<ActivityTicketType[]>(API_PATHS.activities.ticketTypes(activityId))
  },
  createTicketType(activityId: string | number, body: Record<string, unknown>) {
    return post<ActivityTicketType>(API_PATHS.activities.ticketTypes(activityId), body, { idempotent: true })
  },
  updateTicketType(activityId: string | number, id: string | number, body: Record<string, unknown>) {
    return put<ActivityTicketType>(API_PATHS.activities.ticketType(activityId, id), body, { idempotent: true })
  },
  removeTicketType(activityId: string | number, id: string | number) {
    return del<void>(API_PATHS.activities.ticketType(activityId, id), { idempotent: true })
  },
  /** 今日活动/核销概览。 */
  today() {
    return get<StoreActivity[]>(API_PATHS.activities.today)
  },
}
