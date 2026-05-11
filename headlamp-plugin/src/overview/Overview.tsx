import { Box, Chip, Typography } from '@mui/material';
import {
  Link,
  SectionBox,
  SimpleTable,
  PercentageCircle,
} from '@kinvolk/headlamp-plugin/lib/components/common';
import { ManagedCluster, Placement, ManifestWork, ClusterManagementAddOn, ManagedClusterAddOn, ClusterManager } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { useOCMContext, getConditionStatus } from '../helpers';


export default function Overview() {
  const ctx = useOCMContext();
  const [clusters] = ManagedCluster.useList();
  const [placements] = Placement.useList();
  const [works] = ManifestWork.useList();
  const [addons] = ClusterManagementAddOn.useList();
  const [clusterAddons] = ManagedClusterAddOn.useList();
  const [clusterManagers] = ClusterManager.useList();

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="Fleet Overview">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'managed') {
    return (
      <SectionBox title="Fleet Overview">
        <p>This is a managed cluster. Switch to the hub cluster to see fleet overview.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="Fleet Overview">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  const total = clusters?.length ?? 0;

  let availableCount = 0;
  let unavailableCount = 0;
  let joinedCount = 0;
  let notJoinedCount = 0;

  clusters?.forEach((cluster: InstanceType<typeof ManagedCluster>) => {
    const available = getConditionStatus(
      cluster.jsonData.status?.conditions,
      'ManagedClusterConditionAvailable'
    );
    const joined = getConditionStatus(
      cluster.jsonData.status?.conditions,
      'ManagedClusterJoined'
    );

    if (available?.status === 'True') availableCount++;
    else unavailableCount++;

    if (joined?.status === 'True') joinedCount++;
    else notJoinedCount++;
  });

  return (
    <>
      <SectionBox title="Fleet Overview">
        <Box sx={{ display: 'flex', gap: 4, justifyContent: 'space-around', py: 2 }}>
          <Box sx={{ textAlign: 'center' }}>
            <Typography variant="h3">{total}</Typography>
            <Typography variant="body2" color="text.secondary">
              Total Clusters
            </Typography>
          </Box>
          <PercentageCircle
            title="Availability"
            size={140}
            total={total}
            data={[
              { name: 'Available', value: availableCount, fill: '#1b7a3d' },
              { name: 'Unavailable', value: unavailableCount, fill: '#c5221f' },
            ]}
          />
          <PercentageCircle
            title="Join Status"
            size={140}
            total={total}
            data={[
              { name: 'Joined', value: joinedCount, fill: '#1b7a3d' },
              { name: 'Not Joined', value: notJoinedCount, fill: '#c5221f' },
            ]}
          />
        </Box>
      </SectionBox>

      <SectionBox title="Managed Clusters">
        <SimpleTable
          data={clusters}
          columns={[
            {
              label: 'Name',
              getter: (cluster: InstanceType<typeof ManagedCluster>) => (
                <Link routeName="ocm-cluster-detail" params={{ name: cluster.metadata.name }}>
                  {cluster.metadata.name}
                </Link>
              ),
              sort: true,
            },
            {
              label: 'Role',
              getter: (cluster: InstanceType<typeof ManagedCluster>) =>
                cluster.jsonData.metadata?.labels?.['local-cluster'] === 'true'
                  ? <Chip label="Hub" size="small" sx={{ backgroundColor: '#7b1fa2', color: '#fff' }} />
                  : <Chip label="Managed" size="small" />,
            },
            {
              label: 'Status',
              getter: (cluster: InstanceType<typeof ManagedCluster>) => {
                const available = getConditionStatus(
                  cluster.jsonData.status?.conditions,
                  'ManagedClusterConditionAvailable'
                );
                return (
                  <StatusLabel status={conditionToStatus(available?.status)}>
                    {available?.status === 'True'
                      ? 'Available'
                      : available?.status === 'False'
                        ? 'Unavailable'
                        : 'Unknown'}
                  </StatusLabel>
                );
              },
            },
            {
              label: 'Joined',
              getter: (cluster: InstanceType<typeof ManagedCluster>) => {
                const joined = getConditionStatus(
                  cluster.jsonData.status?.conditions,
                  'ManagedClusterJoined'
                );
                return (
                  <StatusLabel status={conditionToStatus(joined?.status)}>
                    {joined?.status === 'True'
                      ? 'Joined'
                      : joined?.status === 'False'
                        ? 'Not Joined'
                        : 'Unknown'}
                  </StatusLabel>
                );
              },
            },
            {
              label: 'K8s Version',
              getter: (cluster: InstanceType<typeof ManagedCluster>) =>
                cluster.kubernetesVersion || '-',
            },
            {
              label: 'Age',
              getter: (cluster: InstanceType<typeof ManagedCluster>) =>
                cluster.metadata.creationTimestamp ?? '-',
              sort: true,
            },
          ]}
        />
      </SectionBox>

      <SectionBox title="Placements">
        {placements && placements.length > 0 ? (
          <SimpleTable
            data={placements}
            columns={[
              {
                label: 'Name',
                getter: (p: InstanceType<typeof Placement>) => (
                  <Link
                    routeName="ocm-placement-detail"
                    params={{ namespace: p.metadata.namespace, name: p.metadata.name }}
                  >
                    {p.metadata.name}
                  </Link>
                ),
              },
              {
                label: 'Namespace',
                getter: (p: InstanceType<typeof Placement>) => p.metadata.namespace,
              },
              {
                label: 'Selected',
                getter: (p: InstanceType<typeof Placement>) =>
                  `${p.numberOfSelectedClusters}` +
                  (p.jsonData.spec?.numberOfClusters
                    ? ` / ${p.jsonData.spec.numberOfClusters}`
                    : ''),
              },
              {
                label: 'Status',
                getter: (p: InstanceType<typeof Placement>) => {
                  const satisfied = getConditionStatus(
                    p.jsonData.status?.conditions,
                    'PlacementSatisfied'
                  );
                  return (
                    <StatusLabel status={conditionToStatus(satisfied?.status)}>
                      {satisfied?.status === 'True'
                        ? 'Satisfied'
                        : satisfied?.status === 'False'
                          ? 'Not Satisfied'
                          : 'Unknown'}
                    </StatusLabel>
                  );
                },
              },
            ]}
          />
        ) : (
          <p>No placements found.</p>
        )}
      </SectionBox>

      <SectionBox title="ManifestWorks">
        {works && works.length > 0 ? (
          <SimpleTable
            data={works}
            columns={[
              {
                label: 'Name',
                getter: (w: InstanceType<typeof ManifestWork>) => (
                  <Link
                    routeName="ocm-manifestwork-detail"
                    params={{ namespace: w.metadata.namespace, name: w.metadata.name }}
                  >
                    {w.metadata.name}
                  </Link>
                ),
              },
              {
                label: 'Target Cluster',
                getter: (w: InstanceType<typeof ManifestWork>) => (
                  <Link routeName="ocm-cluster-detail" params={{ name: w.metadata.namespace }}>
                    {w.metadata.namespace}
                  </Link>
                ),
              },
              {
                label: 'Source',
                getter: (w: InstanceType<typeof ManifestWork>) => {
                  const replicaSet = w.jsonData.metadata?.labels?.['work.open-cluster-management.io/manifestworkreplicaset'];
                  return replicaSet
                    ? <StatusLabel status="success">ReplicaSet</StatusLabel>
                    : <StatusLabel status="unknown">Manual</StatusLabel>;
                },
              },
              {
                label: 'Applied',
                getter: (w: InstanceType<typeof ManifestWork>) => {
                  const applied = getConditionStatus(w.jsonData.status?.conditions, 'Applied');
                  return (
                    <StatusLabel status={conditionToStatus(applied?.status)}>
                      {applied?.status === 'True'
                        ? 'Applied'
                        : applied?.status === 'False'
                          ? 'Not Applied'
                          : 'Unknown'}
                    </StatusLabel>
                  );
                },
              },
              {
                label: 'Available',
                getter: (w: InstanceType<typeof ManifestWork>) => {
                  const available = getConditionStatus(w.jsonData.status?.conditions, 'Available');
                  return (
                    <StatusLabel status={conditionToStatus(available?.status)}>
                      {available?.status === 'True'
                        ? 'Available'
                        : available?.status === 'False'
                          ? 'Unavailable'
                          : 'Unknown'}
                    </StatusLabel>
                  );
                },
              },
            ]}
          />
        ) : (
          <p>No manifest works found.</p>
        )}
      </SectionBox>

      <SectionBox title="Add-ons">
        {addons && addons.length > 0 ? (
          <SimpleTable
            data={addons}
            columns={[
              {
                label: 'Name',
                getter: (a: InstanceType<typeof ClusterManagementAddOn>) => (
                  <Link routeName="ocm-addon-detail" params={{ name: a.metadata.name }}>
                    {a.displayName}
                  </Link>
                ),
              },
              {
                label: 'Install Strategy',
                getter: (a: InstanceType<typeof ClusterManagementAddOn>) => a.installStrategy,
              },
              {
                label: 'Clusters',
                getter: (a: InstanceType<typeof ClusterManagementAddOn>) => {
                  const cas = clusterAddons?.filter(
                    (ca: InstanceType<typeof ManagedClusterAddOn>) =>
                      ca.metadata.name === a.metadata.name
                  ) ?? [];
                  const available = cas.filter(
                    (ca: InstanceType<typeof ManagedClusterAddOn>) => {
                      const cond = getConditionStatus(ca.jsonData.status?.conditions, 'Available');
                      return cond?.status === 'True';
                    }
                  ).length;
                  return `${available} / ${cas.length} available`;
                },
              },
            ]}
          />
        ) : (
          <p>No add-ons found.</p>
        )}
      </SectionBox>

      {(() => {
        const cm = clusterManagers?.[0];
        const gates = cm?.featureGates ?? [];
        if (gates.length === 0) return null;
        return (
          <SectionBox title="Feature Gates">
            <SimpleTable
              data={gates}
              columns={[
                { label: 'Component', getter: (g: { component: string }) => g.component },
                { label: 'Feature', getter: (g: { feature: string }) => g.feature },
                {
                  label: 'Mode',
                  getter: (g: { mode: string }) => (
                    <StatusLabel status={g.mode === 'Enable' ? 'success' : 'unknown'}>
                      {g.mode}
                    </StatusLabel>
                  ),
                },
              ]}
            />
          </SectionBox>
        );
      })()}
    </>
  );
}
