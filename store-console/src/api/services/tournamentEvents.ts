import { del, get, getPaged, post, put } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { PageQuery } from '@/types/api'
import type { TournamentEvent } from '@/types/models'

export const tournamentEventService = {
  list(params?: PageQuery) {
    return getPaged<TournamentEvent>(API_PATHS.tournamentEvents.list, params)
  },
  detail(id: string | number) {
    return get<TournamentEvent>(API_PATHS.tournamentEvents.detail(id))
  },
  create(body: Record<string, unknown>) {
    return post<TournamentEvent>(API_PATHS.tournamentEvents.list, body, { idempotent: true })
  },
  update(id: string | number, body: Record<string, unknown>) {
    return put<TournamentEvent>(API_PATHS.tournamentEvents.detail(id), body, { idempotent: true })
  },
  remove(id: string | number) {
    return del<void>(API_PATHS.tournamentEvents.detail(id), { idempotent: true })
  },
}
