import { del, get, getPaged, post, put } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { CouponTemplate } from '@/types/models'

export const couponService = {
  list: (params?: PageQuery) => getPaged<CouponTemplate>(API_PATHS.coupons.templates, params),
  detail: (id: string | number) => get<CouponTemplate>(API_PATHS.coupons.template(id)),
  create: (body: Partial<CouponTemplate>) => post<CouponTemplate>(API_PATHS.coupons.templates, body, { idempotent: true }),
  update: (id: string | number, body: Partial<CouponTemplate>) => put<CouponTemplate>(API_PATHS.coupons.template(id), body, { idempotent: true }),
  remove: (id: string | number) => del<void>(API_PATHS.coupons.template(id), { idempotent: true }),
  publish: (id: string | number) => post<CouponTemplate>(API_PATHS.coupons.publishTemplate(id), undefined, { idempotent: true }),
  disable: (id: string | number) => post<CouponTemplate>(API_PATHS.coupons.disableTemplate(id), undefined, { idempotent: true }),
}
