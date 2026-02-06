declare module 'use-double-click' {
  interface UseDoubleClickOptions {
    onSingleClick?: (e?: MouseEvent) => void
    onDoubleClick?: (e?: MouseEvent) => void
    ref: React.RefObject<HTMLElement | null>
    latency?: number
    delay?: number
  }
  function useDoubleClick(options: UseDoubleClickOptions): void
  export default useDoubleClick
}
