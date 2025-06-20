import { useState, useEffect } from 'react';
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
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material";
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Launch as LaunchIcon,
} from "@mui/icons-material";
import { fetchClusters } from '../api/clusterService';
import type { Cluster } from '../api/clusterService';
import { useCluster } from '../hooks/useCluster';
import ClusterDetailContent from './ClusterDetailContent';
import DrawerLayout from './layout/DrawerLayout';
import { fetchClusterAddons } from '../api/addonService';

// Gets the Hub Accepted status of a cluster
const getHubAcceptedStatus = (cluster: Cluster) => {
  return cluster.hubAccepted ? 'Accepted' : 'Not Accepted';
};

/**
 * Page component for displaying a list of clusters with a detail drawer
 */
export default function ClusterListPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState("all");
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(true);

  // Get selected cluster from URL query parameter
  const selectedClusterId = searchParams.get('selected');

  // Find the selected cluster in the list
  const selectedClusterData = clusters.find(c => c.id === selectedClusterId);

  // Use our custom hook to manage the selected cluster data
  const { cluster: detailCluster } = useCluster(
    selectedClusterId,
    { initialData: selectedClusterData, skipFetch: !selectedClusterId }
  );

  // Function to load clusters data
  const loadClusters = async () => {
    try {
      setLoading(true);
      const clusterData = await fetchClusters();
      setClusters(clusterData);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching cluster list:', error);
      setLoading(false);
    }
  };

  // Load clusters on component mount
  useEffect(() => {
    loadClusters();
  }, []);

  // Fetch addon counts/names for all clusters after clusters are loaded
  useEffect(() => {
    if (!clusters.length) return;
    let cancelled = false;
    async function fetchAllAddons() {
      const updated = await Promise.all(clusters.map(async (cluster) => {
        try {
          const addons = await fetchClusterAddons(cluster.name);
          return {
            ...cluster,
            addonCount: addons.length,
            addonNames: addons.map(a => a.name),
          };
        } catch {
          return { ...cluster, addonCount: 0, addonNames: [] };
        }
      }));
      if (!cancelled) {
        setClusters(updated);
      }
    }
    fetchAllAddons();
    return () => { cancelled = true; };
  }, [loading]);

  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(event.target.value);
  };

  const handleFilterStatusChange = (event: SelectChangeEvent) => {
    setFilterStatus(event.target.value);
  };

  const handleClusterSelect = (clusterId: string) => {
    // Update URL with selected cluster
    setSearchParams({ selected: clusterId });
  };

  const handleCloseDetail = () => {
    // Remove selected parameter from URL
    setSearchParams({});
  };

  const handleViewFullDetails = () => {
    if (selectedClusterId) {
      navigate(`/clusters/${selectedClusterData?.name}`);
    }
  };

  // Get status icon based on ManagedClusterConditionAvailable
  const getStatusIcon = (status: string) => {
    switch (status) {
      case "Online":
        return <CheckCircleIcon sx={{ color: "success.main" }} />;
      case "Offline":
        return <ErrorIcon sx={{ color: "error.main" }} />;
      default:
        return null;
    }
  };

  // Filter clusters based on search term and status filter
  const filteredClusters = clusters.filter(cluster => {
    const matchesSearch =
      cluster.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (cluster.labels?.region || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      // Search in clusterClaims
      cluster.clusterClaims?.some(claim =>
        claim.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        claim.value.toLowerCase().includes(searchTerm.toLowerCase())
      );
    const matchesStatus = filterStatus === 'all' || cluster.status === filterStatus;
    return matchesSearch && matchesStatus;
  });

  return (
    <Box sx={{ display: "flex", height: "calc(100vh - 64px)" }}>
      {/* Cluster list */}
      <Box
        sx={{
          flex: selectedClusterId ? "0 0 60%" : "1 1 auto",
          p: 3,
          transition: "flex 0.3s",
          overflow: "auto",
        }}
      >
        <Typography variant="h5" sx={{ mb: 3, fontWeight: "bold" }}>
          Clusters
        </Typography>

        {/* Filters and search */}
        <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
          <Grid container spacing={2} alignItems="center" sx={{ width: '100%' }}>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                fullWidth
                placeholder="Search clusters..."
                value={searchTerm}
                onChange={handleSearchChange}
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
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel id="filter-status-label">Status</InputLabel>
                <Select labelId="filter-status-label" value={filterStatus} label="Status" onChange={handleFilterStatusChange}>
                  <MenuItem value="all">All Statuses</MenuItem>
                  <MenuItem value="Online">Online</MenuItem>
                  <MenuItem value="Offline">Offline</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 2 }} sx={{ display: "flex", justifyContent: "flex-end" }}>
              <Tooltip title="Refresh">
                <IconButton onClick={loadClusters} disabled={loading}>
                  {loading ? <CircularProgress size={24} /> : <RefreshIcon />}
                </IconButton>
              </Tooltip>
            </Grid>
          </Grid>
        </Paper>

        {/* Cluster Table */}
        <TableContainer component={Paper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Version</TableCell>
                <TableCell>Hub Accepted</TableCell>
                <TableCell>Labels</TableCell>
                <TableCell>Cluster Claims</TableCell>
                <TableCell align="center">
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    Add-ons
                  </Box>
                </TableCell>
                <TableCell>Created</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredClusters.map((cluster) => (
                <TableRow
                  key={cluster.id}
                  hover
                  selected={selectedClusterId === cluster.id}
                  sx={{
                    cursor: "pointer",
                    py: 1.5,
                    '& > td': {
                      padding: '12px 16px',
                    }
                  }}
                  onClick={() => handleClusterSelect(cluster.id)}
                >
                  <TableCell>{cluster.name}</TableCell>
                  <TableCell>
                    <Box sx={{ display: "flex", alignItems: "center" }}>
                      {getStatusIcon(cluster.status)}
                      <Typography variant="body2" sx={{ ml: 1 }}>{cluster.status}</Typography>
                    </Box>
                  </TableCell>
                  <TableCell>{cluster.version || "-"}</TableCell>
                  <TableCell>{getHubAcceptedStatus(cluster)}</TableCell>
                  <TableCell>
                    {cluster.labels ? (
                      <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                        {Object.entries(cluster.labels).map(([key, value]) => (
                          <Chip
                            key={key}
                            label={`${key}: ${value}`}
                            size="small"
                            variant="outlined"
                          />
                        ))}
                      </Box>
                    ) : (
                      "-"
                    )}
                  </TableCell>
                  <TableCell>
                    {cluster.clusterClaims && cluster.clusterClaims.length > 0 ? (
                      <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                        {cluster.clusterClaims.map((claim) => (
                          <Chip
                            key={claim.name}
                            label={`${claim.name}: ${claim.value}`}
                            size="small"
                            variant="outlined"
                          />
                        ))}
                      </Box>
                    ) : (
                      "-"
                    )}
                  </TableCell>
                  <TableCell align="center">
                    {typeof cluster.addonCount === 'number' ? (
                      <Tooltip
                        title={
                          cluster.addonNames && cluster.addonNames.length > 0
                            ? cluster.addonNames.join(', ')
                            : 'No add-ons'
                        }
                        arrow
                      >
                        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                          <Typography variant="body2">{cluster.addonCount}</Typography>
                        </Box>
                      </Tooltip>
                    ) : (
                      <CircularProgress size={18} />
                    )}
                  </TableCell>
                  <TableCell>
                    {cluster.creationTimestamp
                      ? new Date(cluster.creationTimestamp).toLocaleDateString()
                      : "-"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Box>

      {/* Detail drawer */}
      {selectedClusterId && detailCluster && (
        <DrawerLayout
          title={detailCluster.name}
          onClose={handleCloseDetail}
        >
          <Box sx={{ mb: 2, display: "flex", justifyContent: "flex-start" }}>
            <Button
              variant="contained"
              onClick={handleViewFullDetails}
              endIcon={<LaunchIcon />}
            >
              View Full Details
            </Button>
          </Box>
          <ClusterDetailContent cluster={detailCluster} compact />
        </DrawerLayout>
      )}
    </Box>
  );
}
