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
  Card,
  CardHeader,
  CardContent
} from '@mui/material';
import type { ManagedClusterAddon } from '../api/addonService';
import { useState } from 'react';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';

interface AddonRowProps {
  addon: ManagedClusterAddon;
}

// Format date
const formatDate = (dateString?: string) => {
  if (!dateString) return 'Unknown';
  return new Date(dateString).toLocaleString('en-US');
};

// Get status of an addon based on conditions
const getAddonStatus = (addon: ManagedClusterAddon): { status: string; color: 'success' | 'error' | 'warning' | 'default' } => {
  if (!addon.conditions || addon.conditions.length === 0) {
    return { status: 'Unknown', color: 'default' };
  }

  const availableCondition = addon.conditions.find(c => c.type === 'Available');
  if (availableCondition) {
    if (availableCondition.status === 'True') {
      return { status: 'Available', color: 'success' };
    } else {
      return { status: 'Unavailable', color: 'error' };
    }
  }

  const progressingCondition = addon.conditions.find(c => c.type === 'Progressing');
  if (progressingCondition && progressingCondition.status === 'True') {
    return { status: 'Progressing', color: 'warning' };
  }

  return { status: 'Unknown', color: 'default' };
};

// Component for a single addon row with expandable details
function AddonRow({ addon }: AddonRowProps) {
  const [open, setOpen] = useState(false);
  const addonStatus = getAddonStatus(addon);

  return (
    <>
      <TableRow sx={{ '& > *': { borderBottom: 'unset' } }}>
        <TableCell>
          <IconButton
            aria-label="expand row"
            size="small"
            onClick={() => setOpen(!open)}
          >
            {open ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
          </IconButton>
        </TableCell>
        <TableCell component="th" scope="row">
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <Typography variant="body1">{addon.name}</Typography>
          </Box>
        </TableCell>
        <TableCell>
          <Chip
            label={addonStatus.status}
            color={addonStatus.color}
            size="small"
          />
        </TableCell>
        <TableCell>{addon.installNamespace}</TableCell>
        <TableCell>{formatDate(addon.creationTimestamp)}</TableCell>
      </TableRow>
      <TableRow>
        <TableCell style={{ paddingBottom: 0, paddingTop: 0 }} colSpan={6}>
          <Collapse in={open} timeout="auto" unmountOnExit>
            <Box sx={{ margin: 2 }}>
              {addon.registrations && addon.registrations.length > 0 && (
                <>
                  <Typography variant="subtitle2" gutterBottom component="div" sx={{ fontWeight: 'bold', mt: 0 }}>
                    Registrations
                  </Typography>
                  {addon.registrations.map((registration, idx) => (
                    <Card key={idx} variant="outlined" sx={{ mb: 2 }}>
                      <CardHeader
                        title={<Typography variant="subtitle2">Signer: {registration.signerName}</Typography>}
                        sx={{ pb: 0 }}
                      />
                      <CardContent>
                        <Typography variant="body2" color="text.secondary" gutterBottom>
                          User: {registration.subject.user}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" gutterBottom>
                          Groups:
                        </Typography>
                        <Box sx={{ ml: 2 }}>
                          {registration.subject.groups.map((group, gIdx) => (
                            <Typography key={gIdx} variant="body2" color="text.secondary">
                              • {group}
                            </Typography>
                          ))}
                        </Box>
                      </CardContent>
                    </Card>
                  ))}
                </>
              )}

              {addon.supportedConfigs && addon.supportedConfigs.length > 0 && (
                <>
                  <Typography variant="subtitle2" gutterBottom component="div" sx={{ fontWeight: 'bold', mt: 2 }}>
                    Supported Configurations
                  </Typography>
                  <TableContainer component={Paper} variant="outlined">
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
                            <TableCell>{config.group}</TableCell>
                            <TableCell>{config.resource}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </>
              )}

              <Typography variant="subtitle2" gutterBottom component="div" sx={{ fontWeight: 'bold', mt: 2 }}>
                Conditions
              </Typography>
              {addon.conditions && addon.conditions.length > 0 ? (
                <TableContainer component={Paper} variant="outlined" sx={{ mb: 2 }}>
                  <Table size="small" aria-label="addon conditions">
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
                          <TableCell component="th" scope="row">{condition.type}</TableCell>
                          <TableCell>
                            <Chip
                              label={condition.status}
                              color={condition.status === 'True' ? 'success' : condition.status === 'False' ? 'default' : 'warning'}
                              size="small"
                            />
                          </TableCell>
                          <TableCell>{condition.reason || '-'}</TableCell>
                          <TableCell>{condition.message || '-'}</TableCell>
                          <TableCell>{formatDate(condition.lastTransitionTime)}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              ) : (
                <Typography variant="body2" color="text.secondary">No conditions available</Typography>
              )}
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
  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mb: 3 }}>
        Error loading add-ons: {error}
      </Alert>
    );
  }

  if (addons.length === 0) {
    return (
      <Alert severity="info" sx={{ mb: 3 }}>
        No add-ons found for this cluster.
      </Alert>
    );
  }

  return (
    <TableContainer component={Paper} variant="outlined">
      <Table aria-label="cluster addons table">
        <TableHead>
          <TableRow>
            <TableCell width="60px" />
            <TableCell>Name</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Install Namespace</TableCell>
            <TableCell>Created</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {addons.map((addon) => (
            <AddonRow key={addon.id} addon={addon} />
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}