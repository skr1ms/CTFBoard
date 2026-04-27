import { cn } from '@/shared/lib/utils'

export interface StarRatingProps {
  value: number
  max?: number
  onChange?: (value: number) => void
  readOnly?: boolean
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeClasses = {
  sm: 'w-4 h-4',
  md: 'w-6 h-6',
  lg: 'w-8 h-8',
}

export function StarRating({
  value,
  max = 5,
  onChange,
  readOnly = false,
  size = 'md',
  className,
}: StarRatingProps) {
  return (
    <div
      className={cn('flex items-center gap-0.5', className)}
      role={readOnly ? 'img' : 'radiogroup'}
      aria-label={`Rating: ${value} out of ${max}`}
    >
      {Array.from({ length: max }, (_, i) => {
        const starValue = i + 1
        const filled = starValue <= value

        return (
          <button
            key={i}
            type="button"
            disabled={readOnly}
            role={readOnly ? undefined : 'radio'}
            aria-checked={readOnly ? undefined : starValue === value}
            aria-label={`${starValue} star${starValue !== 1 ? 's' : ''}`}
            onClick={() => onChange?.(starValue)}
            className={cn(
              sizeClasses[size],
              'transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue rounded-sm',
              readOnly ? 'cursor-default' : 'cursor-pointer hover:scale-110',
              filled ? 'text-amber-400' : 'text-space-border',
            )}
          >
            <svg viewBox="0 0 24 24" fill="currentColor" className="w-full h-full">
              <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
            </svg>
          </button>
        )
      })}
    </div>
  )
}
