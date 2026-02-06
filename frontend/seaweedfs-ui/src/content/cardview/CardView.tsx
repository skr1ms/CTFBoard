import React from 'react'
import Grid from '@mui/material/Grid'
import Box from '@mui/material/Box'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import Divider from '@mui/material/Divider'
import ContentPasteIcon from '@mui/icons-material/ContentPaste'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import DeleteIcon from '@mui/icons-material/Delete'
import DownloadForOfflineIcon from '@mui/icons-material/DownloadForOffline'
import NoteAddIcon from '@mui/icons-material/NoteAdd'
import CreateNewFolderIcon from '@mui/icons-material/CreateNewFolder'
import CloudUploadIcon from '@mui/icons-material/CloudUpload'
import FileCard from './FileCard'
import FolderCard from './FolderCard'
import ImageCard from './ImageCard'
import TextFileDialog from '../../dialogs/TextFileDialog'
import NewFolderDialog from '../../dialogs/NewFolderDialog'
import UploadFileDialog from '../../dialogs/UploadFileDialog'
import { useTheme } from '@mui/material'
import { SelectionContext } from '../../context/SelectionContextWrapper'
import type { FileEntry } from '../../types/models'
import type { SelectedItem } from '../../types/context'

const SUPPORTED_IMAGE_EXT = ['.png', '.jpg', '.jpeg', '.bmp', '.ico']
const MARQUEE_MIN_DELTA = 5

export const MarqueeDragContext = React.createContext(false)

function handleFileType(obj: FileEntry): React.ReactElement {
    const isImage = SUPPORTED_IMAGE_EXT.some((ext) => obj.name.includes(ext))
    if (isImage) return <ImageCard data={obj} key={obj.name} />
    return <FileCard data={obj} key={obj.name} />
}

function rectsOverlap(
  box: { left: number; top: number; right: number; bottom: number },
  rect: DOMRect
): boolean {
  return !(rect.right < box.left || rect.left > box.right || rect.bottom < box.top || rect.top > box.bottom)
}

interface CardViewProps {
  files: FileEntry[]
}

