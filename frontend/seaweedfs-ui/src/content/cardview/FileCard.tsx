import React from 'react'
import { useTheme } from '@mui/material/styles'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import ListItemText from '@mui/material/ListItemText'
import ListItemIcon from '@mui/material/ListItemIcon'
import Divider from '@mui/material/Divider'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import DescriptionIcon from '@mui/icons-material/Description'
import DownloadForOfflineIcon from '@mui/icons-material/DownloadForOffline'
import InfoIcon from '@mui/icons-material/Info'
import DeleteIcon from '@mui/icons-material/Delete'
import BookmarkIcon from '@mui/icons-material/Bookmark'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import CheckBoxOutlineBlankIcon from '@mui/icons-material/CheckBoxOutlineBlank'
import CheckBoxIcon from '@mui/icons-material/CheckBox'
import useDoubleClick from 'use-double-click'
import Filer from '../../seaweedfs/filer'
import { LocationContext } from '../../context/LocationContextWrapper'
import DeleteItemConfirmation from '../../dialogs/DeleteItemConfirmation'
import { SelectionContext } from '../../context/SelectionContextWrapper'
import { MarqueeDragContext } from './CardView'
import { ProfileContext } from '../../context/ProfileContextWrapper'
import type { FileEntry } from '../../types/models'

interface FileCardRightClickMenuProps {
  open: boolean
  close: () => void
  anchorElement: HTMLElement | null
  download: () => void
  del: () => Promise<void>
  name: string
  favorite: () => void
  copy: () => void
  onDeleteClick: () => void
  deleteItemOpen: boolean
  closeDelete: () => void
}

function RightClickMenu(props: FileCardRightClickMenuProps): React.ReactElement {
    const { open, close, download, anchorElement, del, name, favorite, copy, onDeleteClick, deleteItemOpen, closeDelete } = props

    return (
        <React.Fragment>
            <Menu
                aria-label="file context menu"
                role="menu"
                anchorEl={anchorElement}
                open={open}
                onClose={close}
            >
                <MenuItem onClick={download} aria-label="open file">
                    <ListItemIcon><DownloadForOfflineIcon /></ListItemIcon>
                    <ListItemText>Download</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem onClick={favorite} aria-label="favorite file">
                    <ListItemIcon><BookmarkIcon /></ListItemIcon>
                    <ListItemText>Favorite</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem onClick={() => { copy(); close() }} aria-label="copy">
                    <ListItemIcon><ContentCopyIcon /></ListItemIcon>
                    <ListItemText>Copy</ListItemText>
                </MenuItem>
                <MenuItem onClick={() => { onDeleteClick(); close() }} aria-label="delete">
                    <ListItemIcon><DeleteIcon /></ListItemIcon>
                    <ListItemText>Delete</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem>
                    <ListItemIcon><InfoIcon /></ListItemIcon>
                    <ListItemText>Properties</ListItemText>
                </MenuItem>
            </Menu>
            <DeleteItemConfirmation name={name} del={del} close={closeDelete} open={deleteItemOpen} isFile={true} />
        </React.Fragment>
    )
}


interface FileCardProps {
  data: FileEntry
}

function FileCard(props: FileCardProps): React.ReactElement {
    const theme = useTheme()
    const context = React.useContext(LocationContext)
    const selectionContext = React.useContext(SelectionContext)
    const isMarqueeDragging = React.useContext(MarqueeDragContext)
    const isSelected = selectionContext.selected.map((obj) => obj.path).includes(props.data.FullPath)
    const selfRef = React.useRef<HTMLDivElement>(null)

    useDoubleClick({
        onSingleClick: (e?: MouseEvent) => {
          if (e?.ctrlKey || e?.metaKey) selectionContext.handle(props.data.FullPath, true)
        },
        onDoubleClick: download,
        ref: selfRef
    })

    const [menuOpen, setMenuOpen] = React.useState(false)
    const [anchorElement, setAnchorElement] = React.useState<HTMLElement | null>(null)

    function rightClick(event: React.MouseEvent): void {
        event.preventDefault()
        setMenuOpen(true)
        setAnchorElement(event.currentTarget as HTMLElement)
    }

    function close(): void {
        setMenuOpen(false)
        setAnchorElement(null)
    }

    async function download(): Promise<void> {
        close()
        const blob = await Filer.getRawContent(props.data.FullPath)
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = props.data.name
        a.click()
        URL.revokeObjectURL(url)
    }

    async function del(): Promise<void> {
        await Filer.deleteItem(props.data.FullPath)
        context.refresh()
    }

    const profile = React.useContext(ProfileContext)

    function favorite(): void {
        profile.addBookmark({
            fullPath: props.data.FullPath,
            shortName: props.data.name,
            isFile: true
        })
        close()
    }

    function copy(): void {
        if (isSelected && selectionContext.selected.length > 1) {
          selectionContext.copy()
        } else {
          selectionContext.copy({ path: props.data.FullPath, isFile: true })
        }
    }

    const [deleteItemOpen, setDeleteItemOpen] = React.useState(false)

    function onDeleteClick(): void {
        if (selectionContext.selected.length >= 1) {
          selectionContext.openDeleteDialog()
        } else {
          setDeleteItemOpen(true)
        }
    }

    return (
        <Grid
            item
            xs={3}
        >
            <Card
                ref={selfRef}
                data-selection-card
                data-file-path={props.data.FullPath}
                data-is-file="true"
                sx={{
                    display: "flex",
                    position: 'relative',
                    background: isSelected ? theme.palette.action.selected : theme.palette.background.paper,
                    '&:hover': {
                        cursor: isMarqueeDragging ? 'crosshair' : 'pointer',
                        background: theme.palette.action.hover
                    }
                }}
                onContextMenu={rightClick}
                aria-label={`${isSelected ? "selected " : ''}${props.data.name}`}
                role="button"
            >
                <IconButton
                    size="small"
                    onClick={(e) => { e.stopPropagation(); selectionContext.handle(props.data.FullPath, true) }}
                    sx={{ position: 'absolute', top: 4, left: 4, p: 0.25, zIndex: 1 }}
                    aria-label={isSelected ? 'deselect' : 'select'}
                >
                    {isSelected ? <CheckBoxIcon sx={{ fontSize: 18 }} /> : <CheckBoxOutlineBlankIcon sx={{ fontSize: 18 }} />}
                </IconButton>
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                }}
                >
                    <CardContent
                        sx={{
                            flex: "1"
                        }}
                    >
                        <DescriptionIcon
                            sx={{
                                fontSize: "56px"
                            }}
                        />
                    </CardContent>
                    <CardContent
                        sx={{
                            flex: "auto",
                            userSelect: "none",
                            overflow: "hidden",
                            textOverflow: "ellipsis",
                            display: "inline"
                        }}
                    >
                        <Typography
                            variant="h6"
                            noWrap
                        >
                            {props.data.name}
                        </Typography>
                        <Typography
                            variant="subtitle2"
                            noWrap
                        >
                            {String(props.data.Mtime ?? '')}
                        </Typography>
                    </CardContent>
                </Box>
            </Card>
            <RightClickMenu
                open={menuOpen}
                close={close}
                anchorElement={anchorElement}
                download={download}
                del={del}
                name={props.data.name}
                favorite={favorite}
                copy={copy}
                onDeleteClick={onDeleteClick}
                deleteItemOpen={deleteItemOpen}
                closeDelete={() => setDeleteItemOpen(false)}
            />
        </Grid>
    )

}

export default FileCard