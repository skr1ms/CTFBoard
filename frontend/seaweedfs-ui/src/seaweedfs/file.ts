import { connectionString } from './filer'
import type { FilerEntry } from '../types/api'
import type { FileEntry } from '../types/models'

export function getName(path: string): string {
  const finalSlash = path.lastIndexOf('/')
  return path.slice(finalSlash + 1)
}

export function getFullPath(fileName: string, currentLocation: string): string {
  if (fileName.startsWith('/')) {
    return fileName
  }
  if (fileName.startsWith('../')) {
    const parentName = currentLocation.slice(0, currentLocation.lastIndexOf('/'))
    const normalized = fileName.replace('../', '/')
    return `${parentName}${normalized}`
  }
  if (fileName.startsWith('./')) {
    const normalized = fileName.replace('./', '/')
    return `${currentLocation}${normalized}`
  }
  const location = currentLocation.endsWith('/') ? currentLocation : `${currentLocation}/`
  return `${location}${fileName}`
}

export class File implements FileEntry {
  name: string
  FullPath: string
  isFile = true

  constructor(data: FilerEntry) {
    if (!data.FullPath) {
      throw new ReferenceError("File objects require 'FullPath' keyword")
    }
    this.name = getName(data.FullPath)
    this.FullPath = data.FullPath
    Object.assign(this, data)
  }

  async getContent(): Promise<string> {
    const response = await fetch(`${connectionString}${this.FullPath}`)
    return response.text()
  }
}

export default File
