/**
 * 认证桥接层：打破 http 客户端与 auth store 之间的循环依赖。
 *
 * auth store 在初始化时注册这些处理器；http 拦截器在需要刷新 token 或
 * 判定登录态失效时通过桥接调用，而不直接 import store，避免循环引用。
 */

export interface AuthBridge {
  /** 取当前 access token。 */
  getAccessToken(): string | null
  /** 当前门店 ID（来自 token scope / me），用于跨店数据兜底校验。 */
  getStoreId(): string | number | null
  /** 执行 refresh（含 audience/store scope 校验），成功返回 true。 */
  refresh(): Promise<boolean>
  /** 登录态失效：清空并跳转登录页。reason 用于提示与埋点。 */
  invalidate(reason: string): void
}

let bridge: AuthBridge | null = null

export function registerAuthBridge(impl: AuthBridge): void {
  bridge = impl
}

export function getAuthBridge(): AuthBridge | null {
  return bridge
}
