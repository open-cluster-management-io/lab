import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  Tab,
  Paper,
} from '@mui/material';
import type { ManagedResource } from '../api/resourceService';
import StatusFeedbackDisplay from './StatusFeedbackDisplay';

interface Props {
  resource: ManagedResource;
  compact?: boolean;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel({ children, value, index }: TabPanelProps) {
  return (
    <div role="tabpanel" hidden={value !== index}>
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

const statusColor = (status: string): 'success' | 'warning' | 'error' | 'default' => {
  if (status === 'Applied' || status === 'Available') return 'success';
  if (status === 'Pending') return 'warning';
  if (status === 'Failed') return 'error';
  return 'default';
};

const formatDate = (dateString?: string) => {
  if (!dateString) return '-';
  return new Date(dateString).toLocaleString('en-US');
};

function OverviewFields({ resource }: { resource: ManagedResource }) {
  const navigate = useNavigate();
  return (
    <Grid container spacing={2} sx={{ width: '100%' }}>
      <Grid size={{ xs: 6 }}>
        <Typography variant="body2" color="text.secondary">Kind</Typography>
        <Typography variant="body1">{resource.kind}</Typography>
      </Grid>
      <Grid size={{ xs: 6 }}>
        <Typography variant="body2" color="text.secondary">API Version</Typography>
        <Typography variant="body1">{resource.apiVersion}</Typography>
      </Grid>
      <Grid size={{ xs: 6 }}>
        <Typography variant="body2" color="text.secondary">Name</Typography>
        <Typography variant="body1">{resource.name}</Typography>
      </Grid>
      <Grid size={{ xs: 6 }}>
        <Typography variant="body2" color="text.secondary">Namespace</Typography>
        <Typography variant="body1">{resource.namespace || '-'}</Typography>
      </Grid>
      <Grid size={{ xs: 6 }}>
        <Typography variant="body2" color="text.secondary">Cluster</Typography>
        <Typography
          variant="body1"
          sx={{ cursor: 'pointer', color: 'primary.main', '&:hover': { textDecoration: 'underline' } }}
          onClick={() => navigate(`/clusters/${resource.cluster}`)}
        >
          {resource.cluster}
        </Typography>
      </Grid>
      <Grid size={{ xs: 6 }}>
        <Typography variant="body2" color="text.secondary">Status</Typography>
        <Chip label={resource.status} size="small" color={statusColor(resource.status)} />
      </Grid>
      <Grid size={{ xs: 12 }}>
        <Typography variant="body2" color="text.secondary">ManifestWork</Typography>
        <Typography variant="body1">{resource.manifestWorkName}</Typography>
      </Grid>

      {resource.conditions && resource.conditions.length > 0 && (
        <Grid size={{ xs: 12 }}>
          <Typography variant="subtitle2" sx={{ mt: 1, mb: 1, fontWeight: 'medium' }}>Conditions</Typography>
          <TableContainer>
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
                {resource.conditions.map((cond, i) => (
                  <TableRow key={i}>
                    <TableCell>{cond.type}</TableCell>
                    <TableCell>
                      <Chip
                        label={cond.status}
                        size="small"
                        color={cond.status === 'True' ? 'success' : 'default'}
                      />
                    </TableCell>
                    <TableCell>{cond.reason || '-'}</TableCell>
                    <TableCell>{cond.message || '-'}</TableCell>
                    <TableCell>{formatDate(cond.lastTransitionTime)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Grid>
      )}

      {resource.statusFeedback?.values && resource.statusFeedback.values.length > 0 && (
        <Grid size={{ xs: 12 }}>
          <Typography variant="subtitle2" sx={{ mt: 1, mb: 1, fontWeight: 'medium' }}>Status Feedback</Typography>
          <StatusFeedbackDisplay feedback={resource.statusFeedback} variant="table" />
        </Grid>
      )}
    </Grid>
  );
}

export default function ResourceDetailContent({ resource, compact = false }: Props) {
  const [tabValue, setTabValue] = useState(0);

  if (compact) {
    return (
      <Box>
        <OverviewFields resource={resource} />
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)} aria-label="resource detail tabs">
          <Tab label="Overview" />
          <Tab label="Spec" />
        </Tabs>
      </Box>

      <TabPanel value={tabValue} index={0}>
        <OverviewFields resource={resource} />
      </TabPanel>

      <TabPanel value={tabValue} index={1}>
        {resource.rawResource ? (
          <Paper
            variant="outlined"
            sx={{
              p: 2,
              backgroundColor: 'grey.50',
              fontFamily: 'monospace',
              fontSize: '0.8rem',
              overflowX: 'auto',
              whiteSpace: 'pre',
              maxHeight: '60vh',
              overflow: 'auto',
            }}
            component="pre"
          >
            {JSON.stringify(resource.rawResource, null, 2)}
          </Paper>
        ) : (
          <Typography variant="body2" color="text.secondary">
            Spec not available. View full details to load the spec.
          </Typography>
        )}
      </TabPanel>
    </Box>
  );
}
