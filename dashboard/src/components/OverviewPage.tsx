import { Box, Typography, Paper, Grid, alpha, useTheme, Chip } from "@mui/material"
import { Storage as StorageIcon, Layers as LayersIcon, DeviceHub as DeviceHubIcon, DynamicFeed as DynamicFeedIcon } from "@mui/icons-material"
import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { fetchClusters } from "../api/clusterService"
import { fetchClusterSets } from "../api/clusterSetService"
import { fetchPlacements } from "../api/placementService"
import { fetchManifestWorkReplicaSets } from "../api/manifestWorkReplicaSetService"
import { fetchAllManifestWorks } from "../api/manifestWorkService"
import type { Cluster } from "../api/clusterService"
import type { ClusterSet } from "../api/clusterSetService"
import type { Placement } from "../api/placementService"
import type { ManifestWorkReplicaSet } from "../api/manifestWorkReplicaSetService"
import type { ManifestWork } from "../api/manifestWorkService"
import { deriveMWRSStatus, buildMWRSChildMap } from "../utils/statusHelpers"

export default function OverviewPage() {
  const theme = useTheme()
  const navigate = useNavigate()
  const [clusters, setClusters] = useState<Cluster[]>([])
  const [clusterSets, setClusterSets] = useState<ClusterSet[]>([])
  const [placements, setPlacements] = useState<Placement[]>([])
  const [mwrsList, setMwrsList] = useState<ManifestWorkReplicaSet[]>([])
  const [allMWs, setAllMWs] = useState<ManifestWork[]>([])
  const [loading, setLoading] = useState(true)
  const [clusterSetsLoading, setClusterSetsLoading] = useState(true)
  const [placementsLoading, setPlacementsLoading] = useState(true)
  const [mwrsLoading, setMwrsLoading] = useState(true)
  const [clusterSetCounts, setClusterSetCounts] = useState<Record<string, number>>({})

  useEffect(() => {
    const loadClusters = async () => {
      setLoading(true)
      try {
        const data = await fetchClusters()
        setClusters(data)
      } finally {
        setLoading(false)
      }
    }
    loadClusters()
  }, [])

  useEffect(() => {
    const loadClusterSets = async () => {
      setClusterSetsLoading(true)
      try {
        const data = await fetchClusterSets()
        setClusterSets(data)
      } finally {
        setClusterSetsLoading(false)
      }
    }
    loadClusterSets()
  }, [])

  useEffect(() => {
    const loadPlacements = async () => {
      setPlacementsLoading(true)
      try {
        const data = await fetchPlacements()
        setPlacements(data)
      } finally {
        setPlacementsLoading(false)
      }
    }
    loadPlacements()
  }, [])

  useEffect(() => {
    const loadMwrs = async () => {
      setMwrsLoading(true)
      try {
        const [mwrsData, mwData] = await Promise.all([
          fetchManifestWorkReplicaSets(),
          fetchAllManifestWorks(),
        ])
        setMwrsList(mwrsData)
        setAllMWs(mwData)
      } finally {
        setMwrsLoading(false)
      }
    }
    loadMwrs()
  }, [])

  // Calculate cluster counts for each cluster set
  useEffect(() => {
    if (clusters.length === 0 || clusterSets.length === 0) return;

    const counts: Record<string, number> = {};

    clusterSets.forEach(clusterSet => {
      // Get the selector type from the cluster set
      const selectorType = clusterSet.spec?.clusterSelector?.selectorType || 'ExclusiveClusterSetLabel';
      let count = 0;

      // Filter clusters based on the selector type
      switch (selectorType) {
        case 'ExclusiveClusterSetLabel':
          // Use the exclusive cluster set label to filter clusters
          count = clusters.filter(cluster =>
            cluster.labels &&
            cluster.labels['cluster.open-cluster-management.io/clusterset'] === clusterSet.name
          ).length;
          break;

        case 'LabelSelector': {
          // Use the label selector to filter clusters
          const labelSelector = clusterSet.spec?.clusterSelector?.labelSelector;

          if (!labelSelector || Object.keys(labelSelector).length === 0) {
            // If labelSelector is empty, select all clusters (labels.Everything())
            count = clusters.length;
          } else {
            // Filter clusters based on the label selector
            count = clusters.filter(cluster => {
              if (!cluster.labels) return false;

              // Check if all matchLabels are satisfied
              for (const [key, value] of Object.entries(labelSelector)) {
                if (typeof value === 'string' && cluster.labels[key] !== value) {
                  return false;
                }
              }
              return true;
            }).length;
          }
        }
          break;

        default:
          count = 0;
      }

      counts[clusterSet.id] = count;
    });

    setClusterSetCounts(counts);
  }, [clusters, clusterSets]);

  // Calculate stats from real data
  const total = clusters.length
  // 只使用"Online"状态作为可用集群的判断标准
  const available = clusters.filter(c => c.status === "Online").length
  const totalClusterSets = clusterSets.length
  const totalPlacements = placements.length
  const successfulPlacements = placements.filter(p => p.succeeded).length

  const mwrsChildMap = useMemo(() => buildMWRSChildMap(allMWs), [allMWs])

  const totalMwrs = mwrsList.length
  const appliedMwrs = mwrsList.filter(m => {
    const childMWs = mwrsChildMap.get(`${m.namespace}/${m.name}`)
    return deriveMWRSStatus(m, childMWs) === 'Applied'
  }).length
  const degradedMwrs = mwrsList.filter(m => {
    const childMWs = mwrsChildMap.get(`${m.namespace}/${m.name}`)
    return deriveMWRSStatus(m, childMWs) === 'Degraded'
  })
  const failedMwrs = mwrsList.filter(m => {
    const childMWs = mwrsChildMap.get(`${m.namespace}/${m.name}`)
    return deriveMWRSStatus(m, childMWs) === 'Failed'
  })

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h5" sx={{ mb: 3, fontWeight: "bold" }}>
        Overview
      </Typography>

      {/* Simplified KPI cards */}
      <Grid container spacing={3} sx={{ width: '100%' }}>
        {/* Combined Clusters card */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper
            sx={{
              p: 3,
              height: "100%",
              borderRadius: 2,
              display: "flex",
              flexDirection: "column",
            }}
          >
            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  width: 48,
                  height: 48,
                  borderRadius: 2,
                  bgcolor: alpha(theme.palette.primary.main, 0.1),
                  mr: 2,
                }}
              >
                <StorageIcon sx={{ color: "primary.main", fontSize: 24 }} />
              </Box>
              <Box sx={{ display: "flex", alignItems: "flex-end" }}>
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">
                    All Clusters
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: "medium" }}>
                    {loading ? "-" : total}
                  </Typography>
                </Box>
                <Box sx={{ ml: 4 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Available
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: "medium", color: "success.main" }}>
                    {loading ? "-" : available}
                  </Typography>
                </Box>
              </Box>
            </Box>

            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              Availability Rate
            </Typography>

            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  height: 8,
                  width: "100%",
                  bgcolor: alpha(theme.palette.success.main, 0.1),
                  borderRadius: 4,
                  position: "relative",
                  overflow: "hidden",
                }}
              >
                <Box
                  sx={{
                    position: "absolute",
                    left: 0,
                    top: 0,
                    height: "100%",
                    width: total > 0 ? `${(available / total) * 100}%` : 0,
                    bgcolor: "success.main",
                    borderRadius: 4,
                  }}
                />
              </Box>
              <Typography variant="body2" fontWeight="medium" sx={{ ml: 2, minWidth: 40 }}>
                {loading || total === 0 ? '-' : Math.round((available / total) * 100)}%
              </Typography>
            </Box>

            <Box sx={{ mt: "auto" }}>
              <Typography variant="body2" color="text.secondary">
                {loading ? '-' : total - available} clusters currently unavailable
              </Typography>
            </Box>
          </Paper>
        </Grid>

        {/* ManagedClusterSets card */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper
            sx={{
              p: 3,
              height: "100%",
              borderRadius: 2,
              display: "flex",
              flexDirection: "column",
            }}
          >
            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  width: 48,
                  height: 48,
                  borderRadius: 2,
                  bgcolor: alpha(theme.palette.info.main, 0.1),
                  mr: 2,
                }}
              >
                <LayersIcon sx={{ color: "info.main", fontSize: 24 }} />
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">
                  ManagedClusterSets
                </Typography>
                <Typography variant="h3" sx={{ fontWeight: "medium" }}>
                  {clusterSetsLoading ? "-" : totalClusterSets}
                </Typography>
              </Box>
            </Box>

            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              Cluster distribution
            </Typography>

            {!clusterSetsLoading && clusterSets.length > 0 && (
              <Box sx={{ mt: "auto" }}>
                {clusterSets.slice(0, 3).map((set) => (
                  <Box key={set.id} sx={{ display: "flex", justifyContent: "space-between", mb: 1 }}>
                    <Typography variant="body2" sx={{ overflow: "hidden", textOverflow: "ellipsis" }}>
                      {set.name}
                    </Typography>
                    <Typography variant="body2" fontWeight="medium">
                      {clusterSetCounts[set.id] || 0} clusters
                    </Typography>
                  </Box>
                ))}
                {clusterSets.length > 3 && (
                  <Typography variant="body2" color="text.secondary" sx={{ textAlign: "center", mt: 1 }}>
                    + {clusterSets.length - 3} more sets
                  </Typography>
                )}
              </Box>
            )}

            {clusterSetsLoading && (
              <Box sx={{ display: "flex", justifyContent: "center", mt: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  Loading cluster sets...
                </Typography>
              </Box>
            )}

            {!clusterSetsLoading && clusterSets.length === 0 && (
              <Box sx={{ display: "flex", justifyContent: "center", mt: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  No cluster sets found
                </Typography>
              </Box>
            )}
          </Paper>
        </Grid>

        {/* Placements card */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper
            sx={{
              p: 3,
              height: "100%",
              borderRadius: 2,
              display: "flex",
              flexDirection: "column",
            }}
          >
            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  width: 48,
                  height: 48,
                  borderRadius: 2,
                  bgcolor: alpha(theme.palette.success.main, 0.1),
                  mr: 2,
                }}
              >
                <DeviceHubIcon sx={{ color: "success.main", fontSize: 24 }} />
              </Box>
              <Box sx={{ display: "flex", alignItems: "flex-end" }}>
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">
                    All Placements
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: "medium" }}>
                    {placementsLoading ? "-" : totalPlacements}
                  </Typography>
                </Box>
                <Box sx={{ ml: 4 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Successful
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: "medium", color: "success.main" }}>
                    {placementsLoading ? "-" : successfulPlacements}
                  </Typography>
                </Box>
              </Box>
            </Box>

            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              Success Rate
            </Typography>

            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  height: 8,
                  width: "100%",
                  bgcolor: alpha(theme.palette.success.main, 0.1),
                  borderRadius: 4,
                  position: "relative",
                  overflow: "hidden",
                }}
              >
                <Box
                  sx={{
                    position: "absolute",
                    left: 0,
                    top: 0,
                    height: "100%",
                    width: totalPlacements > 0 ? `${(successfulPlacements / totalPlacements) * 100}%` : 0,
                    bgcolor: "success.main",
                    borderRadius: 4,
                  }}
                />
              </Box>
              <Typography variant="body2" fontWeight="medium" sx={{ ml: 2, minWidth: 40 }}>
                {placementsLoading || totalPlacements === 0 ? '-' : Math.round((successfulPlacements / totalPlacements) * 100)}%
              </Typography>
            </Box>

            <Box sx={{ mt: "auto" }}>
              <Typography variant="body2" color="text.secondary">
                {placementsLoading ? '-' : totalPlacements - successfulPlacements} placements currently pending or failed
              </Typography>
            </Box>
          </Paper>
        </Grid>
        {/* WorkReplicaSets card */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper
            sx={{
              p: 3,
              height: "100%",
              borderRadius: 2,
              display: "flex",
              flexDirection: "column",
            }}
          >
            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  width: 48,
                  height: 48,
                  borderRadius: 2,
                  bgcolor: alpha(theme.palette.warning.main, 0.1),
                  mr: 2,
                }}
              >
                <DynamicFeedIcon sx={{ color: "warning.main", fontSize: 24 }} />
              </Box>
              <Box sx={{ display: "flex", alignItems: "flex-end" }}>
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">
                    WorkReplicaSets
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: "medium" }}>
                    {mwrsLoading ? "-" : totalMwrs}
                  </Typography>
                </Box>
                <Box sx={{ ml: 4 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Applied
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: "medium", color: "success.main" }}>
                    {mwrsLoading ? "-" : appliedMwrs}
                  </Typography>
                </Box>
              </Box>
            </Box>

            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              Applied Rate
            </Typography>

            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  height: 8,
                  width: "100%",
                  bgcolor: alpha(theme.palette.success.main, 0.1),
                  borderRadius: 4,
                  position: "relative",
                  overflow: "hidden",
                }}
              >
                <Box
                  sx={{
                    position: "absolute",
                    left: 0,
                    top: 0,
                    height: "100%",
                    width: totalMwrs > 0 ? `${(appliedMwrs / totalMwrs) * 100}%` : 0,
                    bgcolor: "success.main",
                    borderRadius: 4,
                  }}
                />
              </Box>
              <Typography variant="body2" fontWeight="medium" sx={{ ml: 2, minWidth: 40 }}>
                {mwrsLoading || totalMwrs === 0 ? '-' : Math.round((appliedMwrs / totalMwrs) * 100)}%
              </Typography>
            </Box>

            <Box sx={{ mt: "auto" }}>
              {!mwrsLoading && (degradedMwrs.length > 0 || failedMwrs.length > 0) ? (
                <Box sx={{ display: "flex", flexDirection: "column", gap: 0.5 }}>
                  {degradedMwrs.length > 0 && (
                    <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, alignItems: "center" }}>
                      <Typography variant="body2" color="warning.main" sx={{ mr: 0.5 }}>
                        {degradedMwrs.length} degraded:
                      </Typography>
                      {degradedMwrs.slice(0, 3).map(m => (
                        <Chip
                          key={m.id}
                          label={`${m.namespace}/${m.name}`}
                          size="small"
                          color="warning"
                          variant="outlined"
                          clickable
                          onClick={() => navigate(`/manifestworkreplicasets/${m.namespace}/${m.name}`)}
                        />
                      ))}
                      {degradedMwrs.length > 3 && (
                        <Chip
                          label={`+${degradedMwrs.length - 3} more`}
                          size="small"
                          variant="outlined"
                          clickable
                          onClick={() => navigate('/manifestworkreplicasets')}
                        />
                      )}
                    </Box>
                  )}
                  {failedMwrs.length > 0 && (
                    <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, alignItems: "center" }}>
                      <Typography variant="body2" color="error.main" sx={{ mr: 0.5 }}>
                        {failedMwrs.length} failed:
                      </Typography>
                      {failedMwrs.slice(0, 3).map(m => (
                        <Chip
                          key={m.id}
                          label={`${m.namespace}/${m.name}`}
                          size="small"
                          color="error"
                          variant="outlined"
                          clickable
                          onClick={() => navigate(`/manifestworkreplicasets/${m.namespace}/${m.name}`)}
                        />
                      ))}
                      {failedMwrs.length > 3 && (
                        <Chip
                          label={`+${failedMwrs.length - 3} more`}
                          size="small"
                          variant="outlined"
                          clickable
                          onClick={() => navigate('/manifestworkreplicasets')}
                        />
                      )}
                    </Box>
                  )}
                </Box>
              ) : (
                <Typography variant="body2" color="text.secondary">
                  {mwrsLoading ? '-' : totalMwrs - appliedMwrs > 0
                    ? `${totalMwrs - appliedMwrs} WorkReplicaSets not yet applied`
                    : 'All WorkReplicaSets applied successfully'}
                </Typography>
              )}
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  )
}