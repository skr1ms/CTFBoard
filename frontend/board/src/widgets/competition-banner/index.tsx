import { useCompetitionStatus } from '@/features/competition/useCompetitionStatus'

type StatusVariant = 'blue' | 'green' | 'yellow' | 'orange' | 'gray'

interface BannerConfig {
  variant: StatusVariant
  icon: string
  message: (status: ReturnType<typeof useCompetitionStatus>['data']) => string
  show: boolean
}

const variantClasses: Record<StatusVariant, string> = {
  blue: 'bg-cosmic-blue/10 border-cosmic-blue/40 text-cosmic-blue',
  green: 'bg-stellar-green/10 border-stellar-green/40 text-stellar-green',
  yellow: 'bg-solar-yellow/10 border-solar-yellow/40 text-solar-yellow',
  orange: 'bg-orange-500/10 border-orange-500/40 text-orange-400',
  gray: 'bg-space-border/30 border-space-border text-text-muted',
}

function formatDate(iso?: string): string {
  if (!iso) return ''
  return new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

// eslint-disable-next-line react-refresh/only-export-components
export function getBannerConfig(
  data: ReturnType<typeof useCompetitionStatus>['data'] | null,
): BannerConfig | null {
  if (!data) return null
  const status = data.status ?? ''

  switch (status) {
    case 'not_started':
      return {
        variant: 'blue',
        icon: '🕐',
        message: (d) =>
          d?.start_time
            ? `Competition starts ${formatDate(d.start_time)}`
            : 'Competition has not started yet',
        show: true,
      }

    case 'active':
      // Show freeze sub-status if scoreboard is frozen
      if (data.freeze_time && new Date(data.freeze_time) <= new Date()) {
        return {
          variant: 'orange',
          icon: '🧊',
          message: () => 'Scoreboard is frozen',
          show: true,
        }
      }
      // Active + not frozen = no banner (competition running normally)
      return null

    case 'paused':
      return {
        variant: 'yellow',
        icon: '⏸',
        message: () => 'Competition is paused',
        show: true,
      }

    case 'frozen':
      return {
        variant: 'orange',
        icon: '🧊',
        message: () => 'Scoreboard is frozen',
        show: true,
      }

    case 'ended':
      return {
        variant: 'gray',
        icon: '🏁',
        message: (d) =>
          d?.end_time ? `Competition ended ${formatDate(d.end_time)}` : 'Competition has ended',
        show: true,
      }

    default:
      return null
  }
}

export function CompetitionBanner() {
  const { data } = useCompetitionStatus()
  const config = getBannerConfig(data)

  if (!config || !config.show) return null

  return (
    <div
      role="status"
      className={`
        w-full border-b px-4 py-2
        flex items-center justify-center gap-2
        text-sm font-medium
        ${variantClasses[config.variant]}
      `}
    >
      <span aria-hidden="true">{config.icon}</span>
      <span>{config.message(data)}</span>
    </div>
  )
}
