import { Box, Typography, Paper, Grid, Skeleton, Chip } from "@mui/material"
import { Storage as StorageIcon, Layers as LayersIcon, DeviceHub as DeviceHubIcon, Warning as WarningIcon, Error as ErrorIcon, CheckCircle as CheckCircleIcon } from "@mui/icons-material"
import { useEffect, useState, useMemo } from "react"
import { PieChart } from "@mui/x-charts/PieChart"
import { BarChart } from "@mui/x-charts/BarChart"
import { fetchClusters } from "../api/clusterService"
import { fetchClusterSets } from "../api/clusterSetService"
import { fetchPlacements } from "../api/placementService"
import type { Cluster } from "../api/clusterService"
import type { ClusterSet } from "../api/clusterSetService"
import type { Placement } from "../api/placementService"

const cardStyles = [
  {
    gradient: "linear-gradient(90deg, #0066cc, #2b9af3)",
    iconBg: "linear-gradient(135deg, #0066cc 0%, #2b9af3 100%)",
    progressBg: "linear-gradient(90deg, #0066cc 0%, #2b9af3 100%)",
  },
  {
    gradient: "linear-gradient(90deg, #2b9af3, #5cb3f5)",
    iconBg: "linear-gradient(135deg, #2b9af3 0%, #5cb3f5 100%)",
    progressBg: "linear-gradient(90deg, #2b9af3 0%, #5cb3f5 100%)",
  },
  {
    gradient: "linear-gradient(90deg, #3e8635, #4caf50)",
    iconBg: "linear-gradient(135deg, #3e8635 0%, #4caf50 100%)",
    progressBg: "linear-gradient(90deg, #3e8635 0%, #4caf50 100%)",
  },
]

