import { useEffect } from 'react'

interface ShortcutOptions {
  /** Only fire when no input/textarea/select is focused */
  ignoreInputs?: boolean
  disabled?: boolean
}

/**
 * Registers a global keydown handler for a single key combination.
 * @param key - The KeyboardEvent.key value (e.g. 'Escape', 'Enter', '?')
 * @param callback - Handler to invoke
 * @param options - ctrl/alt/shift modifiers + ignoreInputs
 */
export function useKeyboardShortcut(
  key: string,
  callback: (e: KeyboardEvent) => void,
  options: ShortcutOptions & { ctrl?: boolean; alt?: boolean; shift?: boolean } = {},
) {
  const {
    ctrl = false,
    alt = false,
    shift = false,
    ignoreInputs = true,
    disabled = false,
  } = options

  useEffect(() => {
    if (disabled) return

    const handler = (e: KeyboardEvent) => {
      if (ignoreInputs) {
        const tag = (e.target as HTMLElement)?.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
        if ((e.target as HTMLElement)?.isContentEditable) return
      }

      if (e.key !== key) return
      if (ctrl && !e.ctrlKey && !e.metaKey) return
      if (alt && !e.altKey) return
      if (shift && !e.shiftKey) return

      callback(e)
    }

    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [key, callback, ctrl, alt, shift, ignoreInputs, disabled])
}
