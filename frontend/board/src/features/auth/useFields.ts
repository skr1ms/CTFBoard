import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useQuery } from '@tanstack/react-query'

export type FieldResponse = components['schemas']['FieldResponse']

export function useUserFields() {
  return useQuery({
    queryKey: ['fields', 'user'],
    queryFn: async () => {
      const { data, error } = await api.GET('/fields', {
        params: { query: { entity_type: 'user' } },
      })
      if (error) throw error
      return (data ?? []) as FieldResponse[]
    },
    staleTime: staleTime.static,
  })
}
