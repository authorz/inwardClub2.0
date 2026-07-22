/**
 * 收银员与员工账号服务。
 * 停用、重置密码为高风险写操作，带 Idempotency-Key。
 */

import { getPaged, patch, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { Cashier, StaffAccount } from '@/types/models'

/** 新增收银员：服务端仅接受登录用户名与显示名，初始密码由服务端生成并一次性返回。 */
export interface CashierCreatePayload {
  username: string
  displayName: string
}

/** 编辑收银员：仅显示名可改，用户名/角色/门店在创建时固定。 */
export interface CashierUpdatePayload {
  displayName: string
}

/** 新增/编辑员工账号：服务端仅接受姓名。 */
export interface StaffAccountPayload {
  name: string
}

export const cashierService = {
  list(params?: PageQuery) {
    return getPaged<Cashier>(API_PATHS.staff.cashiers, params)
  },
  create(body: CashierCreatePayload) {
    return post<Cashier>(API_PATHS.staff.cashiers, body, { idempotent: true })
  },
  update(id: string | number, body: CashierUpdatePayload) {
    return patch<Cashier>(API_PATHS.staff.cashier(id), body, { idempotent: true })
  },
  disable(id: string | number) {
    return post<Cashier>(API_PATHS.staff.cashierDisable(id), undefined, { idempotent: true })
  },
  /** 重置密码：服务端生成新密码并在响应 initialPassword 中一次性返回，不接受入参密码。 */
  resetPassword(id: string | number) {
    return post<Cashier>(API_PATHS.staff.cashierPasswordReset(id), undefined, { idempotent: true })
  },
}

export const staffAccountService = {
  list(params?: PageQuery) {
    return getPaged<StaffAccount>(API_PATHS.staff.staffAccounts, params)
  },
  create(body: StaffAccountPayload) {
    return post<StaffAccount>(API_PATHS.staff.staffAccounts, body, { idempotent: true })
  },
  update(id: string | number, body: StaffAccountPayload) {
    return patch<StaffAccount>(API_PATHS.staff.staffAccount(id), body, { idempotent: true })
  },
  disable(id: string | number) {
    return post<StaffAccount>(API_PATHS.staff.staffDisable(id), undefined, { idempotent: true })
  },
}
