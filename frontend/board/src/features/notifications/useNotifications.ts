import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useAuthStore } from '@/shared/stores/authStore'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

export type NotificationResponse = components['schemas']['NotificationResponse']
export type UserNotificationResponse = components['schemas']['UserNotificationResponse']

export function useGlobalNotifications() {
  return useQuery({
    queryKey: ['notifications', 'global'],
    queryFn: async () => {
      const { data, error } = await api.GET('/notifications')
      if (error) throw error
      return (data ?? []) as NotificationResponse[]
    },
    staleTime: staleTime.user,
  })
}

export function usePersonalNotifications() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const hydrating = useAuthStore((s) => s.hydrating)
  return useQuery({
    queryKey: ['notifications', 'personal'],
    queryFn: async () => {
      const { data, error } = await api.GET('/user/notifications')
      if (error) throw error
      return (data ?? []) as UserNotificationResponse[]
    },
    staleTime: staleTime.live,
    enabled: isAuthenticated && !hydrating,
  })
}

export function useMarkAsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.PATCH('/user/notifications/{ID}/read', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['notifications', 'personal'] })
    },
    onError: () => {
      toast.error('Failed to mark notification as read')
    },
  })
}

export function useUnreadCount() {
  const { data } = usePersonalNotifications()
  return data?.filter((n) => !n.is_read).length ?? 0
}
