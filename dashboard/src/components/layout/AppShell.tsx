import { useState } from 'react';
import { Box, CssBaseline } from '@mui/material';
import { Outlet } from 'react-router-dom';
import AppBar from './AppBar.tsx';
import Drawer from './Drawer.tsx';

// Drawer width constant
const DRAWER_WIDTH = 240;

export default function AppShell() {
  const [open, setOpen] = useState(true);

  const toggleDrawer = () => {
    setOpen(!open);
  };

  return (
    <Box sx={{ display: 'flex', height: '100%' }}>
      <CssBaseline />

      {/* Top app bar */}
      <AppBar open={open} drawerWidth={DRAWER_WIDTH} onDrawerToggle={toggleDrawer} />

      {/* Side navigation drawer */}
      <Drawer open={open} drawerWidth={DRAWER_WIDTH} onDrawerToggle={toggleDrawer} />

      {/* Main content */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 0,
          width: { sm: `calc(100% - ${DRAWER_WIDTH}px)` },
          ml: { sm: open ? `${DRAWER_WIDTH}px` : 0 },
          transition: theme => theme.transitions.create('margin', {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
          }),
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          pt: '64px', // AppBar height
          backgroundColor: theme =>
            theme.palette.mode === 'light'
              ? '#f5f5f9'
              : theme.palette.background.default,
          marginLeft: '0 !important',
          boxSizing: 'border-box',
        }}
      >
        <Box sx={{
          overflowY: 'auto',
          flexGrow: 1,
          p: 0
        }}>
          <Outlet />
        </Box>
      </Box>
    </Box>
  );
}