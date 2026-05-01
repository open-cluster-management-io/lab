import { useEffect, useState, useMemo } from "react"
import { Box, Typography, Paper, Chip, Skeleton } from "@mui/material"
import Timeline from "@mui/lab/Timeline"
import TimelineItem from "@mui/lab/TimelineItem"
import TimelineSeparator from "@mui/lab/TimelineSeparator"
import TimelineConnector from "@mui/lab/TimelineConnector"
import TimelineContent from "@mui/lab/TimelineContent"
import TimelineOppositeContent from "@mui/lab/TimelineOppositeContent"
import TimelineDot from "@mui/lab/TimelineDot"
import {
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Layers as LayersIcon,
  DeviceHub as DeviceHubIcon,
  Storage as StorageIcon,
} from "@mui/icons-material"
import { fetchClusters } from "../api/clusterService"
import { fetchClusterSets } from "../api/clusterSetService"
import { fetchPlacements } from "../api/placementService"
import type { Cluster } from "../api/clusterService"
import type { ClusterSet } from "../api/clusterSetService"
import type { Placement } from "../api/placementService"

interface TimelineEvent {
  timestamp: string
  title: string
  detail: string
  category: "cluster" | "clusterset" | "placement"
  severity: "success" | "error" | "info" | "warning"
}

const severityConfig = {
  success: { color: "#3e8635" as const, gradient: "linear-gradient(135deg, #3e8635, #6ec964)" },
  error: { color: "#c9190b" as const, gradient: "linear-gradient(135deg, #c9190b, #f56b5e)" },
  warning: { color: "#f0ab00" as const, gradient: "linear-gradient(135deg, #f0ab00, #f5d76e)" },
  info: { color: "#0066cc" as const, gradient: "linear-gradient(135deg, #0066cc, #5cb3f5)" },
}

const categoryIcons = {
  cluster: <StorageIcon sx={{ fontSize: 18, color: "#fff" }} />,
  clusterset: <LayersIcon sx={{ fontSize: 18, color: "#fff" }} />,
  placement: <DeviceHubIcon sx={{ fontSize: 18, color: "#fff" }} />,
}

function formatTimestamp(ts: string) {
  const d = new Date(ts)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffMins = Math.floor(diffMs / 60000)

  let relative: string
  if (diffMins < 1) relative = "just now"
  else if (diffMins < 60) relative = `${diffMins}m ago`
  else if (diffMins < 1440) relative = `${Math.floor(diffMins / 60)}h ago`
  else relative = `${Math.floor(diffMins / 1440)}d ago`

  const absolute = d.toLocaleDateString(undefined, { month: "short", day: "numeric" }) +
    " " + d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })

  return { relative, absolute }
}

function deriveEvents(clusters: Cluster[], clusterSets: ClusterSet[], placements: Placement[]): TimelineEvent[] {
  const events: TimelineEvent[] = []

  clusters.forEach(c => {
    if (c.creationTimestamp) {
      events.push({
        timestamp: c.creationTimestamp,
        title: `Cluster "${c.name}" joined`,
        detail: `Version ${c.version || "unknown"} • Hub accepted: ${c.hubAccepted ? "Yes" : "No"}`,
        category: "cluster",
        severity: "info",
      })
    }
    c.conditions?.forEach(cond => {
      if (!cond.lastTransitionTime) return
      const isHealthy = cond.status === "True"
      if (cond.type === "ManagedClusterConditionAvailable") {
        events.push({
          timestamp: cond.lastTransitionTime,
          title: `${c.name} went ${isHealthy ? "Online" : "Offline"}`,
          detail: cond.message || cond.reason || "",
          category: "cluster",
          severity: isHealthy ? "success" : "error",
        })
      } else if (cond.type === "ManagedClusterJoined") {
        events.push({
          timestamp: cond.lastTransitionTime,
          title: `${c.name} ${isHealthy ? "joined" : "left"} the hub`,
          detail: cond.message || cond.reason || "",
          category: "cluster",
          severity: isHealthy ? "success" : "warning",
        })
      }
    })
  })

  clusterSets.forEach(cs => {
    if (cs.creationTimestamp) {
      events.push({
        timestamp: cs.creationTimestamp,
        title: `ClusterSet "${cs.name}" created`,
        detail: `Selector: ${cs.spec?.clusterSelector?.selectorType || "default"}`,
        category: "clusterset",
        severity: "info",
      })
    }
  })

  placements.forEach(p => {
    if (p.creationTimestamp) {
      events.push({
        timestamp: p.creationTimestamp,
        title: `Placement "${p.name}" created`,
        detail: `Namespace: ${p.namespace} • Selected clusters: ${p.numberOfSelectedClusters}`,
        category: "placement",
        severity: "info",
      })
    }
    p.conditions?.forEach(cond => {
      if (!cond.lastTransitionTime || cond.type !== "PlacementSatisfied") return
      events.push({
        timestamp: cond.lastTransitionTime,
        title: `${p.name} ${cond.status === "True" ? "satisfied" : "unsatisfied"}`,
        detail: cond.message || cond.reason || "",
        category: "placement",
        severity: cond.status === "True" ? "success" : "warning",
      })
    })
  })

  events.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
  return events
}

