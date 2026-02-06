import React from 'react'
import Grid from '@mui/material/Grid'
import { Box, IconButton, TablePagination, Typography, useTheme, Button } from '@mui/material'
import { DataGrid, useGridApiContext, useGridState } from '@mui/x-data-grid'
import DescriptionIcon from '@mui/icons-material/Description'
import { Folder as FolderIcon } from '@mui/icons-material'
import DeleteIcon from '@mui/icons-material/Delete'
import DownloadForOfflineIcon from '@mui/icons-material/DownloadForOffline'
import DriveFileMoveIcon from '@mui/icons-material/DriveFileMove'
import { LocationContext } from '../context/LocationContextWrapper'
import { SelectionContext } from '../context/SelectionContextWrapper'
import Filer, { connectionString } from '../seaweedfs/filer'
import { getName } from '../seaweedfs/file'
import DeleteMultipleDialog from '../dialogs/DeleteMultipleDialog'
import CopyMoveDialog from '../dialogs/CopyMoveDialog'
import type { FileEntry } from '../types/models'
import JSZip from 'jszip'

interface DetailsFooterProps {
  files: FileEntry[]
}

function DetailsFooter(props: DetailsFooterProps): React.ReactElement {
  const { files } = props
  const theme = useTheme()
  const locationContext = React.useContext(LocationContext)
  const { showSuccess } = React.useContext(SelectionContext)
  const apiRef = useGridApiContext()
  const [state] = useGridState(apiRef)
  const [rowsPerPage, setRowsPerPage] = React.useState(10)
  const [deleteOpen, setDeleteOpen] = React.useState(false)
  const [copyMoveOpen, setCopyMoveOpen] = React.useState(false)
  const [downloading, setDownloading] = React.useState(false)

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>): void => {
    setRowsPerPage(parseInt(event.target.value, 10))
    apiRef.current?.setPage(0)
  }

  async function del(): Promise<void> {
    for (const file of files) {
      await Filer.deleteItem(file.FullPath, !file.isFile)
    }
    locationContext.refresh()
  }

  async function handleCopyTo(destPath: string): Promise<void> {
    const count = files.length
    for (const file of files) {
      if (file.isFile) {
        await Filer.copyFile(file.FullPath, destPath)
      } else {
        await Filer.copyDirectory(file.FullPath, destPath)
      }
    }
    locationContext.refresh()
    showSuccess(`Successfully copied ${count} item${count !== 1 ? 's' : ''}.`)
  }

  async function handleMoveTo(destPath: string): Promise<void> {
    const count = files.length
    for (const file of files) {
      if (file.isFile) {
        await Filer.copyFile(file.FullPath, destPath)
        await Filer.deleteItem(file.FullPath)
      } else {
        await Filer.copyDirectory(file.FullPath, destPath)
        await Filer.deleteItem(file.FullPath, true)
      }
    }
    locationContext.refresh()
    showSuccess(`Successfully moved ${count} item${count !== 1 ? 's' : ''}.`)
  }

  async function handleDownload(): Promise<void> {
    const fileList = files.filter((f) => f.isFile)
    if (fileList.length === 0) return
    if (fileList.length === 1) {
      window.open(`${connectionString}${fileList[0].FullPath}`, '_blank')
      return
    }
    setDownloading(true)
    try {
      const zip = new JSZip()
      for (const file of fileList) {
        const blob = await Filer.getRawContent(file.FullPath)
        zip.file(getName(file.FullPath), blob)
      }
      const blob = await zip.generateAsync({ type: 'blob' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'download.zip'
      a.click()
      URL.revokeObjectURL(url)
    } finally {
      setDownloading(false)
    }
  }

  const selectedItems = files.map((f) => ({ path: f.FullPath, isFile: f.isFile }))
  const hasFiles = files.filter((f) => f.isFile).length > 0

  return (
        <React.Fragment>
            <Box
                sx={{
                    display: "flex",
                    padding: theme.spacing(1),
                    alignItems: "center",
                    gap: 1
                }}
            >
                <IconButton
                    onClick={() => setDeleteOpen(true)}
                    disabled={files.length === 0}
                    aria-label="delete"
                >
                    <DeleteIcon />
                </IconButton>
                <Button
                  size="small"
                  startIcon={<DownloadForOfflineIcon />}
                  onClick={() => void handleDownload()}
                  disabled={files.length === 0 || !hasFiles || downloading}
                >
                  Download
                </Button>
                <Button size="small" startIcon={<DriveFileMoveIcon />} onClick={() => setCopyMoveOpen(true)} disabled={files.length === 0}>
                  Copy / Move to…
                </Button>
                <Box sx={{ flex: 1 }} />
                <TablePagination
                    component="div"
                    count={state.pagination.rowCount}
                    page={state.pagination.page}
                    onPageChange={(event, value) => apiRef.current?.setPage(value - 1)}
                    rowsPerPage={rowsPerPage}
                    onRowsPerPageChange={handleChangeRowsPerPage}
                />
            </Box>
            <DeleteMultipleDialog
                files={files.map(file => file.name)}
                del={del}
                close={() => setDeleteOpen(false)}
                open={deleteOpen}
            />
            <CopyMoveDialog
              open={copyMoveOpen}
              close={() => setCopyMoveOpen(false)}
              currentLocation={locationContext.currentLocation}
              items={selectedItems}
              onCopy={handleCopyTo}
              onMove={handleMoveTo}
            />
        </React.Fragment>
    )
}

interface DetailsViewProps {
  files: FileEntry[]
}

function DetailsView(props: DetailsViewProps): React.ReactElement {
    const theme = useTheme()
    const [selected, setSelected] = React.useState<FileEntry[]>([])

    const files = props.files.map((file, index) => {
        return { ...file, id: index }
    })

    function handleSelectionChange(selection: (string | number)[]): void {
        setSelected(selection.map((id) => files[Number(id)]).filter((f): f is FileEntry & { id: number } => f != null))
    }

    const locationContext = React.useContext(LocationContext)

    const columns = [
        {
            field: "name",
            headerName: "Name",
            flex: 1,
            cellClassName: "name-column",
            renderCell: (params: { row: FileEntry & { id: number } }) => {
                return (
                    <React.Fragment>
                        {params.row.isFile
                            ? <DescriptionIcon />
                            : <FolderIcon />
                        }
                        <Typography sx={{ marginLeft: theme.spacing(1), userSelect: "none" }}>
                            {params.row.name}
                        </Typography>
                    </React.Fragment>
                )
            }
        },
        {
            field: "Mtime",
            headerName: "Last Modified",
            flex: 1
        },
        {
            field: "Crtime",
            headerName: "Create Time",
            flex: 1
        },
        {
            field: "Mode",
            headerName: "Permissions",
            flex: 1
        },
        {
            field: "Uid",
            headerName: "UID",
            flex: 1
        },
        {
            field: "Gid",
            headerName: "GID",
            flex: 1
        },
        {
            field: "Mime",
            headerName: "MIME Type",
            flex: 1
        },
        {
            field: "Replication",
            name: "Replication",
            flex: 1
        },
        {
            field: "Collection",
            headerName: "Collection",
            flex: 1
        },
        {
            field: "DiskType",
            headerName: "Disk Type",
            flex: 1
        },
        {
            field: "GroupNames",
            headerName: "Group Names",
            flex: 1
        },
        {
            field: "SymlinkTarget",
            headerName: "Symlink Target",
            flex: 1
        },
        {
            field: "Md5",
            headerName: "MD5",
            flex: 1
        },
        {
            field: "FileSize",
            headerName: "Size",
            flex: 1
        },
        {
            field: "Extended",
            headerName: "Extended",
            flex: 1
        },
        {
            field: "HardLinkId",
            headerName: "Hard Link ID",
            flex: 1
        },
        {
            field: "HardLinkCounter",
            headerName: "Hard Link Counter",
            flex: 1,
        },
        {
            field: "Remote",
            headerName: "Remote",
            flex: 1
        }
    ]

  const onNameDoubleClick = (
    params: { row: FileEntry & { id: number } },
    _event: React.MouseEvent
  ): void => {
    if (params.row.isFile) {
      window.open(`${connectionString}${params.row.FullPath}`, '_blank')
    } else {
      locationContext.updateLocation(params.row.FullPath)
    }
  }
  const onCellDoubleClick = (
    params: { field: string; row: FileEntry & { id: number } },
    event: React.MouseEvent
  ): void => {
    if (params.field === 'name') {
      event.preventDefault()
      onNameDoubleClick(params, event)
    }
  }

    return (
        <Grid
            container
            spacing={2}
            sx={{
                marginLeft: "240px",
                width: "calc(100vw - 240px)",
                paddingTop: "15px",
                height: "calc(100vh - 64px)"
            }}
        >
            <DataGrid
                disableSelectionOnClick
                autoPageSize
                checkboxSelection
                columns={columns}
                rows={files}
                onCellDoubleClick={onCellDoubleClick}
                onSelectionModelChange={handleSelectionChange}
                components={{
                    Footer: DetailsFooter,
                }}
                componentsProps={{
                    footer: { files: selected }
                }}
            />
        </Grid>
    )

}

export default DetailsView