// This entire file is lazy-imported - ECharts stays out of the initial bundle.
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import * as echarts from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { useEffect, useRef } from 'react'

echarts.use([LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

// ---------------------------------------------------------------------------
// Shared chart theme helpers
// ---------------------------------------------------------------------------
const COLORS = {
  blue: '#6C8EF5',
  green: '#34D399',
  yellow: '#F59E0B',
  red: '#F472B6',
  violet: '#A78BFA',
  sky: '#38BDF8',
}

const baseAxis = {
  axisLine: { lineStyle: { color: '#1e2a3a' } },
  axisTick: { lineStyle: { color: '#1e2a3a' } },
  axisLabel: { color: '#64748b', fontSize: 11 },
  splitLine: { lineStyle: { color: '#1e2a3a', type: 'dashed' as const } },
}

const baseTooltip = {
  backgroundColor: '#0d1520',
  borderColor: '#1e2a3a',
  borderWidth: 1,
  textStyle: { color: '#e2e8f0', fontSize: 12 },
}

function useChart(option: echarts.EChartsCoreOption | null, height = 'h-64') {
  const ref = useRef<HTMLDivElement>(null)
  const chartRef = useRef<echarts.ECharts | null>(null)

  useEffect(() => {
    if (!ref.current) return
    const chart = echarts.init(ref.current, null, { renderer: 'canvas' })
    chartRef.current = chart
    const ro = new ResizeObserver(() => chart.resize())
    ro.observe(ref.current)
    return () => {
      ro.disconnect()
      chart.dispose()
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    if (option)
      chartRef.current?.setOption(option as Parameters<echarts.ECharts['setOption']>[0], true)
  }, [option])

  return { ref, height }
}

// ---------------------------------------------------------------------------
// 1. Score Progression (line chart)
// ---------------------------------------------------------------------------
interface ScorePoint {
  points?: number
  team_name?: string
  timestamp?: string
}

interface ScoreProgressionProps {
  data: ScorePoint[]
}

export function ScoreProgressionChart({ data }: ScoreProgressionProps) {
  // Group by team
  const teamMap = new Map<string, [string, number][]>()
  for (const p of data) {
    const name = p.team_name ?? 'Unknown'
    if (!teamMap.has(name)) teamMap.set(name, [])
    teamMap.get(name)!.push([p.timestamp ?? '', p.points ?? 0])
  }

  const colorList = Object.values(COLORS)
  const series = Array.from(teamMap.entries()).map(([name, pts], i) => ({
    name,
    type: 'line' as const,
    smooth: true,
    symbol: 'none',
    lineStyle: { width: 2 },
    itemStyle: { color: colorList[i % colorList.length] },
    data: pts,
  }))

  const option: echarts.EChartsCoreOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#94a3b8' },
    grid: { left: 50, right: 16, top: 8, bottom: 48 },
    xAxis: { type: 'time', ...baseAxis, splitLine: { show: false } },
    yAxis: { type: 'value', ...baseAxis, minInterval: 1 },
    tooltip: { ...baseTooltip, trigger: 'axis' },
    legend: {
      bottom: 0,
      type: 'scroll',
      textStyle: { color: '#94a3b8', fontSize: 11 },
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
    },
    series,
  }

  const { ref } = useChart(series.length ? option : null)
  return <div ref={ref} className="w-full h-64" />
}

// ---------------------------------------------------------------------------
// 2. Solve Percentages (horizontal bar)
// ---------------------------------------------------------------------------
interface SolvePct {
  title?: string
  percentage?: number
  solve_count?: number
}

interface SolvePercentagesProps {
  data: SolvePct[]
}

export function SolvePercentagesChart({ data }: SolvePercentagesProps) {
  const sorted = [...data].sort((a, b) => (b.percentage ?? 0) - (a.percentage ?? 0)).slice(0, 20)

  const option: echarts.EChartsCoreOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#94a3b8' },
    grid: { left: 120, right: 60, top: 8, bottom: 32, containLabel: false },
    xAxis: {
      type: 'value',
      max: 100,
      ...baseAxis,
      axisLabel: { ...baseAxis.axisLabel, formatter: '{value}%' },
    },
    yAxis: {
      type: 'category',
      data: sorted.map((d) => d.title ?? ''),
      ...baseAxis,
      splitLine: { show: false },
      axisLabel: { ...baseAxis.axisLabel, width: 110, overflow: 'truncate' as const },
    },
    tooltip: {
      ...baseTooltip,
      trigger: 'axis',
      formatter: (params: unknown) => {
        const p = (params as { name: string; value: number }[])[0]
        if (!p) return ''
        return `${p.name}<br/><b>${p.value.toFixed(1)}%</b>`
      },
    },
    series: [
      {
        type: 'bar',
        data: sorted.map((d) => d.percentage ?? 0),
        itemStyle: { color: COLORS.blue, borderRadius: [0, 3, 3, 0] },
        label: {
          show: true,
          position: 'right' as const,
          formatter: '{c}%',
          color: '#94a3b8',
          fontSize: 11,
        },
      },
    ],
  }

  const { ref } = useChart(sorted.length ? option : null)
  return <div ref={ref} className="w-full h-64" />
}

// ---------------------------------------------------------------------------
// 3. Score Distribution (bar histogram)
// ---------------------------------------------------------------------------
interface ScoreBucket {
  bucket?: string
  count?: number
}

interface ScoreDistributionProps {
  data: ScoreBucket[]
}

export function ScoreDistributionChart({ data }: ScoreDistributionProps) {
  const option: echarts.EChartsCoreOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#94a3b8' },
    grid: { left: 50, right: 16, top: 8, bottom: 48 },
    xAxis: {
      type: 'category',
      data: data.map((d) => d.bucket ?? ''),
      ...baseAxis,
      splitLine: { show: false },
      axisLabel: { ...baseAxis.axisLabel, rotate: 30 },
    },
    yAxis: { type: 'value', ...baseAxis, minInterval: 1 },
    tooltip: { ...baseTooltip, trigger: 'axis' },
    series: [
      {
        type: 'bar',
        data: data.map((d) => d.count ?? 0),
        itemStyle: { color: COLORS.violet, borderRadius: [3, 3, 0, 0] },
      },
    ],
  }

  const { ref } = useChart(data.length ? option : null)
  return <div ref={ref} className="w-full h-64" />
}

