import React from 'react'
import JSZip from 'jszip'
import { LocationContext } from './LocationContextWrapper'
import DeleteMultipleDialog from '../dialogs/DeleteMultipleDialog'
import Filer from '../seaweedfs/filer'
import { getName } from '../seaweedfs/file'
import SuccessAlert from '../snackbars/SuccessAlert'
import type { SelectionContextValue, SelectedItem, ClipboardState } from '../types/context'

const defaultClipboard: ClipboardState = {
  content: [],
  method: 'copy',
}

const defaultSelectionContext: SelectionContextValue = {
  selected: [],
  handle: () => {},
  replaceSelection: () => {},
  clipboard: defaultClipboard,
  cut: () => {},
  copy: () => {},
  paste: async () => {},
  del: async () => {},
  deleteDialogOpen: false,
  openDeleteDialog: () => {},
  closeDeleteDialog: () => {},
  showSuccess: () => {},
  downloadSelected: async () => {},
}

const SelectionContext = React.createContext<SelectionContextValue>(defaultSelectionContext)

interface SelectionContextWrapperProps {
  children: React.ReactNode
}

function SelectionContextWrapper(props: SelectionContextWrapperProps) {
  const [selected, _setSelected] = React.useState<SelectedItem[]>([])
  const locationContext = React.useContext(LocationContext)
  const locationContextRef = React.useRef(locationContext)
  locationContextRef.current = locationContext

  const selectedRef = React.useRef(selected)
  selectedRef.current = selected

  const setSelected = React.useCallback((selectedList: SelectedItem[]) => {
    selectedRef.current = selectedList
    _setSelected(selectedList)
  }, [])

  const handle = React.useCallback((path: string, isFile: boolean) => {
    const prev = selectedRef.current
    const tempPaths = prev.map((obj) => obj.path)
    let next: SelectedItem[]
    if (tempPaths.includes(path)) {
      const index = tempPaths.indexOf(path)
      next = [...prev]
      next.splice(index, 1)
    } else {
      next = [...prev, { path, isFile }]
    }
    setSelected(next)
  }, [setSelected])

  const replaceSelection = React.useCallback((items: SelectedItem[]) => {
    setSelected(items)
  }, [setSelected])

  React.useEffect(() => {
    _setSelected([])
  }, [locationContext.currentLocation])

  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)

  const del = React.useCallback(async () => {
    const toDelete = [...selectedRef.current]
    setSelected([])
    setDeleteDialogOpen(false)
    for (const item of toDelete) {
      await Filer.deleteItem(item.path, !item.isFile)
    }
    locationContext.refresh()
  }, [setSelected, locationContext])

  const showSuccess = React.useCallback((text: string) => {
    setAlertText(text)
    setAlertOpen(true)
  }, [])

  const downloadSelected = React.useCallback(async () => {
    const items = selectedRef.current.filter((o) => o.isFile)
    if (items.length === 0) return
    if (items.length === 1) {
      const blob = await Filer.getRawContent(items[0].path)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = getName(items[0].path)
      a.click()
      URL.revokeObjectURL(url)
      return
    }
    const zip = new JSZip()
    for (const item of items) {
      const blob = await Filer.getRawContent(item.path)
      zip.file(getName(item.path), blob)
    }
    const blob = await zip.generateAsync({ type: 'blob' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'download.zip'
    a.click()
    URL.revokeObjectURL(url)
  }, [])

  const [alertOpen, setAlertOpen] = React.useState(false)
  const [alertText, setAlertText] = React.useState('')
  const [clipboard, _setClipboard] = React.useState<ClipboardState>(defaultClipboard)
  const clipboardRef = React.useRef(clipboard)

  const setClipboard = React.useCallback((obj: ClipboardState) => {
    clipboardRef.current = obj
    _setClipboard(obj)
  }, [])

  const closeAlert = React.useCallback(() => {
    setAlertOpen(false)
    locationContext.refresh()
  }, [locationContext])

  const copy = React.useCallback((obj: SelectedItem | null = null) => {
    const content = obj ? [obj] : [...selectedRef.current]
    const count = content.length
    if (count === 0) return
    setClipboard({ content, method: 'copy' })
    setAlertText(`Copied ${count} item${count !== 1 ? 's' : ''}!`)
    setAlertOpen(true)
    setSelected([])
  }, [setClipboard, setSelected])

  const cut = React.useCallback((obj: SelectedItem | null = null) => {
    const content = obj ? [obj] : [...selectedRef.current]
    const count = content.length
    if (count === 0) return
    setClipboard({ content, method: 'cut' })
    setAlertText(`Cut ${count} item${count !== 1 ? 's' : ''}!`)
    setAlertOpen(true)
    setSelected([])
  }, [setClipboard, setSelected])

  const paste = React.useCallback(async () => {
    const content = [...clipboardRef.current.content]
    const method = clipboardRef.current.method
    let pasted = 0
    for (const obj of content) {
      const destDir = locationContextRef.current.currentLocation
      if (obj.isFile) {
        await Filer.copyFile(obj.path, destDir)
        pasted++
        if (method === 'cut') await Filer.deleteItem(obj.path)
      } else {
        await Filer.copyDirectory(obj.path, destDir)
        pasted++
        if (method === 'cut') await Filer.deleteItem(obj.path, true)
      }
    }
    setAlertText(pasted > 0 ? `Pasted ${pasted} item${pasted !== 1 ? 's' : ''}!` : 'No items to paste.')
    setAlertOpen(true)
    locationContextRef.current.refresh()
    setClipboard(defaultClipboard)
  }, [setClipboard])

  const handleCutCopyPaste = React.useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'c') copy()
      else if (event.key === 'x') cut()
      else if (event.key === 'v') paste()
    },
    [copy, cut, paste]
  )

  const handleCutCopyPasteRef = React.useRef(handleCutCopyPaste)
  handleCutCopyPasteRef.current = handleCutCopyPaste

  React.useEffect(() => {
    function handleKeydown(event: KeyboardEvent) {
      if (event.repeat) return
      if (event.key === 'Escape') {
        setSelected([])
      } else if (event.code === 'Delete' && selectedRef.current.length) {
        setDeleteDialogOpen(true)
      } else if (event.ctrlKey) {
        handleCutCopyPasteRef.current(event)
      }
    }
    document.addEventListener('keydown', handleKeydown)
    return () => document.removeEventListener('keydown', handleKeydown)
  }, [setSelected])

  const value: SelectionContextValue = {
    selected,
    handle,
    replaceSelection,
    clipboard,
    cut,
    copy,
    paste,
    del,
    deleteDialogOpen,
    openDeleteDialog: () => setDeleteDialogOpen(true),
    closeDeleteDialog: () => setDeleteDialogOpen(false),
    showSuccess,
    downloadSelected,
  }

  return (
    <SelectionContext.Provider value={value}>
      {props.children}
      <DeleteMultipleDialog
        open={deleteDialogOpen}
        close={() => setDeleteDialogOpen(false)}
        files={selected.map((obj) => obj.path)}
        del={del}
      />
      <SuccessAlert open={alertOpen} close={closeAlert} text={alertText} />
    </SelectionContext.Provider>
  )
}

export default SelectionContextWrapper
export { SelectionContext }
