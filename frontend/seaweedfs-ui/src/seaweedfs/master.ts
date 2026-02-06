import type { ClusterInfo, MasterVolStatusResponse } from '../types/api'

const proxyPath = import.meta.env.VITE_MASTER_PROXY_PATH as string | undefined
const host = import.meta.env.VITE_HOST ?? ''
const port = import.meta.env.VITE_MASTER_PORT ?? ''
const masterConnectionString = proxyPath
  ? (proxyPath.startsWith('/') ? proxyPath : `/${proxyPath}`).replace(/\/$/, '')
  : `http://${host}:${port}`

export async function getClusterInfo(): Promise<ClusterInfo> {
  const output: ClusterInfo = {
    datacenters: 0,
    racks: 0,
    nodes: 0,
    size: 0,
  }
  try {
    const response = await fetch(`${masterConnectionString}/vol/status?pretty=y`)
    const text = await response.text()
    let data: MasterVolStatusResponse
    try {
      data = JSON.parse(text) as MasterVolStatusResponse
    } catch {
      return output
    }
    output.version = data.Version
    output.freeVolumes = data.Volumes?.Free
    output.maxVolumes = data.Volumes?.Max
    const dataCenters = data.Volumes?.DataCenters ?? {}
    output.datacenters = Object.keys(dataCenters).length
    const dc = dataCenters as unknown as Record<string, Record<string, Record<string, Array<{ Size?: number }>>>>
    for (const datacenter of Object.keys(dc)) {
      for (const rack of Object.keys(dc[datacenter])) {
        output.racks += 1
        for (const node of Object.keys(dc[datacenter][rack])) {
          output.nodes += 1
          for (const volume of dc[datacenter][rack][node]) {
            output.size += volume.Size ?? 0
          }
        }
      }
    }
  } catch (err) {
    console.error(err)
  }
  return output
}

const Master = {
  getClusterInfo,
}

export default Master
export { masterConnectionString }
