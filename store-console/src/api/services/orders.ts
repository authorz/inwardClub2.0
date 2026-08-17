/**
 * 订单、点餐履约、支付与退款服务。
 * 退款、订单状态流转为高风险写操作，统一带 Idempotency-Key。
 */

import { get, getPaged, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type {
  FoodOrder,
  PaymentOrder,
  PaymentTransactionRecord,
  RefundOrder,
  StoreOrder,
} from '@/types/models'

export const orderService = {
  list(params?: PageQuery) {
    return getPaged<StoreOrder>(API_PATHS.orders.list, params)
  },
  detail(id: string | number) {
    return get<StoreOrder>(API_PATHS.orders.detail(id))
  },
  foodOrders(params?: PageQuery) {
    return getPaged<FoodOrder>(API_PATHS.orders.foodOrders, params)
  },
  foodOrderDetail(id: string | number) {
    return get<FoodOrder>(`${API_PATHS.orders.foodOrders}/${id}`)
  },
  /** 点餐订单状态流转（confirm/prepare/ready/complete/cancel）。 */
  foodAction(id: string | number, action: string, body?: unknown) {
    return post<FoodOrder>(API_PATHS.orders.foodAction(id, action), body, { idempotent: true })
  },
  /** 支付单列表（GET /store/payment-orders）。 */
  paymentOrders(params?: PageQuery) {
    return getPaged<PaymentOrder>(API_PATHS.orders.paymentOrders, params)
  },
  /** 支付单详情（GET /store/payment-orders/:id）。 */
  paymentOrderDetail(id: string | number) {
    return get<PaymentOrder>(API_PATHS.orders.paymentOrderDetail(id))
  },
  /** 支付流水列表（GET /store/payment-transactions）。服务端无流水详情端点，详情复用列表行。 */
  paymentTransactions(params?: PageQuery) {
    return getPaged<PaymentTransactionRecord>(API_PATHS.orders.paymentTransactions, params)
  },
  /** 退款单列表（GET /store/refund-orders，与 refunds() 为同一读模型的别名）。 */
  refundOrders(params?: PageQuery) {
    return getPaged<RefundOrder>(API_PATHS.orders.refundOrders, params)
  },
  /** 退款单列表（GET /store/refunds）。服务端无退款单详情端点，详情复用列表行。 */
  refunds(params?: PageQuery) {
    return getPaged<RefundOrder>(API_PATHS.orders.refunds, params)
  },
  /** 发起退款（POST /store/refunds，按支付单退款）。 */
  requestRefund(body: {
    paymentOrderId: string | number
    amountCent: number
    reason: string
    password: string
  }) {
    return post<unknown>(API_PATHS.orders.refunds, body, { idempotent: true })
  },
}