// ---------------------------------------------------------------------------
// 4. Submission Timeline (stacked area)
// ---------------------------------------------------------------------------
interface SubPoint {
  date?: string
  correct?: number
  incorrect?: number
}

interface SubmissionTimelineProps {
  data: SubPoint[]
}

export function SubmissionTimelineChart({ data }: SubmissionTimelineProps) {
  const option: echarts.EChartsCoreOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#94a3b8' },
    grid: { left: 50, right: 16, top: 8, bottom: 48 },
    xAxis: {
      type: 'category',
      data: data.map((d) => d.date ?? ''),
      ...baseAxis,
      splitLine: { show: false },
      axisLabel: { ...baseAxis.axisLabel, rotate: 30 },
    },
    yAxis: { type: 'value', ...baseAxis, minInterval: 1 },
    tooltip: { ...baseTooltip, trigger: 'axis' },
    legend: {
      bottom: 0,
      textStyle: { color: '#94a3b8', fontSize: 11 },
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
    },
    series: [
      {
        name: 'Correct',
        type: 'line',
        stack: 'total',
        smooth: true,
        symbol: 'none',
        areaStyle: { color: COLORS.green, opacity: 0.3 },
        lineStyle: { color: COLORS.green },
        itemStyle: { color: COLORS.green },
        data: data.map((d) => d.correct ?? 0),
      },
      {
        name: 'Incorrect',
        type: 'line',
        stack: 'total',
        smooth: true,
        symbol: 'none',
        areaStyle: { color: COLORS.red, opacity: 0.3 },
        lineStyle: { color: COLORS.red },
        itemStyle: { color: COLORS.red },
        data: data.map((d) => d.incorrect ?? 0),
      },
    ],
  }

  const { ref } = useChart(data.length ? option : null)
  return <div ref={ref} className="w-full h-64" />
}

// ---------------------------------------------------------------------------
// 5. Registration Growth (line)
// ---------------------------------------------------------------------------
interface RegPoint {
  date?: string
  count?: number
}

interface RegistrationGrowthProps {
  data: RegPoint[]
}

export function RegistrationGrowthChart({ data }: RegistrationGrowthProps) {
  // Cumulative sum
  let running = 0
  const cumulative = data.map((d) => {
    running += d.count ?? 0
    return running
  })

  const option: echarts.EChartsCoreOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#94a3b8' },
    grid: { left: 50, right: 16, top: 8, bottom: 48 },
    xAxis: {
      type: 'category',
      data: data.map((d) => d.date ?? ''),
      ...baseAxis,
      splitLine: { show: false },
      axisLabel: { ...baseAxis.axisLabel, rotate: 30 },
    },
    yAxis: { type: 'value', ...baseAxis, minInterval: 1 },
    tooltip: { ...baseTooltip, trigger: 'axis' },
    series: [
      {
        name: 'Total Users',
        type: 'line',
        smooth: true,
        symbol: 'none',
        areaStyle: { color: COLORS.sky, opacity: 0.2 },
        lineStyle: { color: COLORS.sky, width: 2 },
        itemStyle: { color: COLORS.sky },
        data: cumulative,
      },
    ],
  }

  const { ref } = useChart(data.length ? option : null)
  return <div ref={ref} className="w-full h-64" />
}
