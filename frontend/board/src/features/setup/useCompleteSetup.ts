import { type PublicConfigs } from '@/features/auth/usePublicConfigs'
import { api } from '@/shared/api/client'
import { env } from '@/shared/config/env'
import { useAuthStore } from '@/shared/stores/authStore'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { type SetupFormData } from './types'

interface SetupResponse {
  token: string
  user: {
    id: string
    username: string
    email: string
    role: string
  }
}

async function submitSetup(data: SetupFormData): Promise<SetupResponse> {
  const body = {
    ctf_name: data.ctf_name,
    ctf_description: data.ctf_description,
    mode: data.mode,
    max_team_size: parseInt(data.max_team_size, 10),
    challenge_visibility: data.challenge_visibility,
    score_visibility: data.score_visibility,
    account_visibility: data.account_visibility,
    registration_visibility: data.registration_visibility,
    email_verification_required: data.email_verification_required,
    admin_username: data.admin_username,
    admin_email: data.admin_email,
    admin_password: data.admin_password,
    start_time: data.start_time ? new Date(data.start_time).toISOString() : undefined,
    end_time: data.end_time ? new Date(data.end_time).toISOString() : undefined,
    freeze_time: data.freeze_time ? new Date(data.freeze_time).toISOString() : undefined,
    timezone: data.timezone,
  }

  const res = await fetch(`${env.apiBaseUrl}/setup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Setup-Token': data.setup_token },
    credentials: 'include',
    body: JSON.stringify(body),
  })

  const json = (await res.json()) as Record<string, unknown>

  if (!res.ok) {
    const msg = typeof json.message === 'string' ? json.message : 'Setup failed. Please try again.'
    throw new Error(msg)
  }

  return json as unknown as SetupResponse
}

export function useCompleteSetup() {
  const queryClient = useQueryClient()
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  const login = useAuthStore((s) => s.login)

  return useMutation({
    mutationFn: submitSetup,
    onSuccess: async (data) => {
      // Store the token so subsequent API calls are authenticated.
      setAccessToken(data.token)

      // Fetch the full MeResponse (ban_status, team_id, etc.) now that the
      // token is in the store and the API client will attach it.
      const { data: me } = await api.GET('/auth/me')
      if (me) {
        login(data.token, me)
      }

      // Optimistically mark setup as complete so SetupGuard does not bounce
      // back to /setup before the refetch lands.
      queryClient.setQueryData<PublicConfigs>(['configs', 'public'], (old) => ({
        ...(old ?? {}),
        setup_complete: true,
      }))
      await queryClient.invalidateQueries({ queryKey: ['configs', 'public'] })
    },
  })
}
