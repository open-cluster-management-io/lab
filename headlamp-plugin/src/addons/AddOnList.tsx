import { Link, SectionBox, SimpleTable } from '@kinvolk/headlamp-plugin/lib/components/common';
import { ClusterManagementAddOn, ManagedClusterAddOn } from '../resources';
import StatusLabel, { conditionToStatus } from '../common/StatusLabel';
import { getConditionStatus, useOCMContext } from '../helpers';

export default function AddOnList() {
  const ctx = useOCMContext();
  const [addons, error] = ClusterManagementAddOn.useList();
  const [clusterAddons] = ManagedClusterAddOn.useList();

  if (ctx.type === 'loading') {
    return (
      <SectionBox title="Add-ons">
        <p>Loading...</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'managed') {
    return (
      <SectionBox title="Add-ons">
        <p>This is a managed cluster. Switch to the hub cluster to see add-ons.</p>
      </SectionBox>
    );
  }

  if (ctx.type === 'none') {
    return (
      <SectionBox title="Add-ons">
        <p>Open Cluster Management is not installed on this cluster.</p>
      </SectionBox>
    );
  }

  if (error) {
    return (
      <SectionBox title="Add-ons">
        <p>Error loading add-ons: {error.toString()}</p>
      </SectionBox>
    );
  }

  const clusterAddonsByName = new Map<string, InstanceType<typeof ManagedClusterAddOn>[]>();
  clusterAddons?.forEach((ca: InstanceType<typeof ManagedClusterAddOn>) => {
    const list = clusterAddonsByName.get(ca.metadata.name) ?? [];
    list.push(ca);
    clusterAddonsByName.set(ca.metadata.name, list);
  });

  return (
    <>
      <SectionBox title="Add-ons">
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
              sort: true,
            },
            {
              label: 'Install Strategy',
              getter: (a: InstanceType<typeof ClusterManagementAddOn>) => a.installStrategy,
            },
            {
              label: 'Clusters',
              getter: (a: InstanceType<typeof ClusterManagementAddOn>) => {
                const cas = clusterAddonsByName.get(a.metadata.name) ?? [];
                const available = cas.filter((ca: InstanceType<typeof ManagedClusterAddOn>) => {
                  const cond = getConditionStatus(ca.jsonData.status?.conditions, 'Available');
                  return cond?.status === 'True';
                }).length;
                return `${available} / ${cas.length} available`;
              },
            },
            {
              label: 'Age',
              getter: (a: InstanceType<typeof ClusterManagementAddOn>) =>
                a.metadata.creationTimestamp ?? '-',
              sort: true,
            },
          ]}
        />
      </SectionBox>
    </>
  );
}
