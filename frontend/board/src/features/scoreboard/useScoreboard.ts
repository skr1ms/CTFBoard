import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useQuery } from '@tanstack/react-query'

export type ScoreboardEntry = components['schemas']['ScoreboardEntryResponse']
export type BracketResponse = components['schemas']['BracketResponse']
export type TeamTimeline = components['schemas']['TeamTimeline']
export type ScorePoint = components['schemas']['ScorePoint']

export function useScoreboard(bracketId?: string) {
  return useQuery({
    queryKey: ['scoreboard', bracketId ?? 'all'],
    queryFn: async () => {
      const { data, error } = await api.GET('/scoreboard', {
        ...(bracketId ? { params: { query: { bracket: bracketId } } } : {}),
      })
      if (error) throw error
      return (data ?? []) as ScoreboardEntry[]
    },
    staleTime: staleTime.live,
  })
}

export function useBrackets() {
  return useQuery({
    queryKey: ['brackets'],
    queryFn: async () => {
      const { data, error } = await api.GET('/brackets')
      if (error) throw error
      return (data ?? []) as BracketResponse[]
    },
    staleTime: staleTime.static,
  })
}

export function useScoreboardGraph(top = 10) {
  return useQuery({
    queryKey: ['scoreboard-graph', top],
    queryFn: async () => {
      const { data, error } = await api.GET('/scoreboard/graph', {
        params: { query: { top } },
      })
      if (error) throw error
      return data as { range?: { start?: string; end?: string }; teams?: TeamTimeline[] }
    },
    staleTime: staleTime.live,
  })
}
