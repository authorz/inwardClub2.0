/**
 * 本店活动服务（列表、发布/下架、采用全局活动、今日核销概览）。
 */

import { get, getPaged, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { StoreActivity } from '@/types/models'

export const activityService = {
  list(params?: PageQuery) {
    return getPaged<StoreActivity>(API_PATHS.activities.list, params)
  },
  globalActivities(params?: PageQuery) {
    return getPaged<StoreActivity>(API_PATHS.activities.globalActivities, params)
  },
  detail(id: string | number) {
    return get<StoreActivity>(API_PATHS.activities.detail(id))
  },
  adoptGlobal(id: string | number) {
    return post<StoreActivity>(API_PATHS.activities.adoptGlobal(id), undefined, { idempotent: true })
  },
  publish(id: string | number) {
    return post<StoreActivity>(API_PATHS.activities.publish(id), undefined, { idempotent: true })
  },
  unpublish(id: string | number) {
    return post<StoreActivity>(API_PATHS.activities.unpublish(id), undefined, { idempotent: true })
  },
  /** 今日活动/核销概览。 */
  today() {
    return get<StoreActivity[]>(API_PATHS.activities.today)
  },
}
