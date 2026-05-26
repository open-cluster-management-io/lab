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
import { fetchManifestWorkReplicaSets } from '../api/manifestWorkReplicaSetService';
import type { ManifestWorkReplicaSet } from '../api/manifestWorkReplicaSetService';
import { fetchAllManifestWorks } from '../api/manifestWorkService';
import type { ManifestWork } from '../api/manifestWorkService';
import DrawerLayout from './layout/DrawerLayout';
import { deriveMWRSStatus, chipColor, buildMWRSChildMap } from '../utils/statusHelpers';

const formatDate = (dateString?: string) => {
  if (!dateString) return '-';
  return new Date(dateString).toLocaleDateString();
};

export default function ManifestWorkReplicaSetListPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [searchTerm, setSearchTerm] = useState("");
  const [filterNamespace, setFilterNamespace] = useState("all");
  const [items, setItems] = useState<ManifestWorkReplicaSet[]>([]);
  const [allMWs, setAllMWs] = useState<ManifestWork[]>([]);
  const [loading, setLoading] = useState(true);

  const selectedId = searchParams.get('selected');
  const selectedItem = items.find(i => i.id === selectedId);

  const mwrsChildMap = useMemo(() => buildMWRSChildMap(allMWs), [allMWs]);

  const getStatusInfo = (mwrs: ManifestWorkReplicaSet): { label: string; color: 'success' | 'error' | 'warning' | 'default' } => {
    const childMWs = mwrsChildMap.get(`${mwrs.namespace}/${mwrs.name}`);
    const label = deriveMWRSStatus(mwrs, childMWs);
    return { label, color: chipColor(label) };
  };

  const getStatusIcon = (mwrs: ManifestWorkReplicaSet) => {
    const status = getStatusInfo(mwrs);
    if (status.color === 'success') return <CheckCircleIcon sx={{ color: "success.main" }} />;
    if (status.color === 'error') return <ErrorIcon sx={{ color: "error.main" }} />;
    if (status.color === 'warning') return <WarningIcon sx={{ color: "warning.main" }} />;
    return <ErrorIcon sx={{ color: "text.disabled" }} />;
  };

  const uniqueNamespaces = useMemo(() => {
    const namespaces = new Set<string>();
    items.forEach(item => {
      if (item.namespace) namespaces.add(item.namespace);
    });
    return Array.from(namespaces).sort();
  }, [items]);

  useEffect(() => {
    const load = async () => {
      try {
        setLoading(true);
        const [mwrsData, mwData] = await Promise.all([
          fetchManifestWorkReplicaSets(),
          fetchAllManifestWorks(),
        ]);
        setItems(mwrsData);
        setAllMWs(mwData);
      } catch (error) {
        console.error('Error fetching ManifestWorkReplicaSets:', error);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(event.target.value);
  };

  const handleFilterNamespaceChange = (event: SelectChangeEvent) => {
    setFilterNamespace(event.target.value);
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

    const matchesNamespace =
      filterNamespace === 'all' || item.namespace === filterNamespace;

    return matchesSearch && matchesNamespace;
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
          WorkReplicaSets
        </Typography>

        {/* Filters */}
        <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
          <Grid container spacing={2} alignItems="center" sx={{ width: '100%' }}>
            <Grid size={{ xs: 12, md: 5 }}>
              <TextField
                fullWidth
                placeholder="Search works..."
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
                <InputLabel id="filter-namespace-label">Namespace</InputLabel>
                <Select labelId="filter-namespace-label" value={filterNamespace} label="Namespace" onChange={handleFilterNamespaceChange}>
                  <MenuItem value="all">All Namespaces</MenuItem>
                  {uniqueNamespaces.map(ns => (
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
                  <TableCell>Namespace</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Placements</TableCell>
                  <TableCell align="center">Applied / Total</TableCell>
                  <TableCell>Created</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredItems.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} align="center">
                      <Typography sx={{ py: 2 }}>
                        No ManifestWorkReplicaSets found
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
                        <TableCell>
                          <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                            {item.placementRefs?.map(ref => (
                              <Chip key={ref.name} label={ref.name} size="small" variant="outlined" />
                            )) ?? '-'}
                          </Box>
                        </TableCell>
                        <TableCell align="center">
                          {item.summary.applied} / {item.summary.total}
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
          {/* Overview */}
          <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>Overview</Typography>

            <Box sx={{ display: "flex", mb: 1 }}>
              <Typography variant="subtitle2" sx={{ width: 160 }}>Status:</Typography>
              <Box sx={{ display: "flex", alignItems: "center" }}>
                {getStatusIcon(selectedItem)}
                <Typography variant="body2" sx={{ ml: 1 }}>{getStatusInfo(selectedItem).label}</Typography>
              </Box>
            </Box>

            <Box sx={{ display: "flex", mb: 1 }}>
              <Typography variant="subtitle2" sx={{ width: 160 }}>Placements:</Typography>
              <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                {selectedItem.placementRefs?.map(ref => (
                  <Chip key={ref.name} label={ref.name} size="small" variant="outlined" />
                )) ?? <Typography variant="body2" color="text.secondary">None</Typography>}
              </Box>
            </Box>

            <Box sx={{ display: "flex", mb: 1 }}>
              <Typography variant="subtitle2" sx={{ width: 160 }}>Created:</Typography>
              <Typography>
                {selectedItem.creationTimestamp ? new Date(selectedItem.creationTimestamp).toLocaleString() : '-'}
              </Typography>
            </Box>
          </Paper>

          {/* Summary */}
          <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>Summary</Typography>
            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap" }}>
              <Chip label={`Total: ${selectedItem.summary.total}`} />
              <Chip label={`Applied: ${selectedItem.summary.applied}`} color="success" />
              <Chip label={`Available: ${selectedItem.summary.available}`} color="success" variant="outlined" />
              {selectedItem.summary.progressing > 0 && (
                <Chip label={`Progressing: ${selectedItem.summary.progressing}`} color="warning" />
              )}
              {selectedItem.summary.degraded > 0 && (
                <Chip label={`Degraded: ${selectedItem.summary.degraded}`} color="error" />
              )}
            </Box>
          </Paper>

          {/* Conditions */}
          {selectedItem.conditions && selectedItem.conditions.length > 0 && (
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
                    {selectedItem.conditions.map((condition, idx) => (
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

          {/* View Full Details button */}
          <Box sx={{ display: "flex", justifyContent: "center" }}>
            <Chip
              label="View Full Details"
              clickable
              color="primary"
              onClick={() => navigate(`/manifestworkreplicasets/${selectedItem.namespace}/${selectedItem.name}`)}
            />
          </Box>
        </DrawerLayout>
      )}
    </Box>
  );
}
