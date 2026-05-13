import { useParams } from 'react-router-dom';
import {
  MainInfoSection,
  SectionBox,
  SimpleTable,
  ConditionsTable,
} from '@kinvolk/headlamp-plugin/lib/components/common';
import { ManagedCluster } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { getConditionStatus, useOCMContext } from '../helpers';

export default function ClusterDetail() {
  const { name } = useParams<{ name: string }>();
  const ctx = useOCMContext();
  const [cluster, error] = ManagedCluster.useGet(name!);

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="Managed Cluster">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'managed') {
    return (
      <SectionBox title="Managed Cluster">
        <p>This is a managed cluster. Switch to the hub cluster to see fleet views.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="Managed Cluster">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  if (error) {
    return (
      <SectionBox title="Managed Cluster">
        <p>Error loading cluster: {error.toString()}</p>
      </SectionBox>
    );
  }

  if (!cluster) {
    return (
      <SectionBox title="Managed Cluster">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  const available = getConditionStatus(
    cluster.jsonData.status?.conditions,
    'ManagedClusterConditionAvailable'
  );
  const joined = getConditionStatus(
    cluster.jsonData.status?.conditions,
    'ManagedClusterJoined'
  );

  return (
    <>
      <MainInfoSection
        resource={cluster}
        title="Managed Cluster"
        extraInfo={[
          {
            name: 'Status',
            value: (
              <StatusLabel status={conditionToStatus(available?.status)}>
                {available?.status === 'True'
                  ? 'Available'
                  : available?.status === 'False'
                    ? 'Unavailable'
                    : 'Unknown'}
              </StatusLabel>
            ),
          },
          {
            name: 'Joined',
            value: (
              <StatusLabel status={conditionToStatus(joined?.status)}>
                {joined?.status === 'True'
                  ? 'Joined'
                  : joined?.status === 'False'
                    ? 'Not Joined'
                    : 'Unknown'}
              </StatusLabel>
            ),
          },
          {
            name: 'Kubernetes Version',
            value: cluster.kubernetesVersion || '-',
          },
          {
            name: 'Hub Accepts Client',
            value: cluster.jsonData.spec?.hubAcceptsClient === true
              ? 'Yes'
              : cluster.jsonData.spec?.hubAcceptsClient === false
                ? 'No'
                : '-',
          },
          {
            name: 'Lease Duration',
            value: cluster.jsonData.spec?.leaseDurationSeconds
              ? `${cluster.jsonData.spec.leaseDurationSeconds}s`
              : '-',
          },
        ]}
      />

      <SectionBox title="Capacity & Allocatable">
        <SimpleTable
          data={Object.keys(cluster.jsonData.status?.capacity ?? {}).map(key => ({
            resource: key,
            capacity: cluster.jsonData.status?.capacity?.[key] ?? '-',
            allocatable: cluster.jsonData.status?.allocatable?.[key] ?? '-',
          }))}
          columns={[
            { label: 'Resource', getter: (row: { resource: string }) => row.resource },
            { label: 'Capacity', getter: (row: { capacity: string }) => row.capacity },
            { label: 'Allocatable', getter: (row: { allocatable: string }) => row.allocatable },
          ]}
        />
      </SectionBox>

      {cluster.clusterClaims.length > 0 && (
        <SectionBox title="Cluster Claims">
          <SimpleTable
            data={cluster.clusterClaims}
            columns={[
              { label: 'Name', getter: (row: { name: string }) => row.name },
              { label: 'Value', getter: (row: { value: string }) => row.value },
            ]}
          />
        </SectionBox>
      )}

      {cluster.jsonData.spec?.taints?.length > 0 && (
        <SectionBox title="Taints">
          <SimpleTable
            data={cluster.jsonData.spec.taints}
            columns={[
              { label: 'Key', getter: (row: { key: string }) => row.key },
              { label: 'Value', getter: (row: { value: string }) => row.value ?? '-' },
              { label: 'Effect', getter: (row: { effect: string }) => row.effect },
            ]}
          />
        </SectionBox>
      )}

      <SectionBox title="Conditions">
        <ConditionsTable resource={cluster} />
      </SectionBox>
    </>
  );
}
