import type { FilerEntry } from './api'

export interface FileEntry {
  name: string
  FullPath: string
  isFile: boolean
  path?: string
  entries?: FileEntry[]
  loaded?: boolean
  Mtime?: string
}

export type FileEntryLike = FileEntry | (FilerEntry & { name: string; isFile: boolean; entries?: FileEntry[] })

export interface FolderData {
  path: string
  name: string
  entries: FileEntry[]
  loaded: boolean
  isFile: false
}
