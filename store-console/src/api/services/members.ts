/**
 * 会员详情与钱包流水服务。
 * 人工调账为高风险写操作，带 Idempotency-Key。
 */

import { get, getPaged, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { Member, WalletLedgerEntry } from '@/types/models'

export interface WalletAdjustmentPayload {
  assetType: string
  direction: 'credit' | 'debit'
  amount: number
  reason: string
}

export const memberService = {
  list(params?: PageQuery) {
    return getPaged<Member>(API_PATHS.members.list, params)
  },
  /** 按手机号片段查询可绑定会员，支持尾号搜索。 */
  lookupByPhone(phone: string) {
    return get<Member[]>(`${API_PATHS.members.lookup}?phone=${encodeURIComponent(phone)}`)
  },
  detail(id: string | number) {
    return get<Member>(API_PATHS.members.detail(id))
  },
  adjustWallet(id: string | number, body: WalletAdjustmentPayload) {
    return post<unknown>(API_PATHS.members.walletAdjustments(id), body, { idempotent: true })
  },
  walletLedger(params?: PageQuery) {
    return getPaged<WalletLedgerEntry>(API_PATHS.members.walletLedger, params)
  },
}
