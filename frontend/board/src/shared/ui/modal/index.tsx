import { cn } from '@/shared/lib/utils'
import { useEffect, useRef } from 'react'

export interface ModalProps {
  open: boolean
  onClose: () => void
  title?: string
  children: React.ReactNode
  className?: string
  /** Max width class, defaults to max-w-lg */
  maxWidth?: string
  'data-testid'?: string
}

export function Modal({
  open,
  onClose,
  title,
  children,
  className,
  maxWidth = 'max-w-lg',
  'data-testid': testId,
}: ModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const swipeStartY = useRef<number | null>(null)
  const swipeDeltaY = useRef(0)

  // Close on Esc
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  // Trap scroll
  useEffect(() => {
    if (open) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => {
      document.body.style.overflow = ''
    }
  }, [open])

  if (!open) return null

  // ---------------------------------------------------------------------------
  // Swipe-down to close (mobile UX - only on small screens / touch devices)
  // ---------------------------------------------------------------------------
  const handleTouchStart = (e: React.TouchEvent) => {
    swipeStartY.current = e.touches[0]?.clientY ?? null
    swipeDeltaY.current = 0
  }

  const handleTouchMove = (e: React.TouchEvent) => {
    if (swipeStartY.current === null) return
    const delta = (e.touches[0]?.clientY ?? 0) - swipeStartY.current
    if (delta > 0) {
      swipeDeltaY.current = delta
      // Apply visual drag feedback
      if (dialogRef.current) {
        dialogRef.current.style.transform = `translateY(${Math.min(delta, 120)}px)`
        dialogRef.current.style.transition = 'none'
      }
    }
  }

  const handleTouchEnd = () => {
    if (swipeDeltaY.current > 80) {
      onClose()
    } else {
      // Snap back
      if (dialogRef.current) {
        dialogRef.current.style.transform = ''
        dialogRef.current.style.transition = ''
      }
    }
    swipeStartY.current = null
    swipeDeltaY.current = 0
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      data-testid={testId}
      className="fixed inset-0 z-50 flex items-center justify-center"
    >
      {/* Overlay */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        aria-hidden="true"
        onClick={onClose}
      />

      {/* Panel - fullscreen on mobile, max-w on desktop */}
      <div
        ref={dialogRef}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
        className={cn(
          'relative z-10 flex flex-col bg-space-card border border-space-border shadow-xl',
          'w-full h-full sm:h-auto sm:rounded-[var(--radius-lg)] sm:border',
          maxWidth,
          'animate-in fade-in-0 zoom-in-95 duration-200',
          className,
        )}
      >
        {/* Swipe handle indicator - visible only on mobile */}
        <div className="flex justify-center pt-2 pb-1 sm:hidden shrink-0" aria-hidden="true">
          <div className="w-10 h-1 rounded-full bg-space-border-bright opacity-60" />
        </div>

        {/* Header */}
        {title && (
          <div className="flex items-center justify-between px-5 py-4 border-b border-space-border shrink-0">
            <h2
              className="text-lg font-semibold text-text-primary"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              {title}
            </h2>
            <button
              onClick={onClose}
              aria-label="Close modal"
              data-testid="modal-close"
              className="text-text-muted hover:text-text-primary transition-colors p-2 rounded hover:bg-space-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue min-w-[44px] min-h-[44px] flex items-center justify-center"
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.749.749 0 0 1 1.275.326.749.749 0 0 1-.215.734L9.06 8l3.22 3.22a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L8 9.06l-3.22 3.22a.751.751 0 0 1-1.042-.018.751.751 0 0 1-.018-1.042L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z" />
              </svg>
            </button>
          </div>
        )}

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-5">{children}</div>
      </div>
    </div>
  )
}
