import { api, isApiError } from '@/shared/api/client'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

export function useHintUnlock(challengeId: string) {
  const qc = useQueryClient()

  return useMutation({
    mutationFn: async (hintId: string) => {
      const { data, error } = await api.POST('/challenges/{challengeID}/hints/{hintID}/unlock', {
        params: { path: { challengeID: challengeId, hintID: hintId } },
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['challenge', challengeId] })
      qc.invalidateQueries({ queryKey: ['scoreboard'] })
      qc.invalidateQueries({ queryKey: ['my-team'] })
      toast.success('Hint unlocked')
    },
    onError: (err: unknown) => {
      const code = isApiError(err) ? err.code : undefined
      if (code === 'INSUFFICIENT_POINTS') {
        toast.error('Not enough points to unlock this hint')
      } else {
        toast.error('Failed to unlock hint')
      }
    },
  })
}
