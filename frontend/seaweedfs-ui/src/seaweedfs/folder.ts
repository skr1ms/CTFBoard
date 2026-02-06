import { connectionString } from './filer'
import File from './file'
import { getName } from './file'
import type { FilerEntry, FilerListResponse } from '../types/api'
import type { FileEntry } from '../types/models'

export default class Folder implements FileEntry {
  path: string
  name: string
  FullPath: string
  entries: FileEntry[] = []
  loaded = false
  isFile = false

  constructor(path: string, data: FilerEntry | null = null) {
    this.path = path
    this.name = path
    this.FullPath = path
    if (data) Object.assign(this, data)
  }

  async getContent(): Promise<FileEntry[]> {
    const response = await fetch(`${connectionString}${this.path}`, {
      headers: {
        Accept: 'application/json',
      },
    })
    const text = await response.text()
    let data: FilerListResponse
    try {
      data = JSON.parse(text) as FilerListResponse
    } catch {
      data = { Entries: [] }
    }
    if (this.name !== '/') {
      let parentName = this.name.slice(0, this.name.lastIndexOf('/'))
      if (parentName === '') {
        parentName = '/'
      }
      const parent = new Folder(parentName, { FullPath: parentName })
      this.entries.push(parent)
    }
    const entries = data.Entries ?? []
    for (const obj of entries) {
      if (obj.chunks) {
        this.entries.push(new File(obj))
      } else {
        const name = getName(obj.FullPath)
        this.entries.push(new Folder(name, obj))
      }
    }
    this.loaded = true
    return this.entries
  }
}
