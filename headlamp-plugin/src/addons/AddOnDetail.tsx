import { useParams } from 'react-router-dom';
import {
  MainInfoSection,
  SectionBox,
  SimpleTable,
  ConditionsTable,
  Link,
} from '@kinvolk/headlamp-plugin/lib/components/common';
import { ClusterManagementAddOn, ManagedClusterAddOn } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { getConditionStatus, useOCMContext } from '../helpers';

export default function AddOnDetail() {
  const { name } = useParams<{ name: string }>();
  const ctx = useOCMContext();
  const [addon, error] = ClusterManagementAddOn.useGet(name!);
  const [clusterAddons] = ManagedClusterAddOn.useList();

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="Add-on">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'managed') {
    return (
      <SectionBox title="Add-on">
        <p>This is a managed cluster. Switch to the hub cluster to see add-ons.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="Add-on">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  if (error) {
    return (
      <SectionBox title="Add-on">
        <p>Error loading add-on: {error.toString()}</p>
      </SectionBox>
    );
  }

  if (!addon) {
    return (
      <SectionBox title="Add-on">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  const perCluster = clusterAddons?.filter(
    (ca: InstanceType<typeof ManagedClusterAddOn>) => ca.metadata.name === name
  ) ?? [];

  return (
    <>
      <MainInfoSection
        resource={addon}
        title="Add-on"
        extraInfo={[
          {
            name: 'Display Name',
            value: addon.displayName,
          },
          {
            name: 'Description',
            value: addon.description || '-',
          },
          {
            name: 'Install Strategy',
            value: addon.installStrategy,
          },
          {
            name: 'Clusters',
            value: perCluster.length.toString(),
          },
        ]}
      />

      <SectionBox title="Cluster Status">
        {perCluster.length > 0 ? (
          <SimpleTable
            data={perCluster}
            columns={[
              {
                label: 'Cluster',
                getter: (ca: InstanceType<typeof ManagedClusterAddOn>) => (
                  <Link
                    routeName="ocm-cluster-detail"
                    params={{ name: ca.metadata.namespace }}
                  >
                    {ca.metadata.namespace}
                  </Link>
                ),
                sort: true,
              },
              {
                label: 'Available',
                getter: (ca: InstanceType<typeof ManagedClusterAddOn>) => {
                  const available = getConditionStatus(
                    ca.jsonData.status?.conditions,
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
                label: 'Configured',
                getter: (ca: InstanceType<typeof ManagedClusterAddOn>) => {
                  const configured = getConditionStatus(
                    ca.jsonData.status?.conditions,
                    'Configured'
                  );
                  return (
                    <StatusLabel status={conditionToStatus(configured?.status)}>
                      {configured?.status === 'True'
                        ? 'Yes'
                        : configured?.status === 'False'
                          ? 'No'
                          : 'Unknown'}
                    </StatusLabel>
                  );
                },
              },
              {
                label: 'Install Namespace',
                getter: (ca: InstanceType<typeof ManagedClusterAddOn>) => ca.installNamespace,
              },
            ]}
          />
        ) : (
          <p>Not installed on any clusters.</p>
        )}
      </SectionBox>

      <SectionBox title="Conditions">
        {perCluster.map((ca: InstanceType<typeof ManagedClusterAddOn>) => (
          <div key={ca.metadata.namespace}>
            <h4>{ca.metadata.namespace}</h4>
            <ConditionsTable resource={ca} />
          </div>
        ))}
      </SectionBox>
    </>
  );
}
