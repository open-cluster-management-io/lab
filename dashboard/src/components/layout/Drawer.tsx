import {
  Drawer as MuiDrawer,
  Toolbar,
  Divider,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  IconButton,
  styled
} from '@mui/material';
import type { Theme } from '@mui/material';
import { useLocation, useNavigate } from 'react-router-dom';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import DashboardIcon from '@mui/icons-material/Dashboard';
import StorageIcon from '@mui/icons-material/Storage';
import DeviceHubIcon from '@mui/icons-material/DeviceHub';
import LayersIcon from '@mui/icons-material/Layers';

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

// Navigation items with paths
const navItems = [
  { text: 'Overview', icon: <DashboardIcon />, path: '/overview' },
  { text: 'Clusters', icon: <StorageIcon />, path: '/clusters' },
  { text: 'Clustersets', icon: <LayersIcon />, path: '/clustersets' },
  { text: 'Placements', icon: <DeviceHubIcon />, path: '/placements' },
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
      <Divider />
      <List component="nav">
        {navItems.map((item) => {
          const isActive = currentPath === item.path ||
            (item.path !== '/overview' && currentPath.startsWith(item.path));

          return (
            <ListItem key={item.text} disablePadding sx={{ display: 'block' }}>
              <ListItemButton
                onClick={() => handleNavigation(item.path)}
                sx={{
                  minHeight: 48,
                  justifyContent: open ? 'initial' : 'center',
                  px: 2.5,
                }}
              >
                <ListItemIcon
                  sx={{
                    minWidth: 0,
                    mr: open ? 3 : 'auto',
                    justifyContent: 'center',
                    color: isActive ? 'primary.main' : 'inherit',
                  }}
                >
                  {item.icon}
                </ListItemIcon>
                <ListItemText
                  primary={item.text}
                  sx={{
                    opacity: open ? 1 : 0,
                    color: isActive ? 'primary.main' : 'inherit',
                    '& .MuiTypography-root': {
                      fontWeight: isActive ? 'bold' : 'normal',
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
