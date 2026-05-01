import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Collapse,
  IconButton,
  Chip,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  KeyboardArrowDown as KeyboardArrowDownIcon,
  KeyboardArrowUp as KeyboardArrowUpIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  HelpOutline as HelpOutlineIcon,
  Extension as ExtensionIcon,
} from '@mui/icons-material';
import type { ManagedClusterAddon } from '../api/addonService';
import { useState } from 'react';

interface AddonRowProps {
  addon: ManagedClusterAddon;
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'Unknown';
  return new Date(dateString).toLocaleString('en-US');
};

const getAddonStatus = (addon: ManagedClusterAddon): { status: string; color: 'success' | 'error' | 'warning' | 'default'; icon: React.ReactNode } => {
  if (!addon.conditions || addon.conditions.length === 0) {
    return { status: 'Unknown', color: 'default', icon: <HelpOutlineIcon sx={{ color: 'text.secondary', fontSize: 18, mr: 0.5 }} /> };
  }

  const availableCondition = addon.conditions.find(c => c.type === 'Available');
  if (availableCondition) {
    if (availableCondition.status === 'True') {
      return { status: 'Available', color: 'success', icon: <CheckCircleIcon sx={{ color: 'success.main', fontSize: 18, mr: 0.5 }} /> };
    }
    return { status: 'Unavailable', color: 'error', icon: <ErrorIcon sx={{ color: 'error.main', fontSize: 18, mr: 0.5 }} /> };
  }

  const progressingCondition = addon.conditions.find(c => c.type === 'Progressing');
  if (progressingCondition && progressingCondition.status === 'True') {
    return { status: 'Progressing', color: 'warning', icon: <CircularProgress size={16} sx={{ mr: 0.5, color: 'warning.main' }} /> };
  }

  return { status: 'Unknown', color: 'default', icon: <HelpOutlineIcon sx={{ color: 'text.secondary', fontSize: 18, mr: 0.5 }} /> };
};

