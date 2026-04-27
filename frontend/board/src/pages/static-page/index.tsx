import { extractStatus, staleTime } from '@/app/providers/QueryProvider'
import { api, isApiError } from '@/shared/api/client'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { Skeleton } from '@/shared/ui/skeleton'
import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useNavigate, useParams } from 'react-router'

export function StaticPage() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()

  const { data, isLoading, error } = useQuery({
    queryKey: ['page', slug],
    queryFn: async () => {
      const { data, error } = await api.GET('/pages/{slug}', {
        params: { path: { slug: slug! } },
      })
      if (error) throw error
      return data
    },
    enabled: !!slug,
    staleTime: staleTime.static,
    retry: false,
  })

  useEffect(() => {
    if (!error) return
    const code = isApiError(error) ? error.code : undefined
    if (code === 'PAGE_NOT_FOUND' || extractStatus(error) === 404) {
      void navigate('/404', { replace: true })
    }
  }, [error, navigate])

  if (isLoading) {
    return (
      <div className="flex flex-col gap-4 max-w-3xl mx-auto px-4 py-6 sm:px-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-4 w-4/6" />
      </div>
    )
  }

  if (!data) return null

  return (
    <div className="max-w-3xl mx-auto px-4 py-6 sm:px-6">
      <h1
        className="text-3xl font-black text-text-primary mb-6"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        {data.title ?? ''}
      </h1>
      {data.content && <MarkdownRenderer content={data.content} allowHtml />}
    </div>
  )
}
