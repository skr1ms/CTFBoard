import React from 'react'
import useLocalStorage from '@rehooks/local-storage'
import Filer from '../seaweedfs/filer'
import { getName } from '../seaweedfs/file'
import type { ProfileContextValue, ProfileSettings, Bookmark } from '../types/context'

const defaultSettings: ProfileSettings = {
  showDotFiles: true,
  useDetailsView: false,
  useDarkMode: true,
}

const defaultProfile: ProfileContextValue = {
  settings: defaultSettings,
  bookmarks: [],
  options: [],
  current: 'localstorage',
  switchProfile: () => {},
  updateSetting: () => {},
  makeNewProfile: () => {},
  addBookmark: () => {},
  removeBookmark: () => {},
}

const ProfileContext = React.createContext<ProfileContextValue>(defaultProfile)

interface ProfileContextWrapperProps {
  children: React.ReactNode
}

function ProfileContextWrapper(props: ProfileContextWrapperProps) {
  const [settings, setSettings] = React.useState<ProfileSettings>(defaultSettings)
  const [bookmarks, setBookmarks] = React.useState<Bookmark[]>([])
  const [options, setOptions] = React.useState<string[]>([])
  const [selectedProfile, setSelectedProfile] = useLocalStorage<{ profile: string }>('profile', {
    profile: 'localstorage',
  })
  const [localSettings, setLocalSettings] = useLocalStorage<ProfileSettings>('settings', defaultSettings)
  const [localBookmarks, setLocalBookmarks] = useLocalStorage<Bookmark[]>('bookmarks', [])

  const usingLocalStorage = selectedProfile.profile === 'localstorage'

  const getProfiles = React.useCallback(async () => {
    const output = ['localstorage']
    const profileFiles = await Filer.getFiles('/.seaweedfs-ui/profiles')
    for (const f of profileFiles) {
      const name = getName(f.FullPath)
      if (name.includes('.json')) {
        output.push(name.replace('.json', ''))
      }
    }
    setOptions(output)
  }, [])

  React.useEffect(() => {
    async function loadProfile() {
      let tempSettings: ProfileSettings
      let tempBookmarks: Bookmark[]
      if (usingLocalStorage) {
        tempSettings = localSettings
        tempBookmarks = localBookmarks
      } else {
        const content = await Filer.getContent(
          `/.seaweedfs-ui/profiles/${selectedProfile.profile}.json`
        )
        let data: { settings?: ProfileSettings; bookmarks?: Bookmark[] }
        try {
          data = JSON.parse(content)
        } catch {
          data = { settings: defaultSettings, bookmarks: [] }
        }
        tempSettings = data.settings ?? defaultSettings
        tempBookmarks = data.bookmarks ?? []
      }
      setSettings(tempSettings)
      setBookmarks(tempBookmarks)
    }
    getProfiles()
    loadProfile()
  }, [selectedProfile.profile, localSettings, localBookmarks, usingLocalStorage, getProfiles])

  const switchProfile = React.useCallback((name: string) => {
    setSelectedProfile({ profile: name })
  }, [setSelectedProfile])

  const updateSetting = React.useCallback(
    async (setting: keyof ProfileSettings, value: boolean) => {
      const newSettings = { ...settings, [setting]: value }
      if (usingLocalStorage) {
        setLocalSettings(newSettings)
      } else {
        const content = { settings: newSettings, bookmarks }
        const fileName = `/.seaweedfs-ui/profiles/${selectedProfile.profile}.json`
        const file = new File([JSON.stringify(content)], fileName, {
          type: 'application/json',
        })
        await Filer.uploadFile(fileName, file)
        setSettings(newSettings)
      }
    },
    [settings, bookmarks, usingLocalStorage, selectedProfile.profile, setLocalSettings]
  )

  const makeNewProfile = React.useCallback(
    async (name: string) => {
      const content = { settings, bookmarks }
      const fileName = `/.seaweedfs-ui/profiles/${name}.json`
      const file = new File([JSON.stringify(content)], fileName, {
        type: 'application/json',
      })
      await Filer.uploadFile(fileName, file)
      setSelectedProfile({ profile: name })
    },
    [settings, bookmarks, setSelectedProfile]
  )

  const addBookmark = React.useCallback(
    async (bookmark: Bookmark) => {
      if (bookmarks.some((b) => b.fullPath === bookmark.fullPath)) return
      const newBookmarks = [...bookmarks, bookmark]
      if (usingLocalStorage) {
        setLocalBookmarks(newBookmarks)
      } else {
        const fileName = `/.seaweedfs-ui/profiles/${selectedProfile.profile}.json`
        const content = { settings, bookmarks: newBookmarks }
        const file = new File([JSON.stringify(content)], fileName, {
          type: 'application/json',
        })
        await Filer.uploadFile(fileName, file)
      }
      setBookmarks(newBookmarks)
    },
    [bookmarks, usingLocalStorage, selectedProfile.profile, settings, setLocalBookmarks]
  )

  const removeBookmark = React.useCallback(
    async (index: number) => {
      const newBookmarks = [...bookmarks]
      newBookmarks.splice(index, 1)
      if (usingLocalStorage) {
        setLocalBookmarks(newBookmarks)
      } else {
        const fileName = `/.seaweedfs-ui/profiles/${selectedProfile.profile}.json`
        const content = { settings, bookmarks: newBookmarks }
        const file = new File([JSON.stringify(content)], fileName, {
          type: 'application/json',
        })
        await Filer.uploadFile(fileName, file)
      }
      setBookmarks(newBookmarks)
    },
    [bookmarks, usingLocalStorage, selectedProfile.profile, settings, setLocalBookmarks]
  )

  const value: ProfileContextValue = {
    settings,
    bookmarks,
    options,
    current: selectedProfile.profile,
    switchProfile,
    updateSetting,
    makeNewProfile,
    addBookmark,
    removeBookmark,
  }

  return (
    <ProfileContext.Provider value={value}>
      {props.children}
    </ProfileContext.Provider>
  )
}

export default ProfileContextWrapper
export { defaultProfile, ProfileContext }
