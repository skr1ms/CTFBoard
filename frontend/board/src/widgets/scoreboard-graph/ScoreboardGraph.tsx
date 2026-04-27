import type { TeamTimeline } from '@/features/scoreboard/useScoreboard'
import { useEffect, useRef } from 'react'

// ECharts is imported inside this file which is only loaded lazily.
// This keeps ECharts (~1 MB) out of the initial bundle.
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import * as echarts from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

// Palette that fits the cosmic theme
const SERIES_COLORS = [
  '#6C8EF5', // cosmic-blue
  '#34D399', // stellar-green
  '#F59E0B', // solar-yellow
  '#F472B6', // pink
  '#A78BFA', // violet
  '#38BDF8', // sky
  '#FB7185', // rose
  '#4ADE80', // lime
  '#FBBF24', // amber
  '#818CF8', // indigo
]

interface Props {
  teams: TeamTimeline[]
}

export function ScoreboardGraph({ teams }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<echarts.ECharts | null>(null)

  useEffect(() => {
    if (!containerRef.current) return
    const chart = echarts.init(containerRef.current, null, { renderer: 'canvas' })
    chartRef.current = chart

    const resizeObserver = new ResizeObserver(() => chart.resize())
    resizeObserver.observe(containerRef.current)

    return () => {
      resizeObserver.disconnect()
      chart.dispose()
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    const chart = chartRef.current
    if (!chart) return

    const series = teams.map((team, idx) => ({
      name: team.team_name ?? `Team ${idx + 1}`,
      type: 'line' as const,
      smooth: true,
      symbol: 'circle',
      symbolSize: 5,
      lineStyle: { width: 2 },
      itemStyle: { color: SERIES_COLORS[idx % SERIES_COLORS.length] },
      data: (team.timeline ?? []).map((pt) => [pt.timestamp ?? '', pt.score ?? 0]),
    }))

    chart.setOption({
      backgroundColor: 'transparent',
      textStyle: { color: '#94a3b8', fontFamily: 'Inter, sans-serif', fontSize: 12 },
      grid: { left: 56, right: 24, top: 16, bottom: 56, containLabel: false },
      xAxis: {
        type: 'time',
        axisLine: { lineStyle: { color: '#1e2a3a' } },
        axisTick: { lineStyle: { color: '#1e2a3a' } },
        axisLabel: {
          color: '#64748b',
          formatter: (value: number) => {
            const d = new Date(value)
            return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
          },
        },
        splitLine: { show: false },
      },
      yAxis: {
        type: 'value',
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: '#64748b' },
        splitLine: { lineStyle: { color: '#1e2a3a', type: 'dashed' } },
        minInterval: 1,
      },
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#0d1520',
        borderColor: '#1e2a3a',
        borderWidth: 1,
        textStyle: { color: '#e2e8f0' },
        formatter: (params: unknown) => {
          if (!Array.isArray(params) || params.length === 0) return ''
          const first = params[0] as { value: [string, number]; color: string; seriesName: string }
          const ts = new Date(first.value[0]).toLocaleString(undefined, {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
          })
          const lines = (params as (typeof first)[]).map(
            (p) =>
              `<span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${p.color};margin-right:6px"></span>${p.seriesName}: <b>${p.value[1]}</b>`,
          )
          return `<div style="font-size:11px;color:#64748b;margin-bottom:4px">${ts}</div>${lines.join('<br/>')}`
        },
      },
      legend: {
        bottom: 0,
        type: 'scroll',
        textStyle: { color: '#94a3b8', fontSize: 11 },
        pageTextStyle: { color: '#64748b' },
        pageIconColor: '#6C8EF5',
        pageIconInactiveColor: '#1e2a3a',
        icon: 'circle',
        itemWidth: 8,
        itemHeight: 8,
      },
      series,
    })
  }, [teams])

  return <div ref={containerRef} className="w-full h-72 sm:h-96" />
}
