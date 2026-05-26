import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Typography,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Tabs,
  Tab,
} from '@mui/material';
import {
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  HourglassEmpty as ProgressingIcon,
  ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import {
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material';
import type { ManifestWorkReplicaSet } from '../api/manifestWorkReplicaSetService';
import { fetchManifestWorksByReplicaSet } from '../api/manifestWorkReplicaSetService';
import type { ManifestWork } from '../api/manifestWorkService';
import ClusterManifestWorksList from './ClusterManifestWorksList';
import MWRSFlowChart from './MWRSFlowChart';
import StatusFeedbackDisplay from './StatusFeedbackDisplay';
import { deriveMWRSStatus, chipColor, getMWDegradedReasons } from '../utils/statusHelpers';

interface Props {
  mwrs: ManifestWorkReplicaSet;
  compact?: boolean;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel({ children, value, index, ...other }: TabPanelProps) {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`mwrs-tabpanel-${index}`}
      aria-labelledby={`mwrs-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

function a11yProps(index: number) {
  return {
    id: `mwrs-tab-${index}`,
    'aria-controls': `mwrs-tabpanel-${index}`,
  };
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'Unknown';
  return new Date(dateString).toLocaleString('en-US');
};

export default function ManifestWorkReplicaSetDetailContent({ mwrs, compact = false }: Props) {
  const [tabValue, setTabValue] = useState(0);
  const navigate = useNavigate();
  const [manifestWorks, setManifestWorks] = useState<ManifestWork[]>([]);
  const [mwLoading, setMwLoading] = useState(false);
  const [mwError, setMwError] = useState<string | null>(null);

  useEffect(() => {
    setMwLoading(true);
    setMwError(null);
    fetchManifestWorksByReplicaSet(mwrs.namespace, mwrs.name)
      .then(setManifestWorks)
      .catch((err) => setMwError(err.message || 'Failed to fetch ManifestWorks'))
      .finally(() => setMwLoading(false));
  }, [mwrs.namespace, mwrs.name]);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const status = deriveMWRSStatus(mwrs, manifestWorks.length > 0 ? manifestWorks : undefined);

  const getStatusIcon = (s: string) => {
    const color = chipColor(s);
    if (color === 'success') return <CheckCircleIcon sx={{ color: "success.main", fontSize: 18, mr: 0.5 }} />;
    if (color === 'warning') return <ProgressingIcon sx={{ color: "warning.main", fontSize: 18, mr: 0.5 }} />;
    return <ErrorIcon sx={{ color: "error.main", fontSize: 18, mr: 0.5 }} />;
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {!compact && (
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs
            value={tabValue}
            onChange={handleTabChange}
            aria-label="mwrs detail tabs"
            variant="scrollable"
            scrollButtons="auto"
          >
            <Tab label="Overview" {...a11yProps(0)} />
            <Tab label="ManifestWorks" {...a11yProps(1)} />
            <Tab label="Graph" {...a11yProps(2)} />
          </Tabs>
        </Box>
      )}

      <TabPanel value={tabValue} index={0}>
        {/* Basic info */}
        <Box sx={{ mb: 3 }}>
          <Grid container spacing={2} sx={{ width: '100%' }}>
            <Grid size={{ xs: 6 }}>
              <Typography variant="body2" color="text.secondary">Name</Typography>
              <Typography variant="body1">{mwrs.name}</Typography>
            </Grid>
            <Grid size={{ xs: 6 }}>
              <Typography variant="body2" color="text.secondary">Namespace</Typography>
              <Typography variant="body1">{mwrs.namespace}</Typography>
            </Grid>
            <Grid size={{ xs: 6 }}>
              <Typography variant="body2" color="text.secondary">Status</Typography>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                {getStatusIcon(status)}
                <Typography variant="body1">{status}</Typography>
              </Box>
            </Grid>
            <Grid size={{ xs: 6 }}>
              <Typography variant="body2" color="text.secondary">Created</Typography>
              <Typography variant="body1">{formatDate(mwrs.creationTimestamp)}</Typography>
            </Grid>
            <Grid size={{ xs: 6 }}>
              <Typography variant="body2" color="text.secondary">Manifest Count</Typography>
              <Typography variant="body1">{mwrs.manifestCount}</Typography>
            </Grid>
          </Grid>
        </Box>

        {/* Degraded alert */}
        {status === 'Degraded' && manifestWorks.length > 0 && (() => {
          const reasons = manifestWorks.flatMap(mw => {
            const items = getMWDegradedReasons(mw);
            return items.map(r => ({ cluster: mw.namespace, ...r }));
          });
          if (reasons.length === 0) return null;
          return (
            <Alert severity="warning" sx={{ mb: 3 }}>
              <Typography variant="body2" fontWeight="bold" sx={{ mb: 0.5 }}>
                Workloads are degraded on {new Set(reasons.map(r => r.cluster)).size} cluster(s)
              </Typography>
              {reasons.map((r, i) => (
                <Typography key={i} variant="body2">
                  <strong>{r.cluster}</strong> &mdash; {r.resource}: {r.reason}
                </Typography>
              ))}
            </Alert>
          );
        })()}

        {/* Summary */}
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
            Summary
          </Typography>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Total</TableCell>
                  <TableCell>Available</TableCell>
                  <TableCell>Applied</TableCell>
                  <TableCell>Progressing</TableCell>
                  <TableCell>Degraded</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                <TableRow>
                  <TableCell>{mwrs.summary.total}</TableCell>
                  <TableCell>
                    <Chip label={mwrs.summary.available} size="small" color="success" variant="outlined" />
                  </TableCell>
                  <TableCell>
                    <Chip label={mwrs.summary.applied} size="small" color="info" variant="outlined" />
                  </TableCell>
                  <TableCell>
                    {mwrs.summary.progressing > 0 ? (
                      <Chip label={mwrs.summary.progressing} size="small" color="warning" variant="outlined" />
                    ) : (
                      mwrs.summary.progressing
                    )}
                  </TableCell>
                  <TableCell>
                    {mwrs.summary.degraded > 0 ? (
                      <Chip label={mwrs.summary.degraded} size="small" color="error" variant="outlined" />
                    ) : (
                      mwrs.summary.degraded
                    )}
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </TableContainer>
        </Box>

        {/* Placement Refs */}
        {mwrs.placementRefs && mwrs.placementRefs.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
              Placement References
            </Typography>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Placement Name</TableCell>
                    <TableCell>Rollout Strategy</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {mwrs.placementRefs.map((ref) => (
                    <TableRow
                      key={ref.name}
                      hover
                      sx={{ cursor: 'pointer' }}
                      onClick={() => navigate(`/placements?name=${ref.name}&namespace=${mwrs.namespace}`)}
                    >
                      <TableCell>{ref.name}</TableCell>
                      <TableCell>
                        <Chip label={ref.rolloutStrategyType || 'All'} size="small" variant="outlined" />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        )}

        {/* Placement Summaries */}
        {mwrs.placementsSummary && mwrs.placementsSummary.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
              Placement Summaries
            </Typography>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Placement</TableCell>
                    <TableCell>Decision Groups</TableCell>
                    <TableCell align="center">Total</TableCell>
                    <TableCell align="center">Available</TableCell>
                    <TableCell align="center">Applied</TableCell>
                    <TableCell align="center">Degraded</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {mwrs.placementsSummary.map((ps) => (
                    <TableRow key={ps.name}>
                      <TableCell>{ps.name}</TableCell>
                      <TableCell>{ps.availableDecisionGroups}</TableCell>
                      <TableCell align="center">{ps.summary.total}</TableCell>
                      <TableCell align="center">{ps.summary.available}</TableCell>
                      <TableCell align="center">{ps.summary.applied}</TableCell>
                      <TableCell align="center">
                        {ps.summary.degraded > 0 ? (
                          <Chip label={ps.summary.degraded} size="small" color="error" variant="outlined" />
                        ) : (
                          ps.summary.degraded
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        )}

        {/* Conditions */}
        {mwrs.conditions && mwrs.conditions.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
              Conditions
            </Typography>
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
                  {mwrs.conditions.map((condition, index) => (
                    <TableRow key={index}>
                      <TableCell>{condition.type}</TableCell>
                      <TableCell>
                        <Chip
                          label={condition.status}
                          size="small"
                          color={condition.status === 'True' ? 'success' : 'default'}
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
        )}

        {/* Status Feedback (from child ManifestWorks) */}
        {!mwLoading && manifestWorks.some(mw =>
          mw.resourceStatus?.manifests?.some(m => m.statusFeedback?.values?.length)
        ) && (
          <Box>
            <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
              Status Feedback
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              Values reported back from spoke clusters via OCM StatusFeedback.
            </Typography>
            {manifestWorks
              .filter(mw => mw.resourceStatus?.manifests?.some(m => m.statusFeedback?.values?.length))
              .map(mw => (
                <Accordion key={mw.id} variant="outlined" disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
                  <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Typography variant="body2" fontWeight="bold">{mw.namespace}</Typography>
                      <Typography variant="caption" color="text.secondary">
                        ({mw.resourceStatus!.manifests!.filter(m => m.statusFeedback?.values?.length).length} resources with feedback)
                      </Typography>
                    </Box>
                  </AccordionSummary>
                  <AccordionDetails>
                    {mw.resourceStatus!.manifests!
                      .filter(m => m.statusFeedback?.values?.length)
                      .map((manifest, idx) => (
                        <Box key={idx} sx={{ mb: idx < mw.resourceStatus!.manifests!.filter(m => m.statusFeedback?.values?.length).length - 1 ? 2 : 0 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                            <Chip label={manifest.resourceMeta.kind || 'Resource'} size="small" variant="outlined" />
                            <Typography variant="body2">{manifest.resourceMeta.name || '-'}</Typography>
                          </Box>
                          <StatusFeedbackDisplay feedback={manifest.statusFeedback} variant="table" />
                        </Box>
                      ))}
                  </AccordionDetails>
                </Accordion>
              ))}
          </Box>
        )}
      </TabPanel>

      <TabPanel value={tabValue} index={1}>
        {/* ManifestWorks tab */}
        <Box>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            ManifestWorks created by this ManifestWorkReplicaSet across managed clusters.
          </Typography>

          {(() => {
            const byCluster = manifestWorks.reduce<Record<string, ManifestWork[]>>((acc, mw) => {
              const cluster = mw.namespace;
              if (!acc[cluster]) acc[cluster] = [];
              acc[cluster].push(mw);
              return acc;
            }, {});
            const clusterNames = Object.keys(byCluster).sort();

            if (mwLoading || mwError || manifestWorks.length === 0) {
              return (
                <ClusterManifestWorksList
                  manifestWorks={manifestWorks}
                  loading={mwLoading}
                  error={mwError}
                />
              );
            }

            return clusterNames.map((cluster) => (
              <Box key={cluster} sx={{ mb: 3 }}>
                <Typography
                  variant="subtitle1"
                  sx={{ mb: 1, fontWeight: 'medium', cursor: 'pointer', '&:hover': { textDecoration: 'underline' } }}
                  onClick={() => navigate(`/clusters/${cluster}`)}
                >
                  Cluster: {cluster}
                </Typography>
                <ClusterManifestWorksList
                  manifestWorks={byCluster[cluster]}
                  loading={false}
                  error={null}
                />
              </Box>
            ));
          })()}
        </Box>
      </TabPanel>

      <TabPanel value={tabValue} index={2}>
        <MWRSFlowChart mwrs={mwrs} />
      </TabPanel>
    </Box>
  );
}
