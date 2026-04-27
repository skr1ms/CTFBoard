import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useQuery } from '@tanstack/react-query'

export type UserResponse = components['schemas']['UserResponse']
export type UserProfileResponse = components['schemas']['UserProfileResponse']
export type SolveWithDetails = components['schemas']['SolveWithDetailsResponse']
export type AwardResponse = components['schemas']['AwardResponse']
export type PaginationMeta = components['schemas']['PaginationMeta']
export type SubmissionResponse = components['schemas']['SubmissionResponse']

export function useUsers(q: string, page: number) {
  return useQuery({
    queryKey: ['users', q, page],
    queryFn: async () => {
      const { data, error } = await api.GET('/users', {
        params: {
          query: {
            ...(q ? { q } : {}),
            page,
            per_page: 50,
          },
        },
      })
      if (error) throw error
      return data as { data?: UserResponse[]; meta?: PaginationMeta }
    },
    staleTime: staleTime.user,
    placeholderData: (prev) => prev,
  })
}

export function useUserProfile(id: string) {
  return useQuery({
    queryKey: ['user-profile', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/users/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
      return data as UserProfileResponse
    },
    staleTime: staleTime.user,
  })
}

export function useUserSolves(id: string) {
  return useQuery({
    queryKey: ['user-solves', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/users/{ID}/solves', {
        params: { path: { ID: id } },
      })
      if (error) throw error
      return (data ?? []) as SolveWithDetails[]
    },
    staleTime: staleTime.user,
  })
}

export function useUserAwards(id: string) {
  return useQuery({
    queryKey: ['user-awards', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/users/{ID}/awards', {
        params: { path: { ID: id } },
      })
      if (error) throw error
      return (data ?? []) as AwardResponse[]
    },
    staleTime: staleTime.user,
  })
}

export function useUserFails(id: string) {
  return useQuery({
    queryKey: ['user-fails', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/users/{ID}/fails', {
        params: { path: { ID: id }, query: { per_page: 100 } },
      })
      if (error) throw error
      return (data?.data ?? []) as SubmissionResponse[]
    },
    staleTime: 0,
    retry: false,
  })
}
