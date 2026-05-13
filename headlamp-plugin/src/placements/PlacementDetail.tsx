import { useParams } from 'react-router-dom';
import {
  MainInfoSection,
  SectionBox,
  SimpleTable,
  ConditionsTable,
  Link,
} from '@kinvolk/headlamp-plugin/lib/components/common';
import { Placement, PlacementDecision } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { getConditionStatus, useOCMContext } from '../helpers';

export default function PlacementDetail() {
  const { namespace, name } = useParams<{ namespace: string; name: string }>();
  const ctx = useOCMContext();
  const [placement, error] = Placement.useGet(name!, namespace!);
  const [decisions] = PlacementDecision.useList({ namespace: namespace! });

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="Placement">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'managed') {
    return (
      <SectionBox title="Placement">
        <p>This is a managed cluster. Switch to the hub cluster to see placements.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="Placement">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  if (error) {
    return (
      <SectionBox title="Placement">
        <p>Error loading placement: {error.toString()}</p>
      </SectionBox>
    );
  }

  if (!placement) {
    return (
      <SectionBox title="Placement">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  const satisfied = getConditionStatus(
    placement.jsonData.status?.conditions,
    'PlacementSatisfied'
  );

  const placementDecisions = decisions?.filter(
    (d: InstanceType<typeof PlacementDecision>) =>
      d.jsonData.metadata?.labels?.['cluster.open-cluster-management.io/placement'] === name
  );

  const selectedClusters = placementDecisions?.flatMap(
    (d: InstanceType<typeof PlacementDecision>) => d.decisions
  ) ?? [];

  return (
    <>
      <MainInfoSection
        resource={placement}
        title="Placement"
        extraInfo={[
          {
            name: 'Namespace',
            value: placement.metadata.namespace,
          },
          {
            name: 'Cluster Sets',
            value: placement.jsonData.spec?.clusterSets?.join(', ') || '-',
          },
          {
            name: 'Requested Clusters',
            value: placement.jsonData.spec?.numberOfClusters?.toString() ?? 'Any',
          },
          {
            name: 'Selected Clusters',
            value: placement.numberOfSelectedClusters.toString(),
          },
          {
            name: 'Status',
            value: (
              <StatusLabel status={conditionToStatus(satisfied?.status)}>
                {satisfied?.status === 'True'
                  ? 'Satisfied'
                  : satisfied?.status === 'False'
                    ? 'Not Satisfied'
                    : 'Unknown'}
              </StatusLabel>
            ),
          },
        ]}
      />

      <SectionBox title="Selected Clusters">
        {selectedClusters.length > 0 ? (
          <SimpleTable
            data={selectedClusters}
            columns={[
              {
                label: 'Cluster',
                getter: (row: { clusterName: string }) => (
                  <Link
                    routeName="ocm-cluster-detail"
                    params={{ name: row.clusterName }}
                  >
                    {row.clusterName}
                  </Link>
                ),
              },
              {
                label: 'Reason',
                getter: (row: { reason: string }) => row.reason || '-',
              },
            ]}
          />
        ) : (
          <p>No clusters selected.</p>
        )}
      </SectionBox>

      <SectionBox title="Conditions">
        <ConditionsTable resource={placement} />
      </SectionBox>
    </>
  );
}
