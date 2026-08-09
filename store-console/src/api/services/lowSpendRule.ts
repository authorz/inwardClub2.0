import { get, put } from '../request'
import { API_PATHS } from '@/constants/apiPaths'

export interface StoreLowSpendRule {
  storeId: number
  storeName: string
  configured: boolean
  enabled: boolean
  reservationCutoff: string
  consumptionCutoff: string
  minimumAmount: number
  rewardPoints: number
  updatedAt?: string
}

export type StoreLowSpendRuleInput = Pick<
  StoreLowSpendRule,
  'enabled' | 'reservationCutoff' | 'consumptionCutoff' | 'minimumAmount' | 'rewardPoints'
>

export const lowSpendRuleService = {
  get() {
    return get<StoreLowSpendRule>(API_PATHS.lowSpendRule.get)
  },
  update(body: StoreLowSpendRuleInput) {
    return put<StoreLowSpendRule>(API_PATHS.lowSpendRule.update, body, { idempotent: true })
  },
}
