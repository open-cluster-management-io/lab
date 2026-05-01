import {
  Drawer as MuiDrawer,
  Toolbar,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  IconButton,
  Typography,
  Box,
  styled
} from '@mui/material';
import type { Theme } from '@mui/material';
import { useLocation, useNavigate } from 'react-router-dom';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import DashboardIcon from '@mui/icons-material/Dashboard';
import StorageIcon from '@mui/icons-material/Storage';
import DeviceHubIcon from '@mui/icons-material/DeviceHub';
import LayersIcon from '@mui/icons-material/Layers';
import TimelineIcon from '@mui/icons-material/Timeline';

interface DrawerProps {
  open: boolean;
  drawerWidth: number;
  onDrawerToggle: () => void;
}

const DrawerStyled = styled(MuiDrawer, {
  shouldForwardProp: (prop) => prop !== 'open' && prop !== 'drawerWidth',
})<{
  open: boolean;
  drawerWidth: number;
}>(({ theme, open, drawerWidth }) => ({
  '& .MuiDrawer-paper': {
    position: 'relative',
    whiteSpace: 'nowrap',
    width: drawerWidth,
    backgroundColor: '#ffffff',
    borderRight: '1px solid #e5e7ea',
    boxShadow: '0 1px 3px 0 rgba(0,0,0,0.1), 0 1px 2px 0 rgba(0,0,0,0.06)',
    transition: theme.transitions.create('width', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.enteringScreen,
    }),
    boxSizing: 'border-box',
    ...(!open && {
      overflowX: 'hidden',
      transition: theme.transitions.create('width', {
        easing: theme.transitions.easing.sharp,
        duration: theme.transitions.duration.leavingScreen,
      }),
      width: theme.spacing(7),
      [theme.breakpoints.up('sm')]: {
        width: theme.spacing(9),
      },
    }),
  },
}));

const navItems = [
  { text: 'Overview', icon: <DashboardIcon />, path: '/overview' },
  { text: 'Clusters', icon: <StorageIcon />, path: '/clusters' },
  { text: 'Clustersets', icon: <LayersIcon />, path: '/clustersets' },
  { text: 'Placements', icon: <DeviceHubIcon />, path: '/placements' },
  { text: 'Activity', icon: <TimelineIcon />, path: '/activity' },
];

export default function Drawer({ open, drawerWidth, onDrawerToggle }: DrawerProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const currentPath = location.pathname;

  const handleNavigation = (path: string) => {
    navigate(path);
  };

  return (
    <DrawerStyled
      variant="permanent"
      open={open}
      drawerWidth={drawerWidth}
      sx={{
        width: drawerWidth,
        flexShrink: 0,
        whiteSpace: 'nowrap',
        boxSizing: 'border-box',
        ...(open && {
          width: drawerWidth,
        }),
        ...(!open && {
          width: (theme: Theme) => theme.spacing(7),
          [`@media (min-width: ${(theme: Theme) => theme.breakpoints.values.sm}px)`]: {
            width: (theme: Theme) => theme.spacing(9),
          },
        }),
      }}
    >
      <Toolbar
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-end',
          px: [1],
        }}
      >
        <IconButton onClick={onDrawerToggle}>
          <ChevronLeftIcon />
        </IconButton>
      </Toolbar>

      {open && (
        <Box sx={{ px: 2.5, py: 2, bgcolor: '#f8f9fa', borderBottom: '1px solid #e5e7ea' }}>
          <Typography
            sx={{
              fontSize: '12px',
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.5px',
              color: '#495057',
            }}
          >
            Navigation
          </Typography>
        </Box>
      )}

      <List component="nav" sx={{ px: open ? 1.5 : 0, pt: 1 }}>
        {navItems.map((item) => {
          const isActive = currentPath === item.path ||
            (item.path !== '/overview' && currentPath.startsWith(item.path + '/'));

          return (
            <ListItem key={item.text} disablePadding sx={{ display: 'block', mb: 0.5 }}>
              <ListItemButton
                onClick={() => handleNavigation(item.path)}
                sx={{
                  minHeight: 48,
                  justifyContent: open ? 'initial' : 'center',
                  px: 2,
                  borderRadius: '8px',
                  position: 'relative',
                  overflow: 'hidden',
                  transition: 'all 0.2s ease',
                  ...(isActive
                    ? {
                        background: 'linear-gradient(90deg, #0066cc 0%, #2b9af3 100%)',
                        color: 'white',
                        boxShadow: '0 1px 3px 0 rgba(0,0,0,0.1), 0 1px 2px 0 rgba(0,0,0,0.06)',
                        '&::before': {
                          content: '""',
                          position: 'absolute',
                          left: 0,
                          top: 0,
                          bottom: 0,
                          width: '4px',
                          bgcolor: '#ee0000',
                          borderRadius: '0 2px 2px 0',
                        },
                        '&:hover': {
                          background: 'linear-gradient(90deg, #0055aa 0%, #1a8ae3 100%)',
                        },
                      }
                    : {
                        color: '#495057',
                        '&:hover': {
                          bgcolor: '#f1f2f3',
                          color: '#0066cc',
                          transform: 'translateX(2px)',
                        },
                      }),
                }}
              >
                <ListItemIcon
                  sx={{
                    minWidth: 0,
                    mr: open ? 2.5 : 'auto',
                    justifyContent: 'center',
                    color: isActive ? 'white' : '#495057',
                  }}
                >
                  {item.icon}
                </ListItemIcon>
                <ListItemText
                  primary={item.text}
                  sx={{
                    opacity: open ? 1 : 0,
                    '& .MuiTypography-root': {
                      fontWeight: isActive ? 600 : 500,
                      fontSize: '14px',
                    },
                  }}
                />
              </ListItemButton>
            </ListItem>
          );
        })}
      </List>
    </DrawerStyled>
  );
}
