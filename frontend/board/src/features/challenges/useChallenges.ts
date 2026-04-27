import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useQuery, useQueryClient } from '@tanstack/react-query'

export type ChallengeResponse = components['schemas']['ChallengeResponse']
export type ChallengeDetail = NonNullable<Awaited<ReturnType<typeof fetchChallengeDetail>>>
export type HintItem = components['schemas']['HintItem']
export type FileItem = NonNullable<components['schemas']['FileItem']>
export type ChallengeRequirement = components['schemas']['ChallengeRequirementResponse']
export type TagResponse = components['schemas']['TagResponse']

async function fetchChallengeDetail(id: string) {
  const { data, error } = await api.GET('/challenges/{challengeID}', {
    params: { path: { challengeID: id } },
  })
  if (error) throw error
  return data
}

export function useChallenges(tagId?: string) {
  return useQuery({
    queryKey: ['challenges', tagId ?? 'all'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges', {
        ...(tagId ? { params: { query: { tag: tagId } } } : {}),
      })
      if (error) throw error
      return (data ?? []) as ChallengeResponse[]
    },
    staleTime: staleTime.live,
  })
}

export function useChallengeDetail(id: string | null) {
  return useQuery({
    queryKey: ['challenge', id],
    queryFn: () => fetchChallengeDetail(id!),
    enabled: !!id,
    staleTime: staleTime.live,
  })
}

export function useChallengeRequirements(id: string | null) {
  return useQuery({
    queryKey: ['challenge-requirements', id],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}/requirements', {
        params: { path: { challengeID: id! } },
      })
      if (error) throw error
      return (data ?? []) as ChallengeRequirement[]
    },
    enabled: !!id,
    staleTime: staleTime.live,
  })
}

export function useTags() {
  return useQuery({
    queryKey: ['tags'],
    queryFn: async () => {
      const { data, error } = await api.GET('/tags')
      if (error) throw error
      return (data ?? []) as TagResponse[]
    },
    staleTime: staleTime.static,
  })
}

export function useChallengeTypes() {
  return useQuery({
    queryKey: ['challenge-types'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/types')
      if (error) throw error
      return (data ?? []) as string[]
    },
    staleTime: Infinity,
  })
}

export function useInvalidateChallenges() {
  const qc = useQueryClient()
  return () => {
    qc.invalidateQueries({ queryKey: ['challenges'] })
    qc.invalidateQueries({ queryKey: ['challenge'] })
  }
}
