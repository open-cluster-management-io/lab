import { K8s } from '@kinvolk/headlamp-plugin/lib';
import {
  SectionBox,
  SimpleTable,
  ConditionsTable,
} from '@kinvolk/headlamp-plugin/lib/components/common';
import { Klusterlet, ClusterClaim, AppliedManifestWork } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { useOCMContext, getConditionStatus } from '../helpers';

export default function ManagedOverview() {
  const ctx = useOCMContext();
  const [klusterlets] = Klusterlet.useList();
  const [claims] = ClusterClaim.useList();
  const [appliedWorks] = AppliedManifestWork.useList();
  const [pods] = K8s.ResourceClasses.Pod.useList({ namespace: 'open-cluster-management-agent' });

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="This Cluster">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="This Cluster">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'hub') {
    return (
      <SectionBox title="This Cluster">
        <p>This is an OCM hub cluster. Use the Clusters view to manage your fleet.</p>
      </SectionBox>
    );
  }

  const klusterlet = klusterlets?.[0];
  const available = klusterlet
    ? getConditionStatus(klusterlet.jsonData.status?.conditions, 'Available')
    : null;

  return (
    <>
      <SectionBox title="Klusterlet">
        {klusterlet ? (
          <>
            <SimpleTable
              data={[
                { label: 'Cluster Name', value: klusterlet.clusterName || '-' },
                { label: 'Deploy Mode', value: klusterlet.deployMode },
                { label: 'Agent Namespace', value: klusterlet.registrationNamespace },
                {
                  label: 'Status',
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
              ]}
              columns={[
                { label: 'Field', getter: (row: { label: string }) => row.label },
                { label: 'Value', getter: (row: { value: any }) => row.value },
              ]}
            />
            <ConditionsTable resource={klusterlet} />
          </>
        ) : (
          <p>Loading klusterlet...</p>
        )}
      </SectionBox>

      <SectionBox title="Cluster Claims">
        {claims && claims.length > 0 ? (
          <SimpleTable
            data={claims}
            columns={[
              {
                label: 'Name',
                getter: (claim: InstanceType<typeof ClusterClaim>) => claim.metadata.name,
              },
              {
                label: 'Value',
                getter: (claim: InstanceType<typeof ClusterClaim>) => claim.claimValue || '-',
              },
            ]}
          />
        ) : (
          <p>No cluster claims found.</p>
        )}
      </SectionBox>

      <SectionBox title="Applied ManifestWorks">
        {appliedWorks && appliedWorks.length > 0 ? (
          <SimpleTable
            data={appliedWorks}
            columns={[
              {
                label: 'Name',
                getter: (work: InstanceType<typeof AppliedManifestWork>) => work.metadata.name,
              },
              {
                label: 'Resources',
                getter: (work: InstanceType<typeof AppliedManifestWork>) =>
                  work.appliedResources.length.toString(),
              },
              {
                label: 'Age',
                getter: (work: InstanceType<typeof AppliedManifestWork>) =>
                  work.metadata.creationTimestamp ?? '-',
              },
            ]}
          />
        ) : (
          <p>No manifest works applied to this cluster.</p>
        )}
      </SectionBox>

      <SectionBox title="Agent Pods">
        {pods && pods.length > 0 ? (
          <SimpleTable
            data={pods}
            columns={[
              {
                label: 'Name',
                getter: (pod: any) => pod.metadata.name,
              },
              {
                label: 'Status',
                getter: (pod: any) => {
                  const phase = pod.jsonData.status?.phase ?? 'Unknown';
                  const statusType = phase === 'Running' ? 'success' : phase === 'Pending' ? 'warning' : 'error';
                  return <StatusLabel status={statusType}>{phase}</StatusLabel>;
                },
              },
              {
                label: 'Restarts',
                getter: (pod: any) => {
                  const containers = pod.jsonData.status?.containerStatuses ?? [];
                  return containers.reduce((sum: number, c: any) => sum + (c.restartCount ?? 0), 0).toString();
                },
              },
              {
                label: 'Age',
                getter: (pod: any) => pod.metadata.creationTimestamp ?? '-',
              },
            ]}
          />
        ) : (
          <p>No agent pods found.</p>
        )}
      </SectionBox>
    </>
  );
}
