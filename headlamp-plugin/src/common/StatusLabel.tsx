import { type ReactNode } from 'react';
import { Box, Typography } from '@mui/material';

type StatusType = 'success' | 'error' | 'warning' | 'unknown';

const statusColors: Record<StatusType, { bg: string; fg: string }> = {
  success: { bg: '#e6f4ea', fg: '#1b7a3d' },
  error: { bg: '#fce8e6', fg: '#c5221f' },
  warning: { bg: '#fef7e0', fg: '#b06000' },
  unknown: { bg: '#e8eaed', fg: '#5f6368' },
};

interface StatusLabelProps {
  status: StatusType;
  children: ReactNode;
}

export default function StatusLabel({ status, children }: StatusLabelProps) {
  const colors = statusColors[status];
  return (
    <Box
      component="span"
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '4px',
        padding: '2px 8px',
        borderRadius: '4px',
        backgroundColor: colors.bg,
      }}
    >
      <Typography component="span" sx={{ fontSize: '0.8rem', fontWeight: 500, color: colors.fg }}>
        {children}
      </Typography>
    </Box>
  );
}

export function conditionToStatus(conditionStatus: string | undefined): StatusType {
  switch (conditionStatus) {
    case 'True':
      return 'success';
    case 'False':
      return 'error';
    default:
      return 'unknown';
  }
}
