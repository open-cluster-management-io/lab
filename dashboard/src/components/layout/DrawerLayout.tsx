import type { ReactNode } from 'react';
import { Box, IconButton, Typography } from '@mui/material';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';

interface DrawerLayoutProps {
  children: ReactNode;
  title: string;
  icon?: ReactNode;
  onClose?: () => void;
}

/**
 * Layout component for displaying content in a drawer
 */
export default function DrawerLayout({ children, title, icon, onClose }: DrawerLayoutProps) {
  return (
    <Box
      sx={{
        flex: '0 0 40%',
        borderLeft: 1,
        borderColor: 'divider',
        p: 3,
        overflow: 'auto',
        backgroundColor: 'background.paper',
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', flex: 1 }}>
          {icon}
          <Typography variant="h6" sx={{ fontWeight: 'bold', ml: icon ? 1 : 0 }}>
            {title}
          </Typography>
        </Box>
        {onClose && (
          <IconButton onClick={onClose}>
            <ChevronLeftIcon />
          </IconButton>
        )}
      </Box>

      {children}
    </Box>
  );
}
