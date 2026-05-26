import { useState } from 'react';
import {
  Alert,
  Box,
  Typography,
  Paper,
  Chip,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import type { ManifestWork } from '../api/manifestWorkService';
import MWFlowChart from './MWFlowChart';
import StatusFeedbackDisplay from './StatusFeedbackDisplay';
import { deriveMWStatus, chipColor, getMWDegradedReasons, deriveResStatus } from '../utils/statusHelpers';

interface Props {
  mw: ManifestWork;
  compact?: boolean;
}

export default function ManifestWorkDetailContent({ mw, compact }: Props) {
  const [tab, setTab] = useState(0);
  const statusLabel = deriveMWStatus(mw);
  const statusColor = chipColor(statusLabel);
  const degradedReasons = getMWDegradedReasons(mw);

  const overviewContent = (
    <>
      {/* Overview */}
      <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>Overview</Typography>

        <Box sx={{ display: "flex", mb: 1 }}>
          <Typography variant="subtitle2" sx={{ width: 140 }}>Name:</Typography>
          <Typography variant="body2">{mw.name}</Typography>
        </Box>
        <Box sx={{ display: "flex", mb: 1 }}>
          <Typography variant="subtitle2" sx={{ width: 140 }}>Cluster:</Typography>
          <Typography variant="body2">{mw.namespace}</Typography>
        </Box>
        <Box sx={{ display: "flex", mb: 1 }}>
          <Typography variant="subtitle2" sx={{ width: 140 }}>Status:</Typography>
          <Chip label={statusLabel} color={statusColor} size="small" />
        </Box>
        <Box sx={{ display: "flex", mb: 1 }}>
          <Typography variant="subtitle2" sx={{ width: 140 }}>Resources:</Typography>
          <Typography variant="body2">{mw.resourceStatus?.manifests?.length ?? 0}</Typography>
        </Box>
        <Box sx={{ display: "flex", mb: 1 }}>
          <Typography variant="subtitle2" sx={{ width: 140 }}>Created:</Typography>
          <Typography variant="body2">
            {mw.creationTimestamp ? new Date(mw.creationTimestamp).toLocaleString() : '-'}
          </Typography>
        </Box>
      </Paper>

      {/* Degraded alert */}
      {degradedReasons.length > 0 && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          {degradedReasons.map((r, i) => (
            <Typography key={i} variant="body2">
              <strong>{r.resource}</strong>: {r.reason}
            </Typography>
          ))}
        </Alert>
      )}

      {/* Conditions */}
      {mw.conditions && mw.conditions.length > 0 && (
        <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
          <Typography variant="h6" sx={{ mb: 2 }}>Conditions</Typography>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Type</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Reason</TableCell>
                  <TableCell>Message</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {mw.conditions.map((condition, idx) => (
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
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      )}

      {/* Resource Status */}
      {mw.resourceStatus?.manifests && mw.resourceStatus.manifests.length > 0 && !compact && (
        <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
          <Typography variant="h6" sx={{ mb: 2 }}>Resources</Typography>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Kind</TableCell>
                  <TableCell>Name</TableCell>
                  <TableCell>Namespace</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Status Feedback</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {mw.resourceStatus.manifests.map((res, idx) => {
                  const resStatus = deriveResStatus(res.conditions ?? [], res.statusFeedback);
                  return (
                    <TableRow key={idx}>
                      <TableCell>
                        <Chip label={res.resourceMeta.kind ?? 'Unknown'} size="small" variant="outlined" />
                      </TableCell>
                      <TableCell>{res.resourceMeta.name || '-'}</TableCell>
                      <TableCell>{res.resourceMeta.namespace || '-'}</TableCell>
                      <TableCell>
                        <Chip
                          label={resStatus}
                          size="small"
                          color={chipColor(resStatus)}
                        />
                      </TableCell>
                      <TableCell>
                        <StatusFeedbackDisplay feedback={res.statusFeedback} variant="inline" maxItems={3} />
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      )}
    </>
  );

  if (compact) {
    return overviewContent;
  }

  return (
    <Box>
      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 3 }}>
        <Tab label="Overview" />
        <Tab label="Graph" />
      </Tabs>

      {tab === 0 && overviewContent}
      {tab === 1 && <MWFlowChart mw={mw} />}
    </Box>
  );
}
