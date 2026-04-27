import { useCopyToClipboard } from '@/shared/lib/hooks/useCopyToClipboard'
import { cn } from '@/shared/lib/utils'

export interface CodeBlockProps {
  code: string
  language?: string
  className?: string
  showCopy?: boolean
}

export function CodeBlock({ code, language, className, showCopy = true }: CodeBlockProps) {
  const { copy, copied } = useCopyToClipboard()

  return (
    <div
      className={cn(
        'relative group rounded-[var(--radius-md)] border border-space-border bg-space-dark overflow-hidden',
        className,
      )}
    >
      {(language ?? showCopy) && (
        <div className="flex items-center justify-between px-3 py-1.5 border-b border-space-border bg-space-card/50">
          {language && <span className="text-xs text-text-muted font-mono">{language}</span>}
          {showCopy && (
            <button
              onClick={() => copy(code)}
              aria-label="Copy code"
              className="ml-auto text-xs text-text-muted hover:text-text-primary transition-colors px-2 py-0.5 rounded hover:bg-space-border focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue"
            >
              {copied ? '✓ Copied' : 'Copy'}
            </button>
          )}
        </div>
      )}
      <pre
        className="overflow-x-auto p-4 text-sm text-text-primary"
        style={{ fontFamily: 'var(--font-mono)' }}
      >
        <code>{code}</code>
      </pre>
    </div>
  )
}