function CardView(props: CardViewProps): React.ReactElement {
    const theme = useTheme()
    const { files } = props
    const selectionContext = React.useContext(SelectionContext)
    const containerRef = React.useRef<HTMLDivElement>(null)
    const [dragStart, setDragStart] = React.useState<{ x: number; y: number } | null>(null)
    const [marquee, setMarquee] = React.useState<{
      startX: number
      startY: number
      currentX: number
      currentY: number
    } | null>(null)
    const marqueeRef = React.useRef(marquee)
    marqueeRef.current = marquee

    const [emptyAreaMenuAnchor, setEmptyAreaMenuAnchor] = React.useState<null | { x: number; y: number }>(null)
    const [createFileOpen, setCreateFileOpen] = React.useState(false)
    const [createFolderOpen, setCreateFolderOpen] = React.useState(false)
    const [uploadOpen, setUploadOpen] = React.useState(false)
    const hasClipboard = selectionContext.clipboard.content.length > 0
    const hasSelection = selectionContext.selected.length >= 1
    const hasFileSelection = selectionContext.selected.some((s) => s.isFile)

    const onContextMenu = React.useCallback((e: React.MouseEvent) => {
      if ((e.target as Element).closest?.('[data-selection-card]')) return
      e.preventDefault()
      setEmptyAreaMenuAnchor({ x: e.clientX, y: e.clientY })
    }, [])

    const onMouseDown = React.useCallback((e: React.MouseEvent) => {
      if (e.button !== 0) return
      const onCard = (e.target as Element).closest?.('[data-selection-card]')
      if (onCard) return
      selectionContext.replaceSelection([])
      setDragStart({ x: e.clientX, y: e.clientY })
    }, [selectionContext])

    React.useEffect(() => {
      if (!dragStart) return
      function updateSelectionForBox(left: number, right: number, top: number, bottom: number): void {
        const box = { left, top, right, bottom }
        const cards = containerRef.current?.querySelectorAll('[data-selection-card]') ?? []
        const items: SelectedItem[] = []
        cards.forEach((el) => {
          const path = el.getAttribute('data-file-path')
          const isFile = el.getAttribute('data-is-file') === 'true'
          if (!path) return
          const cell = el.parentElement ?? el
          const rect = cell.getBoundingClientRect()
          if (rectsOverlap(box, rect)) items.push({ path, isFile })
        })
        selectionContext.replaceSelection(items)
      }

      const onMouseMove = (e: MouseEvent): void => {
        const dx = e.clientX - dragStart.x
        const dy = e.clientY - dragStart.y
        if (Math.abs(dx) >= MARQUEE_MIN_DELTA || Math.abs(dy) >= MARQUEE_MIN_DELTA) {
          const next = marqueeRef.current
            ? { ...marqueeRef.current, currentX: e.clientX, currentY: e.clientY }
            : { startX: dragStart.x, startY: dragStart.y, currentX: e.clientX, currentY: e.clientY }
          marqueeRef.current = next
          setMarquee(next)
          const left = Math.min(next.startX, next.currentX)
          const right = Math.max(next.startX, next.currentX)
          const top = Math.min(next.startY, next.currentY)
          const bottom = Math.max(next.startY, next.currentY)
          updateSelectionForBox(left, right, top, bottom)
        }
      }
      const onMouseUp = (e: MouseEvent): void => {
        const current = marqueeRef.current
        if (current) {
          const dx = current.currentX - current.startX
          const dy = current.currentY - current.startY
          if (Math.abs(dx) >= MARQUEE_MIN_DELTA || Math.abs(dy) >= MARQUEE_MIN_DELTA) {
            const left = Math.min(current.startX, current.currentX)
            const right = Math.max(current.startX, current.currentX)
            const top = Math.min(current.startY, current.currentY)
            const bottom = Math.max(current.startY, current.currentY)
            updateSelectionForBox(left, right, top, bottom)
            e.preventDefault()
            e.stopPropagation()
          }
        }
        marqueeRef.current = null
        setDragStart(null)
        setMarquee(null)
      }
      document.addEventListener('mousemove', onMouseMove)
      document.addEventListener('mouseup', onMouseUp, true)
      return () => {
        document.removeEventListener('mousemove', onMouseMove)
        document.removeEventListener('mouseup', onMouseUp, true)
      }
    }, [dragStart, selectionContext])

    const children = []
    for (let obj of files) {
        if (obj.isFile) {
            let card = handleFileType(obj)
            children.push(card)
        }
        else {
            children.push(<FolderCard data={obj} key={obj.name} />)
        }
    }

    const boxStyle = marquee
      ? {
          left: Math.min(marquee.startX, marquee.currentX),
          top: Math.min(marquee.startY, marquee.currentY),
          width: Math.abs(marquee.currentX - marquee.startX),
          height: Math.abs(marquee.currentY - marquee.startY),
        }
      : null

    const isMarqueeActive = !!dragStart

    return (
        <MarqueeDragContext.Provider value={isMarqueeActive}>
        <Box
          ref={containerRef}
          onMouseDown={onMouseDown}
          onContextMenu={onContextMenu}
          sx={{
            marginLeft: '240px',
            width: 'calc(100vw - 240px)',
            minHeight: 'calc(100vh - 64px)',
            paddingTop: '7px',
            paddingRight: theme.spacing(1),
            position: 'relative',
            cursor: isMarqueeActive ? 'crosshair' : undefined,
          }}
        >
          <Grid container spacing={2}>
            {children}
          </Grid>
          {boxStyle && (
            <Box
              sx={{
                position: 'fixed',
                left: boxStyle.left,
                top: boxStyle.top,
                width: boxStyle.width,
                height: boxStyle.height,
                border: '2px solid',
                borderColor: 'primary.main',
                bgcolor: 'primary.main',
                opacity: 0.15,
                pointerEvents: 'none',
                zIndex: 1300,
              }}
            />
          )}
        </Box>
        <Menu
          open={!!emptyAreaMenuAnchor}
          onClose={() => setEmptyAreaMenuAnchor(null)}
          anchorReference="anchorPosition"
          anchorPosition={emptyAreaMenuAnchor ? { top: emptyAreaMenuAnchor.y, left: emptyAreaMenuAnchor.x } : undefined}
        >
          <MenuItem onClick={() => { setCreateFileOpen(true); setEmptyAreaMenuAnchor(null) }}>
            <ListItemIcon><NoteAddIcon /></ListItemIcon>
            <ListItemText>Create file</ListItemText>
          </MenuItem>
          <MenuItem onClick={() => { setCreateFolderOpen(true); setEmptyAreaMenuAnchor(null) }}>
            <ListItemIcon><CreateNewFolderIcon /></ListItemIcon>
            <ListItemText>Create folder</ListItemText>
          </MenuItem>
          <MenuItem onClick={() => { setUploadOpen(true); setEmptyAreaMenuAnchor(null) }}>
            <ListItemIcon><CloudUploadIcon /></ListItemIcon>
            <ListItemText>Upload files</ListItemText>
          </MenuItem>
          {hasSelection && (
            <>
              <Divider />
              <MenuItem onClick={() => { selectionContext.copy(); setEmptyAreaMenuAnchor(null) }}>
                <ListItemIcon><ContentCopyIcon /></ListItemIcon>
                <ListItemText>Copy</ListItemText>
              </MenuItem>
              {hasFileSelection && (
                <MenuItem onClick={() => { void selectionContext.downloadSelected(); setEmptyAreaMenuAnchor(null) }}>
                  <ListItemIcon><DownloadForOfflineIcon /></ListItemIcon>
                  <ListItemText>Download</ListItemText>
                </MenuItem>
              )}
              <MenuItem onClick={() => { selectionContext.openDeleteDialog(); setEmptyAreaMenuAnchor(null) }}>
                <ListItemIcon><DeleteIcon /></ListItemIcon>
                <ListItemText>Delete</ListItemText>
              </MenuItem>
            </>
          )}
          {hasClipboard && (
            <>
              <Divider />
              <MenuItem onClick={() => { void selectionContext.paste(); setEmptyAreaMenuAnchor(null) }}>
                <ListItemIcon><ContentPasteIcon /></ListItemIcon>
                <ListItemText>Paste</ListItemText>
              </MenuItem>
            </>
          )}
        </Menu>
        <TextFileDialog open={createFileOpen} close={() => setCreateFileOpen(false)} />
        <NewFolderDialog open={createFolderOpen} close={() => setCreateFolderOpen(false)} />
        <UploadFileDialog open={uploadOpen} close={() => setUploadOpen(false)} />
        </MarqueeDragContext.Provider>
    )
}

export default CardView