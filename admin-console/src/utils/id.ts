/**
 * 统一 ID 生成工具。
 * - requestId：每个请求的 X-Request-ID，用于链路追踪与审计关联。
 * - idempotencyKey：高风险写操作（退款、调账、发券、发布、上下架、库存、支付配置）的幂等键。
 * 页面不得自行拼接 header，只能通过此工具生成。
 */
function uuid(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  // 退化实现（非加密安全，仅用于极旧环境的兜底）
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

export function newRequestId(): string {
  return `adm-${uuid()}`
}

export function newIdempotencyKey(): string {
  return `idem-${uuid()}`
}
