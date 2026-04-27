import { staleTime } from '@/app/providers/QueryProvider'
import { api } from '@/shared/api/client'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router'

function usePublicPages() {
  return useQuery({
    queryKey: ['pages'],
    queryFn: async () => {
      const { data, error } = await api.GET('/pages')
      if (error) throw error
      return (data ?? []) as Array<{
        id?: string
        title?: string
        slug?: string
        order_index?: number
      }>
    },
    staleTime: staleTime.static,
  })
}

export function Footer() {
  const { data: pages } = usePublicPages()
  const visiblePages = (pages ?? [])
    .filter((p) => p.slug && p.title)
    .sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0))

  if (visiblePages.length === 0) return null

  return (
    <footer className="border-t border-space-border bg-space-dark/60 py-4">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
        {visiblePages.map((p) => (
          <Link
            key={p.slug}
            to={`/pages/${p.slug}`}
            className="text-xs text-text-muted hover:text-text-secondary transition-colors"
          >
            {p.title}
          </Link>
        ))}
      </div>
    </footer>
  )
}
