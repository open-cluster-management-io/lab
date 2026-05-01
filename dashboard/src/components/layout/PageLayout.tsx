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

export default function PageLayout({
  children,
  title,
  backLink = '/clusters',
  backLabel = 'Back to Clusters',
  actions
}: PageLayoutProps) {
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box
        sx={{
          bgcolor: 'white',
          borderBottom: '1px solid #e5e7ea',
          px: 4,
          py: 3,
          boxShadow: '0 1px 3px 0 rgba(0,0,0,0.1), 0 1px 2px 0 rgba(0,0,0,0.06)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Box>
            {backLink && (
              <Button
                component={Link}
                to={backLink}
                startIcon={<ArrowBackIcon />}
                sx={{ mb: 1, color: '#0066cc' }}
              >
                {backLabel}
              </Button>
            )}
            <Typography
              variant="h5"
              sx={{
                fontWeight: 700,
                color: '#1a1d21',
                fontFamily: "'Red Hat Display', 'Helvetica Neue', Arial, sans-serif",
              }}
            >
              {title}
            </Typography>
          </Box>
          {actions && <Box>{actions}</Box>}
        </Box>
      </Box>

      <Box sx={{ p: 3, flex: 1, overflow: 'auto', animation: 'fadeInUp 0.5s ease' }}>
        <Paper
          sx={{
            p: 3,
            borderRadius: '12px',
            bgcolor: 'white',
            border: '1px solid #e5e7ea',
            boxShadow: '0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -1px rgba(0,0,0,0.06)',
          }}
        >
          {children}
        </Paper>
      </Box>
    </Box>
  );
}
