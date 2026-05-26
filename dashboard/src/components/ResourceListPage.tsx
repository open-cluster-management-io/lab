import { useState, useEffect, useMemo } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Paper,
  Chip,
  Button,
  IconButton,
  TextField,
  InputAdornment,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  CircularProgress,
} from '@mui/material';
import type { SelectChangeEvent } from '@mui/material';
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Launch as LaunchIcon,
} from '@mui/icons-material';
import { fetchManagedResources, fetchManagedResource, type ManagedResource } from '../api/resourceService';
import ResourceDetailContent from './ResourceDetailContent';
import DrawerLayout from './layout/DrawerLayout';

const statusColor = (status: string): 'success' | 'warning' | 'error' | 'default' => {
  if (status === 'Applied' || status === 'Available') return 'success';
  if (status === 'Pending') return 'warning';
  if (status === 'Failed') return 'error';
  return 'default';
};

export default function ResourceListPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [searchTerm, setSearchTerm] = useState('');
  const [filterKind, setFilterKind] = useState('all');
  const [filterCluster, setFilterCluster] = useState('all');
  const [filterNamespace, setFilterNamespace] = useState('all');

  const [resources, setResources] = useState<ManagedResource[]>([]);
  const [availableKinds, setAvailableKinds] = useState<string[]>([]);
  const [clusters, setClusters] = useState<string[]>([]);
  const [namespaces, setNamespaces] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  const [selectedResource, setSelectedResource] = useState<ManagedResource | null>(null);
  const selectedId = searchParams.get('selected');

  const loadResources = async () => {
    setLoading(true);
    try {
      const data = await fetchManagedResources();
      setResources(data.resources);
      setAvailableKinds(data.availableKinds);
      setClusters(data.clusters);
      setNamespaces(data.namespaces);
    } catch (error) {
      console.error('Error fetching resources:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadResources();
  }, []);

  const filteredResources = useMemo(() => {
    return resources.filter(r => {
      const matchesSearch =
        r.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        r.kind.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesKind = filterKind === 'all' || r.kind === filterKind;
      const matchesCluster = filterCluster === 'all' || r.cluster === filterCluster;
      const matchesNs = filterNamespace === 'all' || r.namespace === filterNamespace;
      return matchesSearch && matchesKind && matchesCluster && matchesNs;
    });
  }, [resources, searchTerm, filterKind, filterCluster, filterNamespace]);

  const handleSelect = async (resource: ManagedResource) => {
    setSearchParams({ selected: resource.id });
    const full = await fetchManagedResource(resource.cluster, resource.manifestWorkName, resource.ordinal);
    setSelectedResource(full || resource);
  };

  const handleCloseDetail = () => {
    setSearchParams({});
    setSelectedResource(null);
  };

  const handleViewFullDetails = () => {
    if (selectedResource) {
      navigate(`/resources/${selectedResource.cluster}/${selectedResource.manifestWorkName}/${selectedResource.ordinal}`);
    }
  };

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)' }}>
      {/* List */}
      <Box
        sx={{
          flex: selectedId ? '0 0 60%' : '1 1 auto',
          p: 3,
          transition: 'flex 0.3s',
          overflow: 'auto',
        }}
      >
        <Typography variant="h5" sx={{ mb: 3, fontWeight: 'bold' }}>
          Resources
        </Typography>

        {/* Filters */}
        <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
          <Grid container spacing={2} alignItems="center" sx={{ width: '100%' }}>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                fullWidth
                placeholder="Search by name or kind..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                variant="outlined"
                size="small"
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon />
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 2 }}>
              <FormControl fullWidth size="small">
                <InputLabel>Kind</InputLabel>
                <Select
                  value={filterKind}
                  label="Kind"
                  onChange={(e: SelectChangeEvent) => setFilterKind(e.target.value)}
                >
                  <MenuItem value="all">All Kinds</MenuItem>
                  {availableKinds.map(k => (
                    <MenuItem key={k} value={k}>{k}</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 2 }}>
              <FormControl fullWidth size="small">
                <InputLabel>Cluster</InputLabel>
                <Select
                  value={filterCluster}
                  label="Cluster"
                  onChange={(e: SelectChangeEvent) => setFilterCluster(e.target.value)}
                >
                  <MenuItem value="all">All Clusters</MenuItem>
                  {clusters.map(c => (
                    <MenuItem key={c} value={c}>{c}</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 2 }}>
              <FormControl fullWidth size="small">
                <InputLabel>Namespace</InputLabel>
                <Select
                  value={filterNamespace}
                  label="Namespace"
                  onChange={(e: SelectChangeEvent) => setFilterNamespace(e.target.value)}
                >
                  <MenuItem value="all">All Namespaces</MenuItem>
                  {namespaces.map(ns => (
                    <MenuItem key={ns} value={ns}>{ns}</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 2 }} sx={{ display: 'flex', justifyContent: 'flex-end' }}>
              <Tooltip title="Refresh">
                <IconButton onClick={loadResources} disabled={loading}>
                  {loading ? <CircularProgress size={24} /> : <RefreshIcon />}
                </IconButton>
              </Tooltip>
            </Grid>
          </Grid>
        </Paper>

        {/* Table */}
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Kind</TableCell>
                  <TableCell>Name</TableCell>
                  <TableCell>Namespace</TableCell>
                  <TableCell>Cluster</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>ManifestWork</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredResources.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} align="center">
                      <Typography sx={{ py: 2 }}>No resources found</Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredResources.map((r) => (
                    <TableRow
                      key={r.id}
                      hover
                      selected={selectedId === r.id}
                      sx={{ cursor: 'pointer', '& > td': { padding: '12px 16px' } }}
                      onClick={() => handleSelect(r)}
                    >
                      <TableCell>
                        <Chip label={r.kind} size="small" variant="outlined" />
                      </TableCell>
                      <TableCell>{r.name}</TableCell>
                      <TableCell>{r.namespace || '-'}</TableCell>
                      <TableCell>
                        <Typography
                          variant="body2"
                          sx={{ color: 'primary.main', cursor: 'pointer', '&:hover': { textDecoration: 'underline' } }}
                          onClick={(e) => { e.stopPropagation(); navigate(`/clusters/${r.cluster}`); }}
                        >
                          {r.cluster}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip label={r.status} size="small" color={statusColor(r.status)} />
                      </TableCell>
                      <TableCell>{r.manifestWorkName}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Box>

      {/* Detail drawer */}
      {selectedId && selectedResource && (
        <DrawerLayout
          title={`${selectedResource.kind}: ${selectedResource.name}`}
          onClose={handleCloseDetail}
        >
          <Box sx={{ mb: 2 }}>
            <Button
              variant="contained"
              onClick={handleViewFullDetails}
              endIcon={<LaunchIcon />}
            >
              View Full Details
            </Button>
          </Box>
          <ResourceDetailContent resource={selectedResource} compact />
        </DrawerLayout>
      )}
    </Box>
  );
}
