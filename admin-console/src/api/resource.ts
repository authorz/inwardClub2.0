/**
 * 通用 REST 资源工厂（公共服务）。
 *
 * 后台绝大多数模块都是「列表 + 详情 + 新增 + 编辑 + 状态切换」的同构 CRUD。
 * 本工厂用配置生成标准资源服务，避免每个模块复制粘贴一套请求代码。
 *
 * 高风险写操作通过 idempotent 标记统一注入 Idempotency-Key。
 */
import { http } from './http'
import type { ListQuery, ListResult } from './types'

export interface ResourceConfig {
  /** 列表 / 创建路径 */
  base: string
  /** 详情 / 更新 / 删除路径构造器；默认 `${base}/${id}` */
  itemPath?: (id: string) => string
  /** 写操作是否默认视为高风险（注入幂等键） */
  idempotentWrites?: boolean
  /** 更新使用的 HTTP 方法；服务端 catalog/activity/coupon-template 用 PUT，其余用 PATCH。默认 patch。 */
  updateMethod?: 'patch' | 'put'
}

export interface ResourceService<T, Q extends ListQuery = ListQuery> {
  list: (query?: Q) => Promise<ListResult<T>>
  get: (id: string) => Promise<T>
  create: (payload: Partial<T>) => Promise<T>
  update: (id: string, payload: Partial<T>) => Promise<T>
  remove: (id: string) => Promise<void>
  /** 对某资源执行子动作，如 publish / disable / assign-stores */
  action: <R = unknown>(path: string, payload?: unknown, idempotent?: boolean) => Promise<R>
}

export function createResource<T, Q extends ListQuery = ListQuery>(
  config: ResourceConfig,
): ResourceService<T, Q> {
  const itemPath = config.itemPath ?? ((id: string) => `${config.base}/${id}`)
  const idem = config.idempotentWrites ?? false
  const updateMethod = config.updateMethod ?? 'patch'

  return {
    list: (query) => http.getList<T>(config.base, query as Record<string, unknown> | undefined),
    get: (id) => http.get<T>(itemPath(id)),
    create: (payload) => http.post<T>(config.base, payload, { idempotent: idem }),
    update: (id, payload) => http[updateMethod]<T>(itemPath(id), payload, { idempotent: idem }),
    remove: (id) => http.delete<void>(itemPath(id), { idempotent: idem }),
    action: <R = unknown>(path: string, payload?: unknown, idempotent = idem) =>
      http.post<R>(path, payload, { idempotent }),
  }
}
