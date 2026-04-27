import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useQuery } from '@tanstack/react-query'

export type TeamResponse = components['schemas']['TeamResponse']
export type PaginationMeta = components['schemas']['PaginationMeta']
export type SubmissionResponse = components['schemas']['SubmissionResponse']

export function useTeamsList(q: string, page: number) {
  return useQuery({
    queryKey: ['teams', q, page],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams', {
        params: {
          query: {
            ...(q ? { q } : {}),
            page,
            per_page: 50,
          },
        },
      })
      if (error) throw error
      return data as { data?: TeamResponse[]; meta?: PaginationMeta }
    },
    staleTime: staleTime.user,
    placeholderData: (prev) => prev,
  })
}

export function useTeamProfile(id: string) {
  return useQuery({
    queryKey: ['team-profile', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
      return data as TeamResponse
    },
    staleTime: staleTime.user,
  })
}

export function useTeamSolves(id: string) {
  return useQuery({
    queryKey: ['team-solves', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/solves/{teamID}', {
        params: { path: { teamID: id } },
      })
      if (error) throw error
      return (data ?? []) as Array<{
        id?: string
        challenge_title?: string
        challenge_category?: string
        challenge_points?: number
        solved_at?: string
        username?: string
      }>
    },
    staleTime: staleTime.user,
  })
}

export function useTeamAwards(id: string) {
  return useQuery({
    queryKey: ['team-awards', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/awards/{teamID}', {
        params: { path: { teamID: id } },
      })
      if (error) throw error
      return (data ?? []) as Array<{
        id?: string
        value?: number
        description?: string
        created_at?: string
      }>
    },
    staleTime: staleTime.user,
  })
}

export function useTeamFails(id: string) {
  return useQuery({
    queryKey: ['team-fails', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/fails/{teamID}', {
        params: { path: { teamID: id }, query: { per_page: 100 } },
      })
      if (error) throw error
      return (data?.data ?? []) as SubmissionResponse[]
    },
    staleTime: 0,
    retry: false,
  })
}
