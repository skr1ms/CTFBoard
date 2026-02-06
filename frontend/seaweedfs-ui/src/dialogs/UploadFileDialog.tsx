import React from 'react'
import { useTheme } from '@mui/material'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button'
import Box from '@mui/material/Box'
import { DropzoneAreaBase } from 'react-mui-dropzone'

import { LocationContext } from '../context/LocationContextWrapper'
import Filer from '../seaweedfs/filer'
import { getFullPath } from '../seaweedfs/file'
import type { UploadFileDialogProps } from '../types/dialogs'

function getRelativePath(file: File): string {
  const f = file as File & { path?: string }
  return f.path ?? file.webkitRelativePath ?? ''
}

function getParentDirs(relativePath: string): string[] {
  const parts = relativePath.split('/').filter(Boolean)
  parts.pop()
  const dirs: string[] = []
  for (let i = 0; i < parts.length; i++) {
    dirs.push(parts.slice(0, i + 1).join('/') + '/')
  }
  return dirs
}

function UploadFileDialog(props: UploadFileDialogProps): React.ReactElement {
    const { open, close, files: propsFiles } = props
    const context = React.useContext(LocationContext)
    const theme = useTheme()

    const [filesFromZone, setFilesFromZone] = React.useState<File[]>(propsFiles ?? [])
    const [filesFromFolder, setFilesFromFolder] = React.useState<File[]>([])
    const [folder, setFolder] = React.useState('')

    React.useEffect(() => {
        if (propsFiles !== undefined) setFilesFromZone([...propsFiles])
    }, [propsFiles])

    function handleClose(): void {
        setFolder('')
        setFilesFromZone([])
        setFilesFromFolder([])
        close()
    }

    function onFolderSelect(e: React.ChangeEvent<HTMLInputElement>): void {
        const input = e.target
        if (!input.files) return
        const list = Array.from(input.files)
        setFilesFromFolder((prev) => [...prev, ...list])
        input.value = ''
    }

    function onDropzoneAdd(newFileObjects: { file: File; data: string | ArrayBuffer | null }[]): void {
        const newFiles = newFileObjects.map((o) => o.file)
        const fromFolder: File[] = []
        const fromZone: File[] = []
        for (const f of newFiles) {
            const p = getRelativePath(f)
            if (p.includes('/')) fromFolder.push(f)
            else fromZone.push(f)
        }
        setFilesFromFolder((prev) => [...prev, ...fromFolder])
        setFilesFromZone((prev) => {
            const next = [...prev, ...fromZone]
            return next.slice(0, 500)
        })
    }

    function onDropzoneDelete(_deleted: { file: File; data: string | ArrayBuffer | null }, index: number): void {
        setFilesFromZone((prev) => {
            const next = [...prev]
            next.splice(index, 1)
            return next
        })
    }

    async function submit(): Promise<void> {
        let basePath = getFullPath(folder, context.currentLocation)
        if (!basePath.endsWith('/')) basePath += '/'

        const allDirs = new Set<string>()
        for (const file of filesFromFolder) {
            const rel = getRelativePath(file)
            if (rel) getParentDirs(rel).forEach((d) => allDirs.add(d))
        }
        const sortedDirs = Array.from(allDirs).sort((a, b) => a.split('/').length - b.split('/').length)
        for (const d of sortedDirs) {
            await Filer.createFolder(basePath + d)
        }

        for (const file of filesFromFolder) {
            const rel = getRelativePath(file)
            const targetPath = rel ? basePath + rel : basePath + file.name
            await Filer.uploadFile(targetPath, file)
        }
        for (const file of filesFromZone) {
            await Filer.uploadFile(basePath + file.name, file)
        }
        context.refresh()
        handleClose()
    }

    const totalCount = filesFromZone.length + filesFromFolder.length
    function isValid(): boolean {
        return totalCount > 0
    }

    return (
        <Dialog
            open={open}
            onClose={handleClose}
            fullWidth
            maxWidth="md"
        >
            <DialogTitle sx={{ textAlign: 'center' }}>
                Upload Files
            </DialogTitle>
            <DialogContent>
                <TextField
                    fullWidth
                    sx={{
                        marginTop: theme.spacing(1),
                        marginBottom: theme.spacing(1)
                    }}
                    label="Subfolder (optional)"
                    inputProps={{ "aria-label": "folder name" }}
                    value={folder}
                    onChange={(event) => setFolder(event.target.value)}
                />
                <DropzoneAreaBase
                    fileObjects={filesFromZone.map((file) => ({ file, data: null }))}
                    onAdd={onDropzoneAdd}
                    onDelete={onDropzoneDelete}
                    useChipsForPreview
                    filesLimit={500}
                    dropzoneText="Drag and drop files or folders here, or click to select files"
                    inputProps={{ "aria-label": "upload input" }}
                    dropzoneProps={{}}
                />
                <Box sx={{ display: 'flex', gap: 1, mt: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                    <Button variant="outlined" size="small" component="label" aria-label="add folder">
                        Add folder
                        <input
                            type="file"
                            hidden
                            multiple
                            {...({ webkitdirectory: '' } as React.InputHTMLAttributes<HTMLInputElement>)}
                            onChange={onFolderSelect}
                        />
                    </Button>
                    {filesFromFolder.length > 0 && (
                        <Box component="span" sx={{ fontSize: '0.875rem', color: 'text.secondary' }}>
                            + {filesFromFolder.length} files from folder(s)
                        </Box>
                    )}
                </Box>
            </DialogContent>
            <DialogActions>
                <Button variant="contained" aria-label="close" onClick={handleClose}>
                    Close
                </Button>
                <Button variant="contained" aria-label="submit" onClick={() => void submit()} disabled={!isValid()}>
                    Submit
                </Button>
            </DialogActions>
        </Dialog>
    )
}

export default UploadFileDialog