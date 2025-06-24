import { useState, useEffect, useMemo } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
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
} from "@mui/icons-material";
import { fetchPlacements, fetchPlacementDecisions } from '../api/placementService';
import type { Placement, PlacementDecision } from '../api/placementService';
import DrawerLayout from './layout/DrawerLayout';

/**
 * Page component for displaying a list of placements
 */
export default function PlacementListPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState("all");
  const [filterNamespace, setFilterNamespace] = useState("all");
  const [placements, setPlacements] = useState<Placement[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingDecisions, setLoadingDecisions] = useState(false);
  const [placementDecisions, setPlacementDecisions] = useState<PlacementDecision[]>([]);

  // Get selected placement from URL query parameter
  const selectedPlacementId = searchParams.get('selected');

  // Find the selected placement in the list
  const selectedPlacementData = placements.find(p => p.id === selectedPlacementId);

  // Get unique namespaces from placements
  const uniqueNamespaces = useMemo(() => {
    const namespaces = new Set<string>();
    placements.forEach(placement => {
      if (placement.namespace) {
        namespaces.add(placement.namespace);
      }
    });
    return Array.from(namespaces).sort();
  }, [placements]);

  // Load placements on component mount
  useEffect(() => {
    const loadPlacements = async () => {
      try {
        setLoading(true);
        const placementData = await fetchPlacements();
        setPlacements(placementData);
        setLoading(false);
      } catch (error) {
        console.error('Error fetching placement list:', error);
        setLoading(false);
      }
    };

    loadPlacements();
  }, []);

  // Load placement decisions for a selected placement
  useEffect(() => {
    const loadPlacementDecisions = async () => {
      if (selectedPlacementData) {
        try {
          setLoadingDecisions(true);
          const decisions = await fetchPlacementDecisions(
            selectedPlacementData.namespace,
            selectedPlacementData.name
          );
          setPlacementDecisions(decisions);
          setLoadingDecisions(false);
        } catch (error) {
          console.error('Error fetching placement decisions:', error);
          setLoadingDecisions(false);
        }
      } else {
        // If no placement is selected, clear decisions
        setPlacementDecisions([]);
      }
    };

    // Load decisions for both the Clusters tab and the Decisions tab
    if (selectedPlacementData) {
      loadPlacementDecisions();
    }
  }, [selectedPlacementData]);

  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(event.target.value);
  };

  const handleFilterStatusChange = (event: SelectChangeEvent) => {
    setFilterStatus(event.target.value);
  };

  const handleFilterNamespaceChange = (event: SelectChangeEvent) => {
    setFilterNamespace(event.target.value);
  };

  const handlePlacementSelect = (placementId: string) => {
    // Update URL with selected placement
    setSearchParams({ selected: placementId });
  };

  const handleCloseDetail = () => {
    // Remove selected parameter from URL
    setSearchParams({});
  };

  // Get status icon based on succeeded condition
  const getStatusIcon = (succeeded: boolean) => {
    return succeeded ? (
      <CheckCircleIcon sx={{ color: "success.main" }} />
    ) : (
      <ErrorIcon sx={{ color: "error.main" }} />
    );
  };

  // Filter placements based on search term, status filter, and namespace filter
  const filteredPlacements = placements.filter(placement => {
    const matchesSearch =
      placement.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      placement.namespace.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesStatus =
      filterStatus === 'all' ||
      (filterStatus === 'succeeded' && placement.succeeded) ||
      (filterStatus === 'not-succeeded' && !placement.succeeded);

    const matchesNamespace =
      filterNamespace === 'all' ||
      placement.namespace === filterNamespace;

    return matchesSearch && matchesStatus && matchesNamespace;
  });

  const placementStatusText = (placement: Placement) => {
    if (placement.succeeded) {
      return "Succeeded";
    } else {
      return placement.reasonMessage || "Not Succeeded";
    }
  };

  return (
    <Box sx={{ display: "flex", height: "calc(100vh - 64px)" }}>
      {/* Placement list */}
      <Box
        sx={{
          flex: selectedPlacementId ? "0 0 60%" : "1 1 auto",
          p: 3,
          transition: "flex 0.3s",
          overflow: "auto",
        }}
      >
        <Typography variant="h5" sx={{ mb: 3, fontWeight: "bold" }}>
          Placements
        </Typography>

        {/* Filters and search */}
        <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
          <Grid container spacing={2} alignItems="center" sx={{ width: '100%' }}>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                fullWidth
                placeholder="Search placements..."
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
            <Grid size={{ xs: 12, md: 3 }}>
              <FormControl fullWidth size="small">
                <InputLabel id="filter-status-label">Status</InputLabel>
                <Select labelId="filter-status-label" value={filterStatus} label="Status" onChange={handleFilterStatusChange}>
                  <MenuItem value="all">All Statuses</MenuItem>
                  <MenuItem value="succeeded">Succeeded</MenuItem>
                  <MenuItem value="not-succeeded">Not Succeeded</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <FormControl fullWidth size="small">
                <InputLabel id="filter-namespace-label">Namespace</InputLabel>
                <Select labelId="filter-namespace-label" value={filterNamespace} label="Namespace" onChange={handleFilterNamespaceChange}>
                  <MenuItem value="all">All Namespaces</MenuItem>
                  {uniqueNamespaces.map(namespace => (
                    <MenuItem key={namespace} value={namespace}>{namespace}</MenuItem>
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

        {/* Placement Table */}
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
                  <TableCell align="center">Requested</TableCell>
                  <TableCell align="center">Selected</TableCell>
                  <TableCell>ClusterSets</TableCell>
                  <TableCell>Created</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredPlacements.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} align="center">
                      <Typography sx={{ py: 2 }}>
                        No placements found
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredPlacements.map((placement) => (
                    <TableRow
                      key={placement.id}
                      hover
                      selected={selectedPlacementId === placement.id}
                      sx={{
                        cursor: "pointer",
                        py: 1.5,
                        '& > td': {
                          padding: '12px 16px',
                        }
                      }}
                      onClick={() => handlePlacementSelect(placement.id)}
                    >
                      <TableCell>{placement.name}</TableCell>
                      <TableCell>{placement.namespace}</TableCell>
                      <TableCell>
                        <Box sx={{ display: "flex", alignItems: "center" }}>
                          {getStatusIcon(!!placement.succeeded)}
                          <Typography variant="body2" sx={{ ml: 1 }}>
                            {placementStatusText(placement)}
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell align="center">
                        {placement.numberOfClusters ?? "All"}
                      </TableCell>
                      <TableCell align="center">
                        {placement.numberOfSelectedClusters}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                          {placement.clusterSets && placement.clusterSets.length > 0 ? (
                            placement.clusterSets.map((set) => (
                              <Chip
                                key={set}
                                label={set}
                                size="small"
                                variant="outlined"
                              />
                            ))
                          ) : (
                            <Typography variant="body2" color="text.secondary">-</Typography>
                          )}
                        </Box>
                      </TableCell>
                      <TableCell>
                        {placement.creationTimestamp
                          ? new Date(placement.creationTimestamp).toLocaleDateString()
                          : "-"}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Box>

      {/* Detail drawer */}
      {selectedPlacementId && selectedPlacementData && (
        <DrawerLayout
          title={`${selectedPlacementData.namespace}/${selectedPlacementData.name}`}
          onClose={handleCloseDetail}
        >


          {/* Basic Placement Info */}
          <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Overview
            </Typography>

            <Box sx={{ display: "flex", mb: 1 }}>
              <Typography variant="subtitle2" sx={{ width: 160 }}>Status:</Typography>
              <Box sx={{ display: "flex", alignItems: "center" }}>
                {getStatusIcon(!!selectedPlacementData.succeeded)}
                <Typography variant="body2" sx={{ ml: 1 }}>
                  {placementStatusText(selectedPlacementData)}
                </Typography>
              </Box>
            </Box>

            <Box sx={{ display: "flex", mb: 1 }}>
              <Typography variant="subtitle2" sx={{ width: 160 }}>Requested Clusters:</Typography>
              <Typography>{selectedPlacementData.numberOfClusters ?? "All matching"}</Typography>
            </Box>

            <Box sx={{ display: "flex", mb: 1 }}>
              <Typography variant="subtitle2" sx={{ width: 160 }}>Selected Clusters:</Typography>
              <Typography>{selectedPlacementData.numberOfSelectedClusters}</Typography>
            </Box>

            <Box sx={{ display: "flex", mb: 1 }}>
              <Typography variant="subtitle2" sx={{ width: 160 }}>ClusterSets:</Typography>
              <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                {selectedPlacementData.clusterSets && selectedPlacementData.clusterSets.length > 0 ? (
                  selectedPlacementData.clusterSets.map((set) => (
                    <Chip
                      key={set}
                      label={set}
                      size="small"
                      variant="outlined"
                    />
                  ))
                ) : (
                  <Typography variant="body2" color="text.secondary">None</Typography>
                )}
              </Box>
            </Box>

            <Box sx={{ display: "flex", mb: 1 }}>
              <Typography variant="subtitle2" sx={{ width: 160 }}>Created:</Typography>
              <Typography>
                {selectedPlacementData.creationTimestamp
                  ? new Date(selectedPlacementData.creationTimestamp).toLocaleString()
                  : "-"}
              </Typography>
            </Box>
          </Paper>

          {/* Decision Groups */}
          {selectedPlacementData.decisionGroups && selectedPlacementData.decisionGroups.length > 0 && (
            <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
              <Typography variant="h6" sx={{ mb: 2 }}>
                Clusters
              </Typography>

              {loadingDecisions ? (
                <Box sx={{ display: "flex", justifyContent: "center", p: 2 }}>
                  <CircularProgress size={24} />
                </Box>
              ) : (
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Name</TableCell>
                        <TableCell>Group Index</TableCell>
                        <TableCell>Group Name</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {selectedPlacementData.decisionGroups.flatMap((group) => {
                        // For each decision group
                        const rows: React.ReactNode[] = [];

                        // Find the corresponding placement decisions for this group
                        const groupDecisions = placementDecisions.filter(decision =>
                          group.decisions.includes(decision.name)
                        );

                        // Extract cluster names from each placement decision's status
                        groupDecisions.forEach(decision => {
                          if (decision.decisions) {
                            decision.decisions.forEach((decisionStatus: {clusterName: string; reason: string}) => {
                              if (decisionStatus.clusterName) {
                                rows.push(
                                  <TableRow key={`${group.decisionGroupIndex}-${decisionStatus.clusterName}`}>
                                    <TableCell>{decisionStatus.clusterName}</TableCell>
                                    <TableCell>{group.decisionGroupIndex}</TableCell>
                                    <TableCell>{group.decisionGroupName || "(default)"}</TableCell>
                                  </TableRow>
                                );
                              }
                            });
                          }
                        });

                        return rows;
                      })}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </Paper>
          )}

          {/* Selected Clusters */}
          {selectedPlacementData.selectedClusters && selectedPlacementData.selectedClusters.length > 0 && (
            <Paper sx={{ p: 2, mb: 3, borderRadius: 2 }}>
              <Typography variant="h6" sx={{ mb: 2 }}>
                Selected Clusters
              </Typography>

              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Cluster Name</TableCell>
                      <TableCell>Status</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {selectedPlacementData.selectedClusters.map((cluster) => (
                      <TableRow
                        key={cluster.id}
                        hover
                        sx={{ cursor: "pointer" }}
                        onClick={() => navigate(`/clusters/${cluster.name}`)}
                      >
                        <TableCell>{cluster.name}</TableCell>
                        <TableCell>
                          <Box sx={{ display: "flex", alignItems: "center" }}>
                            {cluster.status === "Online" ? (
                              <CheckCircleIcon sx={{ color: "success.main", fontSize: 16, mr: 1 }} />
                            ) : (
                              <ErrorIcon sx={{ color: "error.main", fontSize: 16, mr: 1 }} />
                            )}
                            {cluster.status}
                          </Box>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Paper>
          )}
        </DrawerLayout>
      )}
    </Box>
  );
}