import { Link, SectionBox, SimpleTable } from '@kinvolk/headlamp-plugin/lib/components/common';
import { ManagedCluster } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { getConditionStatus, useOCMContext } from '../helpers';

export default function ClusterList() {
  const ctx = useOCMContext();
  const [clusters, error] = ManagedCluster.useList();

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="Managed Clusters">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'managed') {
    return (
      <SectionBox title="Managed Clusters">
        <p>This is a managed cluster. Switch to the hub cluster to see fleet views.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="Managed Clusters">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  if (error) {
    return (
      <SectionBox title="Managed Clusters">
        <p>Error loading clusters: {error.toString()}</p>
      </SectionBox>
    );
  }

  return (
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
            label: 'Status',
            getter: (cluster: InstanceType<typeof ManagedCluster>) => {
              const available = getConditionStatus(
                cluster.jsonData.status?.conditions,
                'ManagedClusterConditionAvailable'
              );
              const statusType = conditionToStatus(available?.status);
              const label =
                available?.status === 'True'
                  ? 'Available'
                  : available?.status === 'False'
                    ? 'Unavailable'
                    : 'Unknown';
              return <StatusLabel status={statusType}>{label}</StatusLabel>;
            },
          },
          {
            label: 'Joined',
            getter: (cluster: InstanceType<typeof ManagedCluster>) => {
              const joined = getConditionStatus(
                cluster.jsonData.status?.conditions,
                'ManagedClusterJoined'
              );
              const statusType = conditionToStatus(joined?.status);
              const label =
                joined?.status === 'True'
                  ? 'Joined'
                  : joined?.status === 'False'
                    ? 'Not Joined'
                    : 'Unknown';
              return <StatusLabel status={statusType}>{label}</StatusLabel>;
            },
          },
          {
            label: 'K8s Version',
            getter: (cluster: InstanceType<typeof ManagedCluster>) =>
              cluster.kubernetesVersion || '-',
          },
          {
            label: 'CPU',
            getter: (cluster: InstanceType<typeof ManagedCluster>) =>
              cluster.cpuCapacity || '-',
          },
          {
            label: 'Memory',
            getter: (cluster: InstanceType<typeof ManagedCluster>) =>
              cluster.memoryCapacity || '-',
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
  );
}
