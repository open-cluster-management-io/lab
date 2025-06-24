import type { ReactNode } from 'react';
import { Box, Typography, Paper, Button } from '@mui/material';
import { Link } from 'react-router-dom';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';

interface PageLayoutProps {
  children: ReactNode;
  title: string;
  backLink?: string;
  backLabel?: string;
  actions?: ReactNode;
}

/**
 * Layout component for displaying content in a full page
 */
export default function PageLayout({
  children,
  title,
  backLink = '/clusters',
  backLabel = 'Back to Clusters',
  actions
}: PageLayoutProps) {
  return (
    <Box sx={{ p: 3, height: '100%' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3, justifyContent: 'space-between' }}>
        <Box>
          {backLink && (
            <Button
              component={Link}
              to={backLink}
              startIcon={<ArrowBackIcon />}
              sx={{ mb: 2 }}
              color="inherit"
            >
              {backLabel}
            </Button>
          )}
          <Typography variant="h5" sx={{ fontWeight: 'bold' }}>
            {title}
          </Typography>
        </Box>
        {actions && (
          <Box>
            {actions}
          </Box>
        )}
      </Box>

      <Paper
        sx={{
          p: 3,
          borderRadius: 2,
          bgcolor: (theme) => (theme.palette.mode === 'light' ? 'white' : 'background.paper'),
        }}
      >
        {children}
      </Paper>
    </Box>
  );
}
