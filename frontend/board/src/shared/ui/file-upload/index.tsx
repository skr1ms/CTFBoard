import { cn } from '@/shared/lib/utils'
import { useRef, useState } from 'react'

export interface FileUploadProps {
  onFile: (file: File) => void
  accept?: string
  maxSizeMb?: number
  label?: string
  className?: string
  disabled?: boolean
}

export function FileUpload({
  onFile,
  accept,
  maxSizeMb = 50,
  label = 'Drag & drop or click to upload',
  className,
  disabled = false,
}: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const validate = (file: File): string | null => {
    if (file.size > maxSizeMb * 1024 * 1024) return `File exceeds ${maxSizeMb} MB limit`
    return null
  }

  const handle = (file: File) => {
    const err = validate(file)
    if (err) {
      setError(err)
      return
    }
    setError(null)
    onFile(file)
  }

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragging(false)
    if (disabled) return
    const file = e.dataTransfer.files[0]
    if (file) handle(file)
  }

  const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) handle(file)
    e.target.value = ''
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div
        role="button"
        tabIndex={disabled ? -1 : 0}
        aria-disabled={disabled}
        onClick={() => !disabled && inputRef.current?.click()}
        onKeyDown={(e) => e.key === 'Enter' && !disabled && inputRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault()
          if (!disabled) setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        className={cn(
          'flex flex-col items-center justify-center gap-2 p-6 rounded-[var(--radius-md)]',
          'border-2 border-dashed transition-colors cursor-pointer select-none',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue',
          dragging
            ? 'border-cosmic-blue bg-cosmic-blue/10'
            : 'border-space-border hover:border-cosmic-blue/50 hover:bg-space-card',
          disabled && 'opacity-50 pointer-events-none',
          className,
        )}
      >
        <svg
          className="w-8 h-8 text-text-muted"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5"
          />
        </svg>
        <p className="text-sm text-text-secondary text-center">{label}</p>
        <p className="text-xs text-text-muted">Max {maxSizeMb} MB</p>
      </div>

      {error && (
        <p role="alert" className="text-xs text-nova-red">
          {error}
        </p>
      )}

      <input
        ref={inputRef}
        type="file"
        accept={accept}
        className="hidden"
        onChange={onChange}
        aria-hidden="true"
        tabIndex={-1}
      />
    </div>
  )
}
