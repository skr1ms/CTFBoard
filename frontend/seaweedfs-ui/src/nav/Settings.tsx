import React from 'react'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import ListItemText from '@mui/material/ListItemText'
import ListItemIcon from '@mui/material/ListItemIcon'
import IconButton from '@mui/material/IconButton'
import Divider from '@mui/material/Divider'
import SettingsIcon from '@mui/icons-material/Settings'
import HelpIcon from '@mui/icons-material/Help'
import StorageIcon from '@mui/icons-material/Storage'
import InfoIcon from '@mui/icons-material/Info'
import LogoutIcon from '@mui/icons-material/Logout'
import MoreVertIcon from '@mui/icons-material/MoreVert'

import { AuthContext } from '../context/AuthContextWrapper'
import ClusterInfoDialog from '../dialogs/ClusterInfoDialog'
import AboutDialog from '../dialogs/AboutDialog'
import SettingsDialog from '../dialogs/SettingsDialog'

interface SettingsMenuProps {
  open: boolean
  close: () => void
  anchorElement: HTMLElement | null
}

function SettingsMenu(props: SettingsMenuProps): React.ReactElement {
    const { open, close, anchorElement } = props
    const auth = React.useContext(AuthContext)
    const [settingsOpen, setSettingsOpen] = React.useState(false)
    const [clusterInfoOpen, setClusterInfoOpen] = React.useState(false)
    const [aboutOpen, setAboutOpen] = React.useState(false)

    function openHelp(): void {
        window.open('https://github.com/seaweedfs/seaweedfs/wiki', '_blank')
    }

    function openSettings(): void {
        close()
        setSettingsOpen(true)
    }

    function closeSettings(): void {
        setSettingsOpen(false)
    }

    function openClusterInfo(): void {
        close()
        setClusterInfoOpen(true)
    }

    function closeClusterInfo(): void {
        setClusterInfoOpen(false)
    }

    function openAbout(): void {
        close()
        setAboutOpen(true)
    }

    function closeAbout(): void {
        setAboutOpen(false)
    }

    function handleLogout(): void {
        close()
        auth.logout()
    }

    return (
        <React.Fragment>
            <Menu
                aria-label="settings menu"
                role="menu"
                anchorEl={anchorElement}
                open={open}
                onClose={close}
            >
                <MenuItem
                    aria-label="open settings"
                    onClick={openSettings}
                >
                    <ListItemIcon>
                        <SettingsIcon />
                    </ListItemIcon>
                    <ListItemText>Settings</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem
                    onClick={openHelp}
                    aria-label="help page"
                >
                    <ListItemIcon>
                        <HelpIcon />
                    </ListItemIcon>
                    <ListItemText>Help</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem
                    aria-label="open cluster info"
                    onClick={openClusterInfo}
                >
                    <ListItemIcon>
                        <StorageIcon />
                    </ListItemIcon>
                    <ListItemText>Cluster Info</ListItemText>
                </MenuItem>
                <MenuItem
                    aria-label="open about"
                    onClick={openAbout}
                >
                    <ListItemIcon>
                        <InfoIcon />
                    </ListItemIcon>
                    <ListItemText>About</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem
                    aria-label="logout"
                    onClick={handleLogout}
                >
                    <ListItemIcon>
                        <LogoutIcon />
                    </ListItemIcon>
                    <ListItemText>Logout</ListItemText>
                </MenuItem>
            </Menu>
            <ClusterInfoDialog
                open={clusterInfoOpen}
                close={closeClusterInfo}
            />
            <AboutDialog
                open={aboutOpen}
                close={closeAbout}
            />
            <SettingsDialog
                open={settingsOpen}
                close={closeSettings}
            />
        </React.Fragment>
    )
}

function Settings(): React.ReactElement {
    const [anchorElement, setAnchorElement] = React.useState<HTMLElement | null>(null)
    const [menuOpen, setMenuOpen] = React.useState(false)

    function openMenu(event: React.MouseEvent<HTMLButtonElement>): void {
        setAnchorElement(event.currentTarget)
        setMenuOpen(true)
    }

    function closeMenu(): void {
        setMenuOpen(false)
    }

    return (
        <React.Fragment>
            <IconButton
                onClick={openMenu}
                aria-label="settings"
            >
                <MoreVertIcon
                    fontSize="large"
                />
            </IconButton>
            <SettingsMenu
                open={menuOpen}
                close={closeMenu}
                anchorElement={anchorElement}
            />
        </React.Fragment>
    )
}

export default Settings