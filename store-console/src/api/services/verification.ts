/**
 * 核销服务：活动票核销、券核销/作废、核销记录查询。
 * 所有核销/作废为高风险写操作，必须带 Idempotency-Key。
 *
 * 说明：服务端无「待核销活动票 / 待核销券实例」的列表读端点
 * （仅 POST /store/tickets/verify、POST /store/coupon-verifications、POST /store/coupon-voids），
 * 因此本服务不提供列表方法，核销页以扫码即核销的方式工作。
 */

import { getPaged, post } from '../request'
import { getAuthBridge } from '../authBridge'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { VerificationRecord } from '@/types/models'

export const verificationService = {
  /** 通过核销码核销活动票。 */
  verifyTicket(code: string) {
    return post<VerificationRecord>(API_PATHS.tickets.verify, { code }, { idempotent: true })
  },
  /**
   * 核销券实例（POST /store/coupon-verifications）。storeId 从当前 token scope 取（服务端必填）；
   * 券以扫码得到的券号 entitlementNo 或券 id entitlementId 定位。
   */
  verifyCoupon(payload: { entitlementNo?: string; entitlementId?: number }) {
    const storeId = getAuthBridge()?.getStoreId()
    return post<unknown>(
      API_PATHS.coupons.verify,
      { ...payload, storeId: storeId != null ? Number(storeId) : undefined },
      { idempotent: true },
    )
  },
  /** 作废券实例（POST /store/coupon-voids）。 */
  voidCoupon(entitlementId: number, reason: string) {
    return post<unknown>(API_PATHS.coupons.void, { entitlementId, reason }, { idempotent: true })
  },
  /** 核销记录（票+券统一读模型）。 */
  records(params?: PageQuery) {
    return getPaged<VerificationRecord>(API_PATHS.verifications.list, params)
  },
}
