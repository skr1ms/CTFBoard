export interface LocationContextValue {
  currentLocation: string
  history: string[]
  refreshCount: number
  updateLocation: (path: string) => void
  goBack: () => void
  refresh: () => void
}

export interface ProfileSettings {
  showDotFiles: boolean
  useDetailsView: boolean
  useDarkMode: boolean
}

export interface Bookmark {
  fullPath: string
  shortName: string
  isFile: boolean
}

export interface ProfileContextValue {
  settings: ProfileSettings
  bookmarks: Bookmark[]
  options: string[]
  current: string
  switchProfile: (name: string) => void
  updateSetting: (setting: keyof ProfileSettings, value: boolean) => void
  makeNewProfile: (name: string) => void
  addBookmark: (bookmark: Bookmark) => void
  removeBookmark: (index: number) => void
}

export interface SelectedItem {
  path: string
  isFile: boolean
}

export interface ClipboardState {
  content: SelectedItem[]
  method: 'copy' | 'cut'
}

export interface SelectionContextValue {
  selected: SelectedItem[]
  handle: (path: string, isFile: boolean) => void
  replaceSelection: (items: SelectedItem[]) => void
  clipboard: ClipboardState
  cut: (obj?: SelectedItem | null) => void
  copy: (obj?: SelectedItem | null) => void
  paste: () => Promise<void>
  del: () => Promise<void>
  deleteDialogOpen: boolean
  openDeleteDialog: () => void
  closeDeleteDialog: () => void
  showSuccess: (text: string) => void
  downloadSelected: () => Promise<void>
}