function AddonRow({ addon }: AddonRowProps) {
  const [open, setOpen] = useState(false);
  const addonStatus = getAddonStatus(addon);

  return (
    <>
      <TableRow
        hover
        sx={{
          cursor: 'pointer',
          '& > *': { borderBottom: 'unset' },
          '& > td': { padding: '12px 16px' },
        }}
        onClick={() => setOpen(!open)}
      >
        <TableCell width={48}>
          <IconButton size="small" sx={{ transition: 'transform 0.2s ease' }}>
            {open ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
          </IconButton>
        </TableCell>
        <TableCell>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ExtensionIcon sx={{ color: 'primary.main', fontSize: 20 }} />
            <Typography variant="body2" sx={{ fontWeight: 600 }}>{addon.name}</Typography>
          </Box>
        </TableCell>
        <TableCell>
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            {addonStatus.icon}
            <Chip label={addonStatus.status} color={addonStatus.color} size="small" />
          </Box>
        </TableCell>
        <TableCell>
          <Chip label={addon.installNamespace} size="small" variant="outlined" />
        </TableCell>
        <TableCell>
          {addon.creationTimestamp
            ? new Date(addon.creationTimestamp).toLocaleDateString()
            : '-'}
        </TableCell>
      </TableRow>
      <TableRow>
        <TableCell sx={{ py: 0 }} colSpan={6}>
          <Collapse in={open} timeout="auto" unmountOnExit>
            <Box sx={{ py: 2, px: 1 }}>

              {addon.registrations && addon.registrations.length > 0 && (
                <Paper sx={{ p: 2, mb: 2, borderRadius: 2 }}>
                  <Typography variant="subtitle2" sx={{ mb: 1.5 }}>
                    Registrations
                  </Typography>
                  {addon.registrations.map((registration, idx) => (
                    <Box
                      key={idx}
                      sx={{
                        p: 1.5,
                        mb: idx < addon.registrations!.length - 1 ? 1.5 : 0,
                        borderRadius: 1,
                        bgcolor: 'grey.50',
                        border: '1px solid',
                        borderColor: 'divider',
                      }}
                    >
                      <Box sx={{ display: 'flex', mb: 0.5 }}>
                        <Typography variant="body2" sx={{ width: 80, fontWeight: 600 }}>Signer:</Typography>
                        <Typography variant="body2">{registration.signerName}</Typography>
                      </Box>
                      <Box sx={{ display: 'flex', mb: 0.5 }}>
                        <Typography variant="body2" sx={{ width: 80, fontWeight: 600 }}>User:</Typography>
                        <Typography variant="body2">{registration.subject?.user || '-'}</Typography>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'flex-start' }}>
                        <Typography variant="body2" sx={{ width: 80, fontWeight: 600 }}>Groups:</Typography>
                        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                          {(registration.subject?.groups || []).length > 0 ? (
                            (registration.subject?.groups || []).map((group, gIdx) => (
                              <Chip key={gIdx} label={group} size="small" variant="outlined" />
                            ))
                          ) : (
                            <Typography variant="body2" color="text.secondary">-</Typography>
                          )}
                        </Box>
                      </Box>
                    </Box>
                  ))}
                </Paper>
              )}

              {addon.supportedConfigs && addon.supportedConfigs.length > 0 && (
                <Paper sx={{ p: 2, mb: 2, borderRadius: 2 }}>
                  <Typography variant="subtitle2" sx={{ mb: 1.5 }}>
                    Supported Configurations
                  </Typography>
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>Group</TableCell>
                          <TableCell>Resource</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {addon.supportedConfigs.map((config, idx) => (
                          <TableRow key={idx}>
                            <TableCell sx={{ py: 1.5 }}>{config.group}</TableCell>
                            <TableCell sx={{ py: 1.5 }}>{config.resource}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </Paper>
              )}

              <Paper sx={{ p: 2, borderRadius: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: 1.5 }}>
                  Conditions
                </Typography>
                {addon.conditions && addon.conditions.length > 0 ? (
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>Type</TableCell>
                          <TableCell>Status</TableCell>
                          <TableCell>Reason</TableCell>
                          <TableCell>Message</TableCell>
                          <TableCell>Last Transition</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {addon.conditions.map((condition, index) => (
                          <TableRow key={index}>
                            <TableCell sx={{ py: 1.5 }}>{condition.type}</TableCell>
                            <TableCell sx={{ py: 1.5 }}>
                              <Chip
                                label={condition.status}
                                color={condition.status === 'True' ? 'success' : condition.status === 'False' ? 'default' : 'warning'}
                                size="small"
                              />
                            </TableCell>
                            <TableCell sx={{ py: 1.5 }}>{condition.reason || '-'}</TableCell>
                            <TableCell sx={{ py: 1.5, maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                              {condition.message || '-'}
                            </TableCell>
                            <TableCell sx={{ py: 1.5 }}>{formatDate(condition.lastTransitionTime)}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                ) : (
                  <Typography variant="body2" color="text.secondary">No conditions available</Typography>
                )}
              </Paper>

            </Box>
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  );
}

interface ClusterAddonsListProps {
  addons: ManagedClusterAddon[];
  loading: boolean;
  error: string | null;
}

export default function ClusterAddonsList({ addons, loading, error }: ClusterAddonsListProps) {
  const safeAddons = Array.isArray(addons) ? addons : [];

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mb: 3, borderRadius: '12px' }}>
        Error loading add-ons: {error}
      </Alert>
    );
  }

  if (safeAddons.length === 0) {
    return (
      <Alert severity="info" sx={{ mb: 3, borderRadius: '12px' }}>
        No add-ons found for this cluster.
      </Alert>
    );
  }

  return (
    <TableContainer component={Paper} sx={{ borderRadius: '12px', overflow: 'hidden' }}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell width={48} />
            <TableCell>Name</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Install Namespace</TableCell>
            <TableCell>Created</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {safeAddons.map((addon, index) => (
            <AddonRow key={addon.id || `${addon.namespace || 'cluster'}-${addon.name || index}`} addon={addon} />
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
