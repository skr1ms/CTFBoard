import { staleTime } from '@/app/providers/QueryProvider'
import { env } from '@/shared/config/env'
import { useQuery } from '@tanstack/react-query'

interface OAuthProviders {
  github: boolean
  google: boolean
}

async function fetchOAuthProviders(): Promise<OAuthProviders> {
  const res = await fetch(`${env.apiBaseUrl}/auth/oauth/providers`)
  if (!res.ok) return { github: false, google: false }
  return res.json() as Promise<OAuthProviders>
}

export function useOAuthProviders() {
  return useQuery({
    queryKey: ['auth', 'oauth', 'providers'],
    queryFn: fetchOAuthProviders,
    staleTime: staleTime.static,
  })
}
