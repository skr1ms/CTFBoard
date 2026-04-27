import { cn } from '@/shared/lib/utils'

export interface AvatarProps {
  src?: string | null
  alt?: string
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  className?: string
}

const sizes = {
  xs: 'w-6 h-6 text-xs',
  sm: 'w-8 h-8 text-xs',
  md: 'w-10 h-10 text-sm',
  lg: 'w-14 h-14 text-base',
  xl: 'w-20 h-20 text-xl',
}

function initials(alt?: string): string {
  if (!alt) return '?'
  return alt
    .split(' ')
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')
}

export function Avatar({ src, alt, size = 'md', className }: AvatarProps) {
  return (
    <div
      className={cn(
        'rounded-full overflow-hidden shrink-0 inline-flex items-center justify-center',
        'bg-space-border text-text-secondary font-semibold select-none',
        sizes[size],
        className,
      )}
      aria-label={alt}
    >
      {src ? (
        <img src={src} alt={alt ?? ''} className="w-full h-full object-cover" />
      ) : (
        <span>{initials(alt)}</span>
      )}
    </div>
  )
}
