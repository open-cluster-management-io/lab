import { useState, useEffect, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Box,
  Typography,
  Paper,
  Chip,
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
  Warning as WarningIcon,
} from "@mui/icons-material";
import { fetchAllManifestWorks } from '../api/manifestWorkService';
import type { ManifestWork } from '../api/manifestWorkService';
import DrawerLayout from './layout/DrawerLayout';
import ManifestWorkDetailContent from './ManifestWorkDetailContent';
import { deriveMWStatus, chipColor } from '../utils/statusHelpers';

const getStatusInfo = (mw: ManifestWork): { label: string; color: 'success' | 'error' | 'warning' | 'default' } => {
  const label = deriveMWStatus(mw);
  return { label, color: chipColor(label) };
};

const getStatusIcon = (mw: ManifestWork) => {
  const status = getStatusInfo(mw);
  if (status.color === 'success') return <CheckCircleIcon sx={{ color: "success.main" }} />;
  if (status.color === 'error') return <ErrorIcon sx={{ color: "error.main" }} />;
  if (status.color === 'warning') return <WarningIcon sx={{ color: "warning.main" }} />;
  return <ErrorIcon sx={{ color: "text.disabled" }} />;
};

const formatDate = (dateString?: string) => {
  if (!dateString) return '-';
  return new Date(dateString).toLocaleDateString();
};

export default function ManifestWorkListPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [searchTerm, setSearchTerm] = useState("");
  const [filterCluster, setFilterCluster] = useState("all");
  const [items, setItems] = useState<ManifestWork[]>([]);
  const [loading, setLoading] = useState(true);

  const selectedId = searchParams.get('selected');
  const selectedItem = items.find(i => i.id === selectedId);

  const uniqueClusters = useMemo(() => {
    const clusters = new Set<string>();
    items.forEach(item => {
      if (item.namespace) clusters.add(item.namespace);
    });
    return Array.from(clusters).sort();
  }, [items]);

  useEffect(() => {
    const load = async () => {
      try {
        setLoading(true);
        const data = await fetchAllManifestWorks();
        setItems(data);
      } catch (error) {
        console.error('Error fetching ManifestWorks:', error);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(event.target.value);
  };

  const handleFilterClusterChange = (event: SelectChangeEvent) => {
    setFilterCluster(event.target.value);
  };

  const handleSelect = (id: string) => {
    setSearchParams({ selected: id });
  };

  const handleCloseDetail = () => {
    setSearchParams({});
  };

  const filteredItems = items.filter(item => {
    const matchesSearch =
      item.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      item.namespace.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesCluster =
      filterCluster === 'all' || item.namespace === filterCluster;

    return matchesSearch && matchesCluster;
  });

  return (
    <Box sx={{ display: "flex", height: "calc(100vh - 64px)" }}>
      {/* List */}
      <Box
        sx={{
          flex: selectedId ? "0 0 60%" : "1 1 auto",
          p: 3,
          transition: "flex 0.3s",
          overflow: "auto",
        }}
      >
        <Typography variant="h5" sx={{ mb: 3, fontWeight: "bold" }}>
          ManifestWorks
        </Typography>

        {/* Filters */}
        <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
          <Grid container spacing={2} alignItems="center" sx={{ width: '100%' }}>
            <Grid size={{ xs: 12, md: 5 }}>
              <TextField
                fullWidth
                placeholder="Search manifest works..."
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
                <InputLabel id="filter-cluster-label">Cluster</InputLabel>
                <Select labelId="filter-cluster-label" value={filterCluster} label="Cluster" onChange={handleFilterClusterChange}>
                  <MenuItem value="all">All Clusters</MenuItem>
                  {uniqueClusters.map(ns => (
                    <MenuItem key={ns} value={ns}>{ns}</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 1 }} sx={{ display: "flex", justifyContent: "flex-end" }}>
              <Tooltip title="Refresh">
                <IconButton onClick={() => window.location.reload()}>
                  <RefreshIcon />
                </IconButton>
              </Tooltip>
            </Grid>
          </Grid>
        </Paper>

        {/* Table */}
        {loading ? (
          <Box sx={{ display: "flex", justifyContent: "center", mt: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Cluster</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell align="center">Resources</TableCell>
                  <TableCell>Created</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredItems.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} align="center">
                      <Typography sx={{ py: 2 }}>
                        No ManifestWorks found
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredItems.map((item) => {
                    const status = getStatusInfo(item);
                    return (
                      <TableRow
                        key={item.id}
                        hover
                        selected={selectedId === item.id}
                        sx={{
                          cursor: "pointer",
                          '& > td': { padding: '12px 16px' },
                        }}
                        onClick={() => handleSelect(item.id)}
                      >
                        <TableCell>{item.name}</TableCell>
                        <TableCell>{item.namespace}</TableCell>
                        <TableCell>
                          <Box sx={{ display: "flex", alignItems: "center" }}>
                            {getStatusIcon(item)}
                            <Typography variant="body2" sx={{ ml: 1 }}>
                              {status.label}
                            </Typography>
                          </Box>
                        </TableCell>
                        <TableCell align="center">
                          {item.resourceStatus?.manifests?.length ?? 0}
                        </TableCell>
                        <TableCell>{formatDate(item.creationTimestamp)}</TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Box>

      {/* Detail drawer */}
      {selectedId && selectedItem && (
        <DrawerLayout
          title={`${selectedItem.namespace}/${selectedItem.name}`}
          onClose={handleCloseDetail}
        >
          <ManifestWorkDetailContent mw={selectedItem} compact />
          <Box sx={{ display: "flex", justifyContent: "center", mt: 2 }}>
            <Chip
              label="View Full Details"
              clickable
              color="primary"
              onClick={() => navigate(`/manifestworks/${selectedItem.namespace}/${selectedItem.name}`)}
            />
          </Box>
        </DrawerLayout>
      )}
    </Box>
  );
}
