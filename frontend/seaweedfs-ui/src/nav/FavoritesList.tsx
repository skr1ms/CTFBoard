import React from 'react'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import Collapse from '@mui/material/Collapse'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import BookmarksIcon from '@mui/icons-material/Bookmarks'
import BookmarkRemoveIcon from '@mui/icons-material/BookmarkRemove'
import ExpandLess from '@mui/icons-material/ExpandLess'
import ExpandMore from '@mui/icons-material/ExpandMore'
import DescriptionIcon from '@mui/icons-material/Description'
import { Folder as FolderIcon } from '@mui/icons-material'
import { LocationContext } from '../context/LocationContextWrapper'
import { connectionString } from '../seaweedfs/filer'
import { ProfileContext } from '../context/ProfileContextWrapper'
import type { Bookmark } from '../types/context'

interface FavoritesRightClickMenuProps {
  open: boolean
  close: () => void
  index: number
  anchorElement: HTMLElement | null
}

function RightClickMenu(props: FavoritesRightClickMenuProps): React.ReactElement {
    const { open, close, index, anchorElement } = props
    const profile = React.useContext(ProfileContext)

    function remove(): void {
        profile.removeBookmark(index)
        close()
    }

    return (
        <Menu
            aria-label="favorite item context menu"
            role="menu"
            anchorEl={anchorElement}
            open={open}
            onClose={close}
        >
            <MenuItem
                onClick={remove}
                aria-label="remove favorite item"
            >
                <ListItemIcon>
                    <BookmarkRemoveIcon />
                </ListItemIcon>
                <ListItemText>Remove From Favorites</ListItemText>
            </MenuItem>
        </Menu>
    )
}

interface FavoriteItemProps {
  data: Bookmark
  index: number
}

function FavoriteItem(props: FavoriteItemProps): React.ReactElement {
    const { data, index } = props
    const context = React.useContext(LocationContext)
    const [menuOpen, setMenuOpen] = React.useState(false)
    const [anchorElement, setAnchorElement] = React.useState<HTMLElement | null>(null)

    function handleClick(): void {
        if (data.isFile) window.open(`${connectionString}${data.fullPath}`, '_blank')
        else context.updateLocation(data.fullPath)
    }

    function rightClick(event: React.MouseEvent<HTMLElement>): void {
        event.preventDefault()
        setMenuOpen(true)
        setAnchorElement(event.currentTarget as HTMLElement)
    }

    function close() {
        setMenuOpen(false)
        setAnchorElement(null)
    }

    return (
        <React.Fragment>
            <ListItem
                button
                sx={{ pl: 4 }}
                onClick={handleClick}
                onContextMenu={rightClick}
            >
                <ListItemIcon>
                    {data.isFile ? <DescriptionIcon /> : <FolderIcon />}
                </ListItemIcon>
                <ListItemText
                    primary={data.shortName}
                />
            </ListItem>
            <RightClickMenu
                open={menuOpen}
                close={close}
                index={index}
                anchorElement={anchorElement}
            />
        </React.Fragment>
    )

}

function FavoritesList(): React.ReactElement {
    const [open, setOpen] = React.useState(false)
    const profile = React.useContext(ProfileContext)
    const prevLengthRef = React.useRef(profile.bookmarks.length)

    React.useEffect(() => {
        if (prevLengthRef.current === 0 && profile.bookmarks.length > 0) setOpen(true)
        prevLengthRef.current = profile.bookmarks.length
    }, [profile.bookmarks.length])

    return (
        <List>
            <ListItem
                button
                onClick={() => setOpen(!open)}
            >
                <ListItemIcon>
                    <BookmarksIcon />
                </ListItemIcon>
                <ListItemText
                    primary="Favorites"
                />
                {open ? <ExpandLess titleAccess="close favorites" /> : <ExpandMore />}
            </ListItem>
            <Collapse
                in={open}
                timeout="auto"
                unmountOnExit
            >
                <List
                    disablePadding
                >
                    {profile.bookmarks.map((favorite, index) => <FavoriteItem data={favorite} index={index} key={index} />)}
                </List>
            </Collapse>
        </List>
    )
}

export default FavoritesList
export { FavoriteItem }