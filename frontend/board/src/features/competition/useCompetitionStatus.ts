import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useQuery } from '@tanstack/react-query'

export type CompetitionStatus = components['schemas']['CompetitionStatusResponse']

export function useCompetitionStatus() {
  return useQuery({
    queryKey: ['competition-status'],
    queryFn: async () => {
      const { data, error } = await api.GET('/competition/status')
      if (error) throw error
      return data as CompetitionStatus
    },
    staleTime: staleTime.live,
    refetchInterval: 60_000,
  })
}
