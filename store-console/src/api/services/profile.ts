/**
 * 门店资料与设置服务。
 * 资料/状态修改带审计；资产只提交 assetId。
 */

import { get, patch } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { StoreScope } from '@/types/auth'

export interface StoreProfile extends StoreScope {
  address?: string
  phone?: string
  customerServiceQrAssetId?: string | number | null
  customerServiceQrUrl?: string
  businessHours?: string
  /** GPS 坐标：小程序据此计算门店距离并支持「导航前往」，缺失则两者都不可用 */
  latitude?: number | null
  longitude?: number | null
  logoAssetId?: number | null
  status?: 'open' | 'closed'
  statusMode?: 'auto' | 'manual'
  scheduledOpen?: boolean
  statusOverrideUntil?: string | null
}

export const profileService = {
  get() {
    return get<StoreProfile>(API_PATHS.profile.get)
  },
  update(body: Partial<StoreProfile>) {
    return patch<StoreProfile>(API_PATHS.profile.update, body, { idempotent: true })
  },
  updateStatus(status: 'open' | 'closed' | 'auto') {
    return patch<StoreProfile>(API_PATHS.profile.status, { status }, { idempotent: true })
  },
}
