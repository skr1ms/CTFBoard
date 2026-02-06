export interface FilerChunk {
  file_id: string
  size: number
  mtime: number
  e_tag: string
  fid: { volume_id: number; file_key: number; cookie: number }
  is_compressed?: boolean
}

export interface FilerEntry {
  FullPath: string
  Mtime?: string
  Crtime?: string
  Mode?: number
  Uid?: number
  Gid?: number
  Mime?: string
  FileSize?: number
  Md5?: string | null
  chunks?: FilerChunk[]
  [key: string]: unknown
}

export interface FilerListResponse {
  Entries: FilerEntry[] | null
  Limit?: number
  LastFileName?: string
  ShouldDisplayLoadMore?: boolean
}

export interface ClusterInfo {
  version?: string
  freeVolumes?: number
  maxVolumes?: number
  datacenters: number
  racks: number
  nodes: number
  size: number
}

export interface MasterVolStatusDataCenter {
  [rack: string]: {
    [node: string]: Array<{ Size: number; [key: string]: unknown }>
  }
}

export interface MasterVolStatusResponse {
  Version?: string
  Volumes: {
    Free?: number
    Max?: number
    DataCenters: MasterVolStatusDataCenter
  }
}
