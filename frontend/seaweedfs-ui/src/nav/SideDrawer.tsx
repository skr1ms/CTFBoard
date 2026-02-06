import React from 'react'
import { useTheme } from '@mui/material/styles'
import Drawer from '@mui/material/Drawer'
import Divider from '@mui/material/Divider'
import Toolbar from '@mui/material/Toolbar'
import ActionsList from './ActionsList'
import FavoritesList from './FavoritesList'


interface DrawerHeaderProps {
  children: React.ReactNode
}

function DrawerHeader(props: DrawerHeaderProps): React.ReactElement {
    const theme = useTheme()
    return (
        <Toolbar
            sx={{
                display: 'flex',
                alignItems: 'center',
                padding: theme.spacing(0, 1),
                justifyContent: 'flex-end',
            }}
        >
            {props.children}
        </Toolbar>
    )
}

function SideDrawer(): React.ReactElement {

    return (
        <Drawer
            sx={{
                width: "240px",
                flexShrink: 0,
                '& .MuiDrawer-paper': {
                    width: "240px",
                    boxSizing: 'border-box',
                },
            }}
            variant="persistent"
            anchor="left"
            open={true}
        >
            <DrawerHeader>{null}</DrawerHeader>
            <Divider />
            <FavoritesList />
            <Divider />
            <ActionsList />
        </Drawer>
    )
}

export default SideDrawer
export {
    DrawerHeader
}