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
import type { ManifestWork } from '../api/manifestWorkService';
import { useState } from 'react';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';

interface ManifestWorkRowProps {
  manifestWork: ManifestWork;
}

// Format date function
const formatDate = (dateString?: string) => {
  if (!dateString) return 'Unknown';
  return new Date(dateString).toLocaleString('en-US');
};

// Get status of a ManifestWork based on conditions
const getManifestWorkStatus = (manifestWork: ManifestWork): { status: string; color: 'success' | 'error' | 'warning' | 'default' } => {
  if (!manifestWork.conditions || manifestWork.conditions.length === 0) {
    return { status: 'Unknown', color: 'default' };
  }

  const appliedCondition = manifestWork.conditions.find(c => c.type === 'Applied');
  if (appliedCondition) {
    if (appliedCondition.status === 'True') {
      return { status: 'Applied', color: 'success' };
    }
  }

  const availableCondition = manifestWork.conditions.find(c => c.type === 'Available');
  if (availableCondition) {
    if (availableCondition.status === 'True') {
      return { status: 'Available', color: 'success' };
    } else {
      return { status: 'Unavailable', color: 'error' };
    }
  }

  return { status: 'Unknown', color: 'default' };
};

// Component for a single ManifestWork row with expandable details
function ManifestWorkRow({ manifestWork }: ManifestWorkRowProps) {
  const [open, setOpen] = useState(false);
  const manifestWorkStatus = getManifestWorkStatus(manifestWork);

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
            <Typography variant="body1">{manifestWork.name}</Typography>
          </Box>
        </TableCell>
        <TableCell>
          <Chip
            label={manifestWorkStatus.status}
            color={manifestWorkStatus.color}
            size="small"
          />
        </TableCell>
        <TableCell>{formatDate(manifestWork.creationTimestamp)}</TableCell>
      </TableRow>
      <TableRow>
        <TableCell style={{ paddingBottom: 0, paddingTop: 0 }} colSpan={4}>
          <Collapse in={open} timeout="auto" unmountOnExit>
            <Box sx={{ margin: 2 }}>
              <Typography variant="subtitle2" gutterBottom component="div" sx={{ fontWeight: 'bold' }}>
                Manifests
              </Typography>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Kind</TableCell>
                      <TableCell>Name</TableCell>
                      <TableCell>Namespace</TableCell>
                      <TableCell>Status</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {manifestWork.resourceStatus?.manifests?.map((manifest, idx) => {
                      const appliedCondition = manifest.conditions.find(c => c.type === 'Applied');
                      const status = appliedCondition?.status === 'True' ? 'Applied' : 'Not Applied';
                      const statusColor = appliedCondition?.status === 'True' ? 'success' : 'error';

                      return (
                        <TableRow key={idx}>
                          <TableCell>{manifest.resourceMeta.kind || '-'}</TableCell>
                          <TableCell>{manifest.resourceMeta.name || '-'}</TableCell>
                          <TableCell>{manifest.resourceMeta.namespace || '-'}</TableCell>
                          <TableCell>
                            <Chip
                              label={status}
                              color={statusColor}
                              size="small"
                            />
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </TableContainer>

              <Typography variant="subtitle2" gutterBottom component="div" sx={{ fontWeight: 'bold', mt: 2 }}>
                Conditions
              </Typography>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Type</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell>Reason</TableCell>
                      <TableCell>Message</TableCell>
                      <TableCell>Last Updated</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {manifestWork.conditions?.map((condition, idx) => (
                      <TableRow key={idx}>
                        <TableCell>{condition.type}</TableCell>
                        <TableCell>
                          <Chip
                            label={condition.status}
                            color={condition.status === 'True' ? 'success' : 'default'}
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
            </Box>
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  );
}

interface ClusterManifestWorksListProps {
  manifestWorks: ManifestWork[];
  loading: boolean;
  error: string | null;
}

export default function ClusterManifestWorksList({ manifestWorks, loading, error }: ClusterManifestWorksListProps) {
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
        Error loading manifest works: {error}
      </Alert>
    );
  }

  if (manifestWorks.length === 0) {
    return (
      <Alert severity="info" sx={{ mb: 3 }}>
        No manifest works found for this cluster.
      </Alert>
    );
  }

  return (
    <TableContainer component={Paper} variant="outlined">
      <Table aria-label="cluster manifest works table">
        <TableHead>
          <TableRow>
            <TableCell width="60px" />
            <TableCell>Name</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Created</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {manifestWorks.map((manifestWork) => (
            <ManifestWorkRow key={manifestWork.id} manifestWork={manifestWork} />
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}