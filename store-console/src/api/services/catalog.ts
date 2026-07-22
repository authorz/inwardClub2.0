/**
 * 本店商品、分类、库存、价格/支付方式覆盖服务。
 * 库存调整、发布/下架为高风险写操作，带 Idempotency-Key。
 */

import { getPaged, patch, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { CatalogCategory, CatalogItem } from '@/types/models'
import type { PayChannel } from '@/constants/enums'

export const catalogService = {
  items(params?: PageQuery) {
    return getPaged<CatalogItem>(API_PATHS.catalog.items, params)
  },
  globalItems(params?: PageQuery) {
    return getPaged<CatalogItem>(API_PATHS.catalog.globalItems, params)
  },
  categories(params?: PageQuery) {
    return getPaged<CatalogCategory>(API_PATHS.catalog.categories, params)
  },
  adoptGlobalItem(id: string | number) {
    return post<CatalogItem>(API_PATHS.catalog.adoptGlobalItem(id), undefined, { idempotent: true })
  },
  updateStock(id: string | number, stockQuantity: number) {
    return patch<CatalogItem>(API_PATHS.catalog.itemStock(id), { stockQuantity }, { idempotent: true })
  },
  updatePrice(id: string | number, priceCent: number) {
    return patch<CatalogItem>(API_PATHS.catalog.item(id), { priceCent })
  },
  updatePaymentRules(id: string | number, payChannels: PayChannel[]) {
    return patch<CatalogItem>(API_PATHS.catalog.itemPaymentRules(id), { payChannels })
  },
  publish(id: string | number) {
    return post<CatalogItem>(API_PATHS.catalog.publishItem(id), undefined, { idempotent: true })
  },
  unpublish(id: string | number) {
    return post<CatalogItem>(API_PATHS.catalog.unpublishItem(id), undefined, { idempotent: true })
  },
}
