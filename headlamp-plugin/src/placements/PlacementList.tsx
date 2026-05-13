import { Link, SectionBox, SimpleTable } from '@kinvolk/headlamp-plugin/lib/components/common';
import { Placement } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { getConditionStatus, useOCMContext } from '../helpers';

export default function PlacementList() {
  const ctx = useOCMContext();
  const [placements, error] = Placement.useList();

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="Placements">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'managed') {
    return (
      <SectionBox title="Placements">
        <p>This is a managed cluster. Switch to the hub cluster to see placements.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="Placements">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  if (error) {
    return (
      <SectionBox title="Placements">
        <p>Error loading placements: {error.toString()}</p>
      </SectionBox>
    );
  }

  return (
    <SectionBox title="Placements">
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
            sort: true,
          },
          {
            label: 'Namespace',
            getter: (p: InstanceType<typeof Placement>) => p.metadata.namespace,
          },
          {
            label: 'Cluster Sets',
            getter: (p: InstanceType<typeof Placement>) =>
              p.jsonData.spec?.clusterSets?.join(', ') || '-',
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
          {
            label: 'Age',
            getter: (p: InstanceType<typeof Placement>) =>
              p.metadata.creationTimestamp ?? '-',
            sort: true,
          },
        ]}
      />
    </SectionBox>
  );
}