export default function ActivityTimelinePage() {
  const [clusters, setClusters] = useState<Cluster[]>([])
  const [clusterSets, setClusterSets] = useState<ClusterSet[]>([])
  const [placements, setPlacements] = useState<Placement[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      const [c, cs, p] = await Promise.all([fetchClusters(), fetchClusterSets(), fetchPlacements()])
      setClusters(c)
      setClusterSets(cs)
      setPlacements(p)
      setLoading(false)
    }
    load()
  }, [])

  const events = useMemo(() => deriveEvents(clusters, clusterSets, placements), [clusters, clusterSets, placements])

  const [filter, setFilter] = useState<"all" | "cluster" | "clusterset" | "placement">("all")
  const filtered = filter === "all" ? events : events.filter(e => e.category === filter)

  return (
    <Box sx={{ p: 3, animation: "fadeInUp 0.5s ease" }}>
      <Typography
        variant="h5"
        sx={{
          mb: 1,
          fontWeight: 700,
          color: "#1a1d21",
          fontFamily: "'Red Hat Display', 'Helvetica Neue', Arial, sans-serif",
        }}
      >
        Hub Cluster Activity
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Timeline of cluster, clusterset, and placement events
      </Typography>

      <Box sx={{ display: "flex", gap: 1, mb: 3, flexWrap: "wrap" }}>
        {(["all", "cluster", "clusterset", "placement"] as const).map(f => (
          <Chip
            key={f}
            label={f === "all" ? "All Events" : f === "clusterset" ? "ClusterSets" : f.charAt(0).toUpperCase() + f.slice(1) + "s"}
            onClick={() => setFilter(f)}
            variant={filter === f ? "filled" : "outlined"}
            sx={{
              fontWeight: 600,
              ...(filter === f
                ? { bgcolor: "#0066cc", color: "#fff", "&:hover": { bgcolor: "#0055aa" } }
                : { borderColor: "#d2d2d2", "&:hover": { bgcolor: "#f1f2f3" } }),
            }}
          />
        ))}
        <Chip
          label={`${filtered.length} events`}
          size="small"
          sx={{ ml: "auto", bgcolor: "#f1f2f3", fontWeight: 500 }}
        />
      </Box>

      {loading ? (
        <Paper sx={{ p: 3, borderRadius: "12px" }}>
          {[...Array(5)].map((_, i) => (
            <Box key={i} sx={{ display: "flex", gap: 2, mb: 3 }}>
              <Skeleton variant="circular" width={40} height={40} />
              <Box sx={{ flex: 1 }}>
                <Skeleton variant="text" width="40%" />
                <Skeleton variant="text" width="70%" />
              </Box>
            </Box>
          ))}
        </Paper>
      ) : filtered.length === 0 ? (
        <Paper sx={{ p: 4, borderRadius: "12px", textAlign: "center" }}>
          <Typography color="text.secondary">No events found</Typography>
        </Paper>
      ) : (
        <Paper
          sx={{
            borderRadius: "12px",
            position: "relative",
            "&::before": {
              content: '""',
              position: "absolute",
              top: 0,
              left: 0,
              right: 0,
              height: "4px",
              borderRadius: "12px 12px 0 0",
              background: "linear-gradient(90deg, #0066cc, #2b9af3, #3e8635)",
            },
          }}
        >
          <Timeline position="alternate-reverse" sx={{ py: 2 }}>
            {filtered.map((event, i) => {
              const { relative, absolute } = formatTimestamp(event.timestamp)
              const config = severityConfig[event.severity]
              const icon = event.severity === "success"
                ? <CheckCircleIcon sx={{ fontSize: 18, color: "#fff" }} />
                : event.severity === "error"
                  ? <ErrorIcon sx={{ fontSize: 18, color: "#fff" }} />
                  : categoryIcons[event.category]
              return (
                <TimelineItem key={i}>
                  <TimelineOppositeContent sx={{ flex: 0.3, pt: 2 }}>
                    <Typography variant="caption" sx={{ fontWeight: 600, color: "text.secondary" }}>
                      {relative}
                    </Typography>
                    <Typography variant="caption" display="block" sx={{ color: "text.disabled", fontSize: 11 }}>
                      {absolute}
                    </Typography>
                  </TimelineOppositeContent>
                  <TimelineSeparator>
                    <TimelineDot
                      sx={{
                        background: config.gradient,
                        boxShadow: `0 0 8px ${config.color}40`,
                        p: 1,
                      }}
                    >
                      {icon}
                    </TimelineDot>
                    {i < filtered.length - 1 && (
                      <TimelineConnector sx={{ bgcolor: "#e5e7ea" }} />
                    )}
                  </TimelineSeparator>
                  <TimelineContent sx={{ pt: 1.5, pb: 3 }}>
                    <Paper
                      elevation={0}
                      sx={{
                        p: 2,
                        borderRadius: "10px",
                        bgcolor: `${config.color}08`,
                        border: `1px solid ${config.color}20`,
                        transition: "all 0.2s ease",
                        "&:hover": {
                          bgcolor: `${config.color}12`,
                          transform: "translateY(-1px)",
                          boxShadow: `0 4px 12px ${config.color}15`,
                        },
                      }}
                    >
                      <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 0.5 }}>
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>
                          {event.title}
                        </Typography>
                        <Chip
                          label={event.category === "clusterset" ? "ClusterSet" : event.category.charAt(0).toUpperCase() + event.category.slice(1)}
                          size="small"
                          sx={{
                            height: 18,
                            fontSize: 10,
                            fontWeight: 600,
                            bgcolor: `${config.color}18`,
                            color: config.color,
                          }}
                        />
                      </Box>
                      <Typography variant="caption" color="text.secondary">
                        {event.detail}
                      </Typography>
                    </Paper>
                  </TimelineContent>
                </TimelineItem>
              )
            })}
          </Timeline>
        </Paper>
      )}
    </Box>
  )
}
