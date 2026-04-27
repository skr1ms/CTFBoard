import { api, type ApiError } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useAuthStore } from '@/shared/stores/authStore'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

type BanAppeal = components['schemas']['BanAppealResponse']

export function useMyAppeals() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return useQuery<BanAppeal[]>({
    queryKey: ['appeals', 'me'],
    queryFn: async () => {
      const { data, error } = await api.GET('/appeals/me')
      if (error) throw error
      return data?.appeals ?? []
    },
    staleTime: 30_000,
    enabled: isAuthenticated,
  })
}

export function useCreateAppeal() {
  const qc = useQueryClient()

  return useMutation({
    mutationFn: async (message: string) => {
      const { data, error } = await api.POST('/appeals', { body: { message } })
      if (error) throw error
      return data
    },
    onSuccess() {
      void qc.invalidateQueries({ queryKey: ['appeals', 'me'] })
      toast.success('Appeal submitted successfully')
    },
    onError(err: ApiError) {
      toast.error(err.message ?? 'Failed to submit appeal')
    },
  })
}
