import Folder from './folder'
import { getName } from './file'
import { getFullPath } from './file'
import type { FileEntry } from '../types/models'

const filerProxyPath = import.meta.env.VITE_FILER_PROXY_PATH as string | undefined
const host = import.meta.env.VITE_HOST ?? ''
const port = import.meta.env.VITE_FILER_PORT ?? ''
const connectionString = filerProxyPath
  ? (filerProxyPath.startsWith('/') ? filerProxyPath : `/${filerProxyPath}`).replace(/\/$/, '')
  : `http://${host}:${port}`

export async function getFiles(path: string): Promise<FileEntry[]> {
  const folder = new Folder(path)
  return folder.getContent()
}

function checkOk(res: Response, message: string): void {
  if (!res.ok) {
    throw new Error(`${message}: ${res.status} ${res.statusText}`)
  }
}

export async function getContent(file: string): Promise<string> {
  const response = await fetch(`${connectionString}${file}`)
  checkOk(response, 'Failed to get file content')
  return response.text()
}

export async function getRawContent(file: string): Promise<Blob> {
  const response = await fetch(`${connectionString}${file}`)
  checkOk(response, 'Failed to get file')
  return response.blob()
}

export async function uploadFile(path: string, file: File): Promise<Response> {
  const formData = new FormData()
  formData.append(path, file)
  const res = await fetch(`${connectionString}${path}`, {
    method: 'POST',
    body: formData,
  })
  checkOk(res, 'Upload failed')
  return res
}

export async function createFolder(path: string): Promise<Response> {
  const normalizedPath = path.endsWith('/') ? path : `${path}/`
  const res = await fetch(`${connectionString}${normalizedPath}`, {
    method: 'POST',
  })
  checkOk(res, 'Create folder failed')
  return res
}

export async function deleteItem(path: string, recursive = false): Promise<Response> {
  const url = `${connectionString}${path}?recursive=${recursive}`
  const res = await fetch(url, {
    method: 'DELETE',
  })
  checkOk(res, 'Delete failed')
  return res
}

export async function copyFile(srcPath: string, destDir: string): Promise<Response> {
  const blob = await getRawContent(srcPath)
  const name = getName(srcPath)
  const destPath = getFullPath(name, destDir.endsWith('/') ? destDir : destDir + '/')
  const file = new File([blob], name)
  return uploadFile(destPath, file)
}

export async function copyDirectory(srcPath: string, destDir: string): Promise<void> {
  const normalizedSrc = srcPath.endsWith('/') ? srcPath.slice(0, -1) : srcPath
  const name = getName(normalizedSrc)
  const normalizedDest = destDir.endsWith('/') ? destDir : destDir + '/'
  const destPath = normalizedDest + name + '/'
  const normalizedDestPath = destPath.endsWith('/') ? destPath : destPath + '/'
  const createRes = await fetch(`${connectionString}${normalizedDestPath}`, { method: 'POST' })
  if (!createRes.ok && createRes.status !== 409) {
    const body = await createRes.text()
    throw new Error(`Create folder failed (copy): ${createRes.status} ${createRes.statusText}${body ? ` - ${body.slice(0, 100)}` : ''}`)
  }
  const entries = await getFiles(normalizedSrc + '/')
  for (const entry of entries) {
    if (entry.name === '..' || entry.name === '.') continue
    if (entry.isFile) {
      await copyFile(entry.FullPath, destPath)
    } else {
      await copyDirectory(entry.FullPath, destPath)
    }
  }
}

const Filer = {
  getFiles,
  getContent,
  uploadFile,
  createFolder,
  deleteItem,
  getRawContent,
  copyFile,
  copyDirectory,
}

export default Filer
export { connectionString }
