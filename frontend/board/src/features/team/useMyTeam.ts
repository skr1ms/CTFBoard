import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useAuthStore } from '@/shared/stores/authStore'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

export type TeamWithMembers = components['schemas']['TeamWithMembersResponse']
export type TeamMember = components['schemas']['UserResponse']

export function useMyTeam() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return useQuery({
    queryKey: ['my-team'],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/my')
      if (error) throw error
      return data as TeamWithMembers
    },
    staleTime: staleTime.user,
    enabled: isAuthenticated,
  })
}

export function useTeamInvite(isCaptain: boolean) {
  return useQuery({
    queryKey: ['team-invite'],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/me/invite')
      if (error) throw error
      return data as components['schemas']['TeamInviteResponse']
    },
    enabled: isCaptain,
    staleTime: staleTime.user,
  })
}

export function useUpdateTeamName() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (name: string) => {
      const { data, error } = await api.PATCH('/teams/me', { body: { name } })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-team'] })
    },
    onError: () => {
      toast.error('Failed to update team name')
    },
  })
}

export function useRegenerateInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST('/teams/me/invite')
      if (error) throw error
      return data as components['schemas']['TeamInviteResponse']
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['team-invite'] })
    },
    onError: () => {
      toast.error('Failed to regenerate invite link')
    },
  })
}

export function useKickMember() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (userId: string) => {
      const { error } = await api.DELETE('/teams/members/{ID}', {
        params: { path: { ID: userId } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-team'] })
    },
    onError: () => {
      toast.error('Failed to kick member')
    },
  })
}

export function useTransferCaptain() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (newCaptainId: string) => {
      const { error } = await api.POST('/teams/transfer-captain', {
        body: { new_captain_id: newCaptainId },
      })
      if (error) throw error
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-team'] })
    },
    onError: () => {
      toast.error('Failed to transfer captain')
    },
  })
}

export function useLeaveTeam() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const { error } = await api.POST('/teams/leave')
      if (error) throw error
    },
    onSuccess: async () => {
      const { data: me } = await api.GET('/auth/me')
      if (me) useAuthStore.getState().setUser(me)
      qc.invalidateQueries({ queryKey: ['my-team'] })
      qc.invalidateQueries({ queryKey: ['challenges'] })
      qc.invalidateQueries({ queryKey: ['scoreboard'] })
    },
    onError: () => {
      toast.error('Failed to leave team')
    },
  })
}

export function useDisbandTeam() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/teams/me')
      if (error) throw error
    },
    onSuccess: async () => {
      const { data: me } = await api.GET('/auth/me')
      if (me) useAuthStore.getState().setUser(me)
      qc.invalidateQueries({ queryKey: ['my-team'] })
      qc.invalidateQueries({ queryKey: ['challenges'] })
      qc.invalidateQueries({ queryKey: ['scoreboard'] })
      qc.invalidateQueries({ queryKey: ['notifications'] })
    },
    onError: () => {
      toast.error('Failed to disband team')
    },
  })
}
