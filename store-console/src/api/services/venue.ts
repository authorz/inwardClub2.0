import { del, get, getPaged, patch, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { StoreSeat, StoreTable } from '@/types/models'

export const venueService = {
  tables: (params?: PageQuery) => getPaged<StoreTable>(API_PATHS.tables.list, params),
  table: (id: string | number) => get<StoreTable>(API_PATHS.tables.table(id)),
  createTable: (body: Partial<StoreTable>) => post<StoreTable>(API_PATHS.tables.list, body, { idempotent: true }),
  updateTable: (id: string | number, body: Partial<StoreTable>) => patch<StoreTable>(API_PATHS.tables.table(id), body, { idempotent: true }),
  deleteTable: (id: string | number) => del<void>(API_PATHS.tables.table(id), { idempotent: true }),
  seats: (params?: PageQuery) => getPaged<StoreSeat>(API_PATHS.tables.seats, params),
  createSeat: (body: Partial<StoreSeat>) => post<StoreSeat>(API_PATHS.tables.seats, body, { idempotent: true }),
  updateSeat: (id: string | number, body: Partial<StoreSeat>) => patch<StoreSeat>(API_PATHS.tables.seat(id), body, { idempotent: true }),
  deleteSeat: (id: string | number) => del<void>(API_PATHS.tables.seat(id), { idempotent: true }),
}
