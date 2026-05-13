import { Link, SectionBox, SimpleTable } from '@kinvolk/headlamp-plugin/lib/components/common';
import { ManifestWork } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { getConditionStatus, useOCMContext } from '../helpers';

export default function ManifestWorkList() {
  const ctx = useOCMContext();
  const [works, error] = ManifestWork.useList();

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="ManifestWorks">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'managed') {
    return (
      <SectionBox title="ManifestWorks">
        <p>This is a managed cluster. Switch to the hub cluster to see manifest works.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="ManifestWorks">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  if (error) {
    return (
      <SectionBox title="ManifestWorks">
        <p>Error loading manifest works: {error.toString()}</p>
      </SectionBox>
    );
  }

  return (
    <SectionBox title="ManifestWorks">
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
            sort: true,
          },
          {
            label: 'Target Cluster',
            getter: (w: InstanceType<typeof ManifestWork>) => (
              <Link
                routeName="ocm-cluster-detail"
                params={{ name: w.metadata.namespace }}
              >
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
            label: 'Manifests',
            getter: (w: InstanceType<typeof ManifestWork>) =>
              w.manifestCount.toString(),
          },
          {
            label: 'Applied',
            getter: (w: InstanceType<typeof ManifestWork>) => {
              const applied = getConditionStatus(
                w.jsonData.status?.conditions,
                'Applied'
              );
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
              const available = getConditionStatus(
                w.jsonData.status?.conditions,
                'Available'
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
            label: 'Age',
            getter: (w: InstanceType<typeof ManifestWork>) =>
              w.metadata.creationTimestamp ?? '-',
            sort: true,
          },
        ]}
      />
    </SectionBox>
  );
}
