/**
 * 门店后台运行时配置。
 *
 * 所有环境变量集中在此读取并做校验，页面/服务层不得直接读 import.meta.env。
 * 这样可以保证 API base、audience、appId 在整个站点内唯一来源。
 */

const rawBaseUrl = import.meta.env.VITE_API_BASE_URL ?? ''

if (!rawBaseUrl) {
  // 缺失 base url 会导致所有请求打到当前站点，属于配置错误，尽早暴露。
  console.error('[store-console] VITE_API_BASE_URL 未配置，API 请求将失败。')
}

/** 去除结尾斜杠，统一 base 拼接行为。 */
function normalizeBaseUrl(url: string): string {
  return url.replace(/\/+$/, '')
}

export const appConfig = {
  /** 应用标识，用于错误监控 / 埋点区分门店后台。 */
  appId: import.meta.env.VITE_APP_ID ?? 'inwardclub-store-console',
  /** API 根地址，必须指向 /api/v2 网关。 */
  apiBaseUrl: normalizeBaseUrl(rawBaseUrl),
  /** token audience，门店后台固定为 store。 */
  authAudience: import.meta.env.VITE_AUTH_AUDIENCE ?? 'store',
  /** 资产公共访问域名（可选），仅用于展示 assetId 对应图片。 */
  assetPublicDomain: (import.meta.env.VITE_ASSET_PUBLIC_DOMAIN ?? '').replace(/\/+$/, ''),
} as const

/**
 * 门店后台只允许访问的 API 前缀白名单。
 * 任何超出该白名单的请求（尤其 /admin/*）都会在 http 层被拦截。
 */
export const ALLOWED_API_PREFIXES = ['/store/', '/store', '/assets/', '/internal/assets/'] as const

/** 严禁调用的前缀（总后台接口）。命中即视为越权。 */
export const FORBIDDEN_API_PREFIXES = ['/admin/', '/admin'] as const
