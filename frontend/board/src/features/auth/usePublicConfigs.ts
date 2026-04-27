import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import { useQuery } from '@tanstack/react-query'

export type Visibility = 'public' | 'private' | 'hidden' | 'admins'

export interface PublicConfigs {
  oauth_github_enabled?: boolean
  oauth_google_enabled?: boolean
  registration_enabled?: boolean
  challenge_visibility?: Visibility
  score_visibility?: Visibility
  account_visibility?: Visibility
  registration_visibility?: Visibility
  setup_complete?: boolean
  email_verification_required?: boolean
  timezone?: string
  [key: string]: unknown
}

export function usePublicConfigs() {
  return useQuery({
    queryKey: ['configs', 'public'],
    queryFn: async () => {
      const { data, error } = await api.GET('/configs/public')
      if (error) throw error
      return (data ?? {}) as PublicConfigs
    },
    staleTime: staleTime.static,
  })
}
