import { del, get, patch, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { StoreBanner } from '@/types/models'

export const bannerService = {
  list: () => get<StoreBanner[]>(API_PATHS.banners.list),
  create: (body: Partial<StoreBanner>) => post<StoreBanner>(API_PATHS.banners.list, body, { idempotent: true }),
  update: (id: string | number, body: Partial<StoreBanner>) => patch<StoreBanner>(API_PATHS.banners.banner(id), body, { idempotent: true }),
  remove: (id: string | number) => del<void>(API_PATHS.banners.banner(id), { idempotent: true }),
}
