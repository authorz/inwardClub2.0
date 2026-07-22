/**
 * 请求标识生成：X-Request-ID 与 Idempotency-Key。
 *
 * 高风险写操作（创建收款码、退款、核销、发券、库存调整、人工调账、订单状态流转）
 * 必须携带 Idempotency-Key，且同一次用户操作在重试时保持相同的 key。
 */

/** 生成 UUID v4，优先使用原生 crypto。 */
export function uuid(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  // 退化实现
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

export function newRequestId(): string {
  return uuid()
}

export function newIdempotencyKey(): string {
  return uuid()
}