export default function OverviewPage() {
  const [clusters, setClusters] = useState<Cluster[]>([])
  const [clusterSets, setClusterSets] = useState<ClusterSet[]>([])
  const [placements, setPlacements] = useState<Placement[]>([])
  const [loading, setLoading] = useState(true)
  const [clusterSetsLoading, setClusterSetsLoading] = useState(true)
  const [placementsLoading, setPlacementsLoading] = useState(true)
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
    if (clusters.length === 0 || clusterSets.length === 0) return;

    const counts: Record<string, number> = {};

    clusterSets.forEach(clusterSet => {
      const selectorType = clusterSet.spec?.clusterSelector?.selectorType || 'ExclusiveClusterSetLabel';
      let count = 0;

      switch (selectorType) {
        case 'ExclusiveClusterSetLabel':
          count = clusters.filter(cluster =>
            cluster.labels &&
            cluster.labels['cluster.open-cluster-management.io/clusterset'] === clusterSet.name
          ).length;
          break;

        case 'LabelSelector': {
          const labelSelector = clusterSet.spec?.clusterSelector?.labelSelector;

          if (!labelSelector || Object.keys(labelSelector).length === 0) {
            count = clusters.length;
          } else {
            count = clusters.filter(cluster => {
              if (!cluster.labels) return false;

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

  const total = clusters.length
  const available = clusters.filter(c => c.status === "Online").length
  const totalClusterSets = clusterSets.length
  const totalPlacements = placements.length
  const successfulPlacements = placements.filter(p => p.succeeded).length

  const statusData = useMemo(() => {
    const online = clusters.filter(c => c.status === "Online").length
    const offline = clusters.filter(c => c.status !== "Online").length
    return [
      { id: 0, value: online, label: "Online", color: "#3e8635" },
      { id: 1, value: offline, label: "Offline", color: "#c9190b" },
    ]
  }, [clusters])

  const versionData = useMemo(() => {
    const counts: Record<string, number> = {}
    clusters.forEach(c => {
      const v = c.version || "Unknown"
      counts[v] = (counts[v] || 0) + 1
    })
    return Object.entries(counts)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([version, count]) => ({ version, count }))
  }, [clusters])

  const parseMemoryToGiB = (mem: string): number => {
    const match = mem.match(/^(\d+(?:\.\d+)?)\s*(Ki|Mi|Gi|Ti|K|M|G|T|k|m|g|)?/)
    if (!match) return 0
    const value = parseFloat(match[1])
    const unit = match[2] || ""
    switch (unit) {
      case "Ki": case "K": case "k": return Math.round(value / (1024 * 1024))
      case "Mi": case "M": case "m": return Math.round(value / 1024)
      case "Gi": case "G": case "g": return Math.round(value)
      case "Ti": case "T": return Math.round(value * 1024)
      default: return Math.round(value / (1024 * 1024 * 1024))
    }
  }

  const capacityData = useMemo(() => {
    return clusters.map(c => {
      const memGi = parseMemoryToGiB(c.capacity?.memory || "0")
      return {
        name: c.name.length > 12 ? c.name.slice(0, 12) + "..." : c.name,
        cpu: parseInt(c.capacity?.cpu || "0", 10),
        memory: memGi,
      }
    })
  }, [clusters])

  const hasCapacityData = useMemo(() => {
    return clusters.length > 0 && clusters.some(c => c.capacity && (c.capacity.cpu || c.capacity.memory))
  }, [clusters])

  const platformData = useMemo(() => {
    const counts: Record<string, number> = {}
    clusters.forEach(c => {
      const claim = c.clusterClaims?.find(cl => cl.name === "platform.open-cluster-management.io")
      const platform = claim?.value || "Unknown"
      counts[platform] = (counts[platform] || 0) + 1
    })
    const colors = ["#0066cc", "#f0ab00", "#3e8635", "#c9190b", "#2b9af3", "#8b5cf6"]
    return Object.entries(counts).map(([label, value], i) => ({
      id: i,
      value,
      label,
      color: colors[i % colors.length],
    }))
  }, [clusters])

  const formatTimeAgo = (timestamp: string) => {
    const diff = Date.now() - new Date(timestamp).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 60) return `${mins}m ago`
    const hrs = Math.floor(mins / 60)
    if (hrs < 24) return `${hrs}h ago`
    return `${Math.floor(hrs / 24)}d ago`
  }

  const alerts = useMemo(() => {
    const items: { severity: "critical" | "warning"; cluster: string; message: string; timestamp: string }[] = []
    clusters.forEach(c => {
      c.conditions?.forEach(cond => {
        if (cond.status === "True") return
        const severity = cond.type === "ManagedClusterConditionAvailable" ? "critical" as const : "warning" as const
        items.push({
          severity,
          cluster: c.name,
          message: cond.message || `${cond.type}: ${cond.reason || "Unknown"}`,
          timestamp: cond.lastTransitionTime || new Date().toISOString(),
        })
      })
    })
    items.sort((a, b) => {
      if (a.severity !== b.severity) return a.severity === "critical" ? -1 : 1
      return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    })
    return items
  }, [clusters])

  const chartGradients = [
    "linear-gradient(90deg, #0066cc, #2b9af3)",
    "linear-gradient(90deg, #3e8635, #4caf50)",
    "linear-gradient(90deg, #2b9af3, #5cb3f5)",
    "linear-gradient(90deg, #f0ab00, #f5c842)",
  ]

  const chartBgTints = [
    "linear-gradient(180deg, rgba(0,102,204,0.03) 0%, transparent 60%)",
    "linear-gradient(180deg, rgba(62,134,53,0.03) 0%, transparent 60%)",
    "linear-gradient(180deg, rgba(43,154,243,0.03) 0%, transparent 60%)",
    "linear-gradient(180deg, rgba(240,171,0,0.03) 0%, transparent 60%)",
  ]

  const chartCardSx = (index: number) => ({
    p: 3,
    borderRadius: "12px",
    position: "relative",
    transition: "all 0.3s ease",
    background: chartBgTints[index % chartBgTints.length],
    "&:hover": {
      transform: "translateY(-4px)",
      boxShadow: "0 12px 24px -4px rgba(0,0,0,0.12), 0 4px 8px -2px rgba(0,0,0,0.06)",
    },
    "&::before": {
      content: '""',
      position: "absolute",
      top: 0,
      left: 0,
      right: 0,
      height: "4px",
      borderRadius: "12px 12px 0 0",
      background: chartGradients[index % chartGradients.length],
    },
  })

  const metricCardSx = (styleIndex: number) => ({
    p: 3,
    height: "100%",
    borderRadius: "12px",
    display: "flex",
    flexDirection: "column",
    position: "relative",
    overflow: "hidden",
    transition: "all 0.3s ease",
    "&:hover": {
      transform: "translateY(-4px)",
      boxShadow: "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)",
    },
    "&::before": {
      content: '""',
      position: "absolute",
      top: 0,
      left: 0,
      right: 0,
      height: "4px",
      background: cardStyles[styleIndex].gradient,
    },
  })

  return (
    <Box sx={{ p: 3, animation: "fadeInUp 0.5s ease" }}>
      <Typography
        variant="h5"
        sx={{
          mb: 3,
          fontWeight: 700,
          color: "#1a1d21",
          fontFamily: "'Red Hat Display', 'Helvetica Neue', Arial, sans-serif",
        }}
      >
        Overview
      </Typography>

      <Grid container spacing={3} sx={{ width: '100%' }}>
        {/* Clusters card */}
        <Grid size={{ xs: 12, md: 4 }}>
          <Paper sx={metricCardSx(0)}>
            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  width: 48,
                  height: 48,
                  borderRadius: "12px",
                  background: cardStyles[0].iconBg,
                  mr: 2,
                }}
              >
                <StorageIcon sx={{ color: "white", fontSize: 24 }} />
              </Box>
              <Box sx={{ display: "flex", alignItems: "flex-end" }}>
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">
                    All Clusters
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: 600 }}>
                    {loading ? "-" : total}
                  </Typography>
                </Box>
                <Box sx={{ ml: 4 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Available
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: 600, color: "#3e8635" }}>
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
                  bgcolor: "#e5e7ea",
                  borderRadius: "10px",
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
                    background: cardStyles[0].progressBg,
                    borderRadius: "10px",
                    transition: "width 0.5s ease",
                  }}
                />
              </Box>
              <Typography variant="body2" fontWeight={600} sx={{ ml: 2, minWidth: 40 }}>
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

        {/* ClusterSets card */}
        <Grid size={{ xs: 12, md: 4 }}>
          <Paper sx={metricCardSx(1)}>
            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  width: 48,
                  height: 48,
                  borderRadius: "12px",
                  background: cardStyles[1].iconBg,
                  mr: 2,
                }}
              >
                <LayersIcon sx={{ color: "white", fontSize: 24 }} />
              </Box>
              <Box>
                <Typography variant="subtitle2" color="text.secondary">
                  ManagedClusterSets
                </Typography>
                <Typography variant="h3" sx={{ fontWeight: 600 }}>
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
                    <Typography variant="body2" fontWeight={600}>
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
        <Grid size={{ xs: 12, md: 4 }}>
          <Paper sx={metricCardSx(2)}>
            <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  width: 48,
                  height: 48,
                  borderRadius: "12px",
                  background: cardStyles[2].iconBg,
                  mr: 2,
                }}
              >
                <DeviceHubIcon sx={{ color: "white", fontSize: 24 }} />
              </Box>
              <Box sx={{ display: "flex", alignItems: "flex-end" }}>
                <Box>
                  <Typography variant="subtitle2" color="text.secondary">
                    All Placements
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: 600 }}>
                    {placementsLoading ? "-" : totalPlacements}
                  </Typography>
                </Box>
                <Box sx={{ ml: 4 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Successful
                  </Typography>
                  <Typography variant="h3" sx={{ fontWeight: 600, color: "#3e8635" }}>
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
                  bgcolor: "#e5e7ea",
                  borderRadius: "10px",
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
                    background: cardStyles[2].progressBg,
                    borderRadius: "10px",
                    transition: "width 0.5s ease",
                  }}
                />
              </Box>
              <Typography variant="body2" fontWeight={600} sx={{ ml: 2, minWidth: 40 }}>
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
      </Grid>

      {/* Cluster Alerts */}
      <Paper
        sx={{
          mt: 3,
          p: 3,
          borderRadius: "12px",
          position: "relative",
          transition: "all 0.3s ease",
          "&:hover": {
            boxShadow: "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)",
          },
          "&::before": {
            content: '""',
            position: "absolute",
            top: 0,
            left: 0,
            right: 0,
            height: "4px",
            borderRadius: "12px 12px 0 0",
            background: alerts.length > 0
              ? "linear-gradient(90deg, #c9190b, #f56b5e)"
              : "linear-gradient(90deg, #3e8635, #6ec964)",
          },
        }}
      >
        <Box sx={{ display: "flex", alignItems: "center", mb: 2 }}>
          {alerts.length > 0 ? (
            <ErrorIcon sx={{ color: "#c9190b", mr: 1.5, fontSize: 28 }} />
          ) : (
            <CheckCircleIcon sx={{ color: "#3e8635", mr: 1.5, fontSize: 28 }} />
          )}
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
            Cluster Alerts
          </Typography>
          {alerts.length > 0 && (
            <Chip
              label={alerts.length}
              size="small"
              sx={{ ml: 1.5, bgcolor: "#c9190b", color: "#fff", fontWeight: 600, fontSize: 12, height: 22 }}
            />
          )}
        </Box>

        {loading ? (
          <Skeleton variant="rounded" height={60} />
        ) : alerts.length === 0 ? (
          <Box sx={{ display: "flex", alignItems: "center", py: 2, px: 1, bgcolor: "rgba(62,134,53,0.06)", borderRadius: "8px" }}>
            <CheckCircleIcon sx={{ color: "#3e8635", mr: 1.5 }} />
            <Typography variant="body2" sx={{ color: "#3e8635", fontWeight: 500 }}>
              All clusters healthy — no active alerts
            </Typography>
          </Box>
        ) : (
          <Box sx={{ maxHeight: 300, overflow: "auto" }}>
            {alerts.map((alert, i) => (
              <Box
                key={i}
                sx={{
                  display: "flex",
                  alignItems: "center",
                  gap: 1.5,
                  py: 1.5,
                  px: 1,
                  borderRadius: "8px",
                  mb: 1,
                  bgcolor: alert.severity === "critical" ? "rgba(201,25,11,0.05)" : "rgba(240,171,0,0.05)",
                  "&:last-child": { mb: 0 },
                }}
              >
                {alert.severity === "critical" ? (
                  <ErrorIcon sx={{ color: "#c9190b", fontSize: 20, flexShrink: 0 }} />
                ) : (
                  <WarningIcon sx={{ color: "#f0ab00", fontSize: 20, flexShrink: 0 }} />
                )}
                <Chip
                  label={alert.severity === "critical" ? "Critical" : "Warning"}
                  size="small"
                  sx={{
                    fontWeight: 600,
                    fontSize: 11,
                    height: 20,
                    bgcolor: alert.severity === "critical" ? "#c9190b" : "#f0ab00",
                    color: "#fff",
                    flexShrink: 0,
                  }}
                />
                <Typography variant="body2" sx={{ fontWeight: 600, flexShrink: 0 }}>
                  {alert.cluster}
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                  {alert.message}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ flexShrink: 0 }}>
                  {formatTimeAgo(alert.timestamp)}
                </Typography>
              </Box>
            ))}
          </Box>
        )}
      </Paper>

      <Typography
        variant="h5"
        sx={{
          mt: 5,
          mb: 3,
          fontWeight: 700,
          color: "#1a1d21",
          fontFamily: "'Red Hat Display', 'Helvetica Neue', Arial, sans-serif",
        }}
      >
        Metrics
      </Typography>

      <Grid container spacing={3} sx={{ width: "100%" }}>
        {/* Cluster Status Distribution */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={chartCardSx(0)}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              Cluster Status
            </Typography>
            {loading ? (
              <Skeleton variant="rounded" height={300} />
            ) : (
              <PieChart
                series={[{
                  data: statusData.map((d, i) => ({ ...d, color: `url(#statusGrad${i})` })),
                  innerRadius: 55,
                  outerRadius: 100,
                  paddingAngle: 3,
                  cornerRadius: 6,
                  highlightScope: { fade: "global", highlight: "item" },
                }]}
                height={300}
                margin={{ top: 10, bottom: 50 }}
                slotProps={{ legend: { direction: "row", position: { vertical: "bottom", horizontal: "middle" } } }}
              >
                <defs>
                  <linearGradient id="statusGrad0" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#3e8635" />
                    <stop offset="100%" stopColor="#6ec964" />
                  </linearGradient>
                  <linearGradient id="statusGrad1" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#c9190b" />
                    <stop offset="100%" stopColor="#f56b5e" />
                  </linearGradient>
                  <filter id="pieShadow">
                    <feDropShadow dx="0" dy="2" stdDeviation="3" floodOpacity="0.15" />
                  </filter>
                </defs>
              </PieChart>
            )}
          </Paper>
        </Grid>

        {/* Kubernetes Version Distribution */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={chartCardSx(1)}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              Kubernetes Versions
            </Typography>
            {loading ? (
              <Skeleton variant="rounded" height={300} />
            ) : (
              <BarChart
                dataset={versionData}
                xAxis={[{ scaleType: "band", dataKey: "version" }]}
                series={[{ dataKey: "count", label: "Clusters", color: "url(#versionBarGrad)" }]}
                height={300}
                margin={{ top: 10, bottom: 30 }}
                borderRadius={6}
                slotProps={{ legend: { hidden: true } }}
              >
                <defs>
                  <linearGradient id="versionBarGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#3e8635" />
                    <stop offset="50%" stopColor="#4caf50" />
                    <stop offset="100%" stopColor="#81c784" stopOpacity={0.7} />
                  </linearGradient>
                </defs>
              </BarChart>
            )}
          </Paper>
        </Grid>

        {/* CPU Capacity */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={chartCardSx(2)}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              CPU Capacity (cores)
            </Typography>
            {loading ? (
              <Skeleton variant="rounded" height={300} />
            ) : !hasCapacityData ? (
              <Box sx={{ py: 4, px: 2, textAlign: "center", bgcolor: "rgba(201,25,11,0.04)", borderRadius: "8px" }}>
                <ErrorIcon sx={{ color: "#c9190b", fontSize: 40, mb: 1 }} />
                <Typography variant="body2" sx={{ fontWeight: 600, color: "#c9190b", mb: 1 }}>
                  Metrics data unavailable
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  Capacity metrics could not be retrieved. Please ensure the metrics-server is installed:
                </Typography>
                <Box
                  component="code"
                  sx={{
                    display: "block",
                    p: 1.5,
                    bgcolor: "#1a1d21",
                    color: "#e5e7ea",
                    borderRadius: "6px",
                    fontSize: 12,
                    fontFamily: "monospace",
                    overflowX: "auto",
                    textAlign: "left",
                    whiteSpace: "pre-wrap",
                    wordBreak: "break-all",
                  }}
                >
                  kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
                </Box>
              </Box>
            ) : (
              <BarChart
                dataset={capacityData}
                xAxis={[{ scaleType: "band", dataKey: "name", tickLabelStyle: { angle: -35, textAnchor: "end", fontSize: 12 } }]}
                series={[{ dataKey: "cpu", label: "CPU (cores)", color: "url(#cpuGrad)" }]}
                height={300}
                margin={{ top: 10, bottom: 70 }}
                borderRadius={6}
                slotProps={{ legend: { hidden: true } }}
              >
                <defs>
                  <linearGradient id="cpuGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#0066cc" />
                    <stop offset="100%" stopColor="#5cb3f5" stopOpacity={0.6} />
                  </linearGradient>
                </defs>
              </BarChart>
            )}
          </Paper>
        </Grid>

        {/* Memory Capacity */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={chartCardSx(2)}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              Memory Capacity (Gi)
            </Typography>
            {loading ? (
              <Skeleton variant="rounded" height={300} />
            ) : !hasCapacityData ? (
              <Box sx={{ py: 4, px: 2, textAlign: "center", bgcolor: "rgba(201,25,11,0.04)", borderRadius: "8px" }}>
                <ErrorIcon sx={{ color: "#c9190b", fontSize: 40, mb: 1 }} />
                <Typography variant="body2" sx={{ fontWeight: 600, color: "#c9190b", mb: 1 }}>
                  Metrics data unavailable
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  Capacity metrics could not be retrieved. Please ensure the metrics-server is installed:
                </Typography>
                <Box
                  component="code"
                  sx={{
                    display: "block",
                    p: 1.5,
                    bgcolor: "#1a1d21",
                    color: "#e5e7ea",
                    borderRadius: "6px",
                    fontSize: 12,
                    fontFamily: "monospace",
                    overflowX: "auto",
                    textAlign: "left",
                    whiteSpace: "pre-wrap",
                    wordBreak: "break-all",
                  }}
                >
                  kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
                </Box>
              </Box>
            ) : (
              <BarChart
                dataset={capacityData}
                xAxis={[{ scaleType: "band", dataKey: "name", tickLabelStyle: { angle: -35, textAnchor: "end", fontSize: 12 } }]}
                yAxis={[{ valueFormatter: (v: number | null) => v == null ? "" : `${v}` }]}
                series={[{ dataKey: "memory", label: "Memory (Gi)", color: "url(#memGrad)", valueFormatter: (v: number | null) => v == null ? "" : `${v} Gi` }]}
                height={300}
                margin={{ top: 10, bottom: 70 }}
                borderRadius={6}
                slotProps={{ legend: { hidden: true } }}
              >
                <defs>
                  <linearGradient id="memGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#2b9af3" />
                    <stop offset="100%" stopColor="#a0d4ff" stopOpacity={0.6} />
                  </linearGradient>
                </defs>
              </BarChart>
            )}
          </Paper>
        </Grid>

        {/* Platform Distribution */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={chartCardSx(3)}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
              Platform Distribution
            </Typography>
            {loading ? (
              <Skeleton variant="rounded" height={300} />
            ) : (
              <PieChart
                series={[{
                  data: platformData.map((d, i) => ({ ...d, color: `url(#platformGrad${i})` })),
                  innerRadius: 55,
                  outerRadius: 100,
                  paddingAngle: 3,
                  cornerRadius: 6,
                  highlightScope: { fade: "global", highlight: "item" },
                }]}
                height={300}
                margin={{ top: 10, bottom: 50 }}
                slotProps={{ legend: { direction: "row", position: { vertical: "bottom", horizontal: "middle" } } }}
              >
                <defs>
                  <linearGradient id="platformGrad0" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#0066cc" />
                    <stop offset="100%" stopColor="#5cb3f5" />
                  </linearGradient>
                  <linearGradient id="platformGrad1" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#f0ab00" />
                    <stop offset="100%" stopColor="#f5d76e" />
                  </linearGradient>
                  <linearGradient id="platformGrad2" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#3e8635" />
                    <stop offset="100%" stopColor="#6ec964" />
                  </linearGradient>
                  <linearGradient id="platformGrad3" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#c9190b" />
                    <stop offset="100%" stopColor="#f56b5e" />
                  </linearGradient>
                  <linearGradient id="platformGrad4" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#2b9af3" />
                    <stop offset="100%" stopColor="#a0d4ff" />
                  </linearGradient>
                  <linearGradient id="platformGrad5" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#8b5cf6" />
                    <stop offset="100%" stopColor="#c4b5fd" />
                  </linearGradient>
                </defs>
              </PieChart>
            )}
          </Paper>
        </Grid>
      </Grid>
    </Box>
  )
}
