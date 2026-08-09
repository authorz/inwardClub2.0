/**
 * 本店商品、分类、库存、价格/支付方式覆盖服务。
 * 库存调整、发布/下架为高风险写操作，带 Idempotency-Key。
 */

import { del, get, getPaged, post, put } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { CatalogCategory, CatalogItem } from '@/types/models'

export const catalogService = {
  items(params?: PageQuery) {
    return getPaged<CatalogItem>(API_PATHS.catalog.items, params)
  },
  categories(params?: PageQuery) {
    return getPaged<CatalogCategory>(API_PATHS.catalog.categories, params)
  },
  category(id: string | number) {
    return get<CatalogCategory>(API_PATHS.catalog.category(id))
  },
  createCategory(body: Record<string, unknown>) {
    return post<CatalogCategory>(API_PATHS.catalog.categories, body, { idempotent: true })
  },
  updateCategory(id: string | number, body: Record<string, unknown>) {
    return put<CatalogCategory>(API_PATHS.catalog.category(id), body, { idempotent: true })
  },
  deleteCategory(id: string | number) {
    return del<void>(API_PATHS.catalog.category(id), { idempotent: true })
  },
  detail(id: string | number) {
    return get<CatalogItem>(API_PATHS.catalog.item(id))
  },
  create(body: Record<string, unknown>) {
    return post<CatalogItem>(API_PATHS.catalog.items, body, { idempotent: true })
  },
  update(id: string | number, body: Record<string, unknown>) {
    return put<CatalogItem>(API_PATHS.catalog.item(id), body, { idempotent: true })
  },
  remove(id: string | number) {
    return del<void>(API_PATHS.catalog.item(id), { idempotent: true })
  },
}
