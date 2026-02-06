import React from 'react'
import { useTheme } from '@mui/material'
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
import { Folder as FolderIcon } from '@mui/icons-material'
import CheckBoxOutlineBlankIcon from '@mui/icons-material/CheckBoxOutlineBlank'
import CheckBoxIcon from '@mui/icons-material/CheckBox'
import OpenInBrowserIcon from '@mui/icons-material/OpenInBrowser'
import InfoIcon from '@mui/icons-material/Info'
import DeleteIcon from '@mui/icons-material/Delete'
import BookmarkIcon from '@mui/icons-material/Bookmark'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import useDoubleClick from 'use-double-click'
import { LocationContext } from '../../context/LocationContextWrapper'
import Filer from '../../seaweedfs/filer'
import DeleteItemConfirmation from '../../dialogs/DeleteItemConfirmation'
import { SelectionContext } from '../../context/SelectionContextWrapper'
import { MarqueeDragContext } from './CardView'
import { ProfileContext } from '../../context/ProfileContextWrapper'
import type { FileEntry } from '../../types/models'

interface FolderCardRightClickMenuProps {
  open: boolean
  close: () => void
  anchorElement: HTMLElement | null
  enter: () => void
  del: () => void
  name: string
  favorite: () => void
  copy: () => void
  onDeleteClick: () => void
  deleteItemOpen: boolean
  closeDelete: () => void
}

function RightClickMenu(props: FolderCardRightClickMenuProps): React.ReactElement {
    const { open, close, enter, anchorElement, del, name, favorite, copy, onDeleteClick, deleteItemOpen, closeDelete } = props

    return (
        <React.Fragment>
            <Menu aria-label="folder context menu" role="menu" anchorEl={anchorElement} open={open} onClose={close}>
                <MenuItem onClick={enter} aria-label="open folder">
                    <ListItemIcon><OpenInBrowserIcon /></ListItemIcon>
                    <ListItemText>Open</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem onClick={favorite} aria-label="favorite folder">
                    <ListItemIcon><BookmarkIcon /></ListItemIcon>
                    <ListItemText>Favorite</ListItemText>
                </MenuItem>
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
            <DeleteItemConfirmation name={name} del={del} close={closeDelete} open={deleteItemOpen} isFile={false} />
        </React.Fragment>
    )
}


interface FolderCardProps {
  data: FileEntry
}

function FolderCard(props: FolderCardProps): React.ReactElement {
    const theme = useTheme()
    const context = React.useContext(LocationContext)
    const selectionContext = React.useContext(SelectionContext)
    const isMarqueeDragging = React.useContext(MarqueeDragContext)
    const isSelected = selectionContext.selected.map((obj) => obj.path).includes(props.data.FullPath)
    const selfRef = React.useRef<HTMLDivElement>(null)

    useDoubleClick({
        onSingleClick: (e?: MouseEvent) => {
          if (props.data.name === '..') return
          if (e?.ctrlKey || e?.metaKey) selectionContext.handle(props.data.FullPath, false)
        },
        onDoubleClick: enter,
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

    function enter(): void {
        close()
        context.updateLocation(props.data.FullPath)
    }

    async function del(): Promise<void> {
        await Filer.deleteItem(props.data.FullPath, true)
        context.refresh()
    }

    const profile = React.useContext(ProfileContext)

    function favorite(): void {
        profile.addBookmark({
            fullPath: props.data.FullPath,
            shortName: props.data.name,
            isFile: false
        })
        close()
    }

    function copy(): void {
        if (isSelected && selectionContext.selected.length > 1) {
          selectionContext.copy()
        } else {
          selectionContext.copy({ path: props.data.FullPath, isFile: false })
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

    return(
        <Grid 
            item 
            xs={3}

        >
            <Card
                ref={selfRef}
                data-selection-card
                data-file-path={props.data.FullPath}
                data-is-file="false"
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
                    onClick={(e) => { e.stopPropagation(); selectionContext.handle(props.data.FullPath, false) }}
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
                        <FolderIcon
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
                enter={enter}
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

export default FolderCard