import {
  registerSidebarEntry,
  registerRoute,
  registerSidebarEntryFilter,
} from '@kinvolk/headlamp-plugin/lib';
import Overview from './overview/Overview';
import ClusterList from './clusters/ClusterList';
import ClusterDetail from './clusters/ClusterDetail';
import ManagedOverview from './managed/ManagedOverview';
import PlacementList from './placements/PlacementList';
import PlacementDetail from './placements/PlacementDetail';
import ManifestWorkList from './manifestwork/ManifestWorkList';
import ManifestWorkDetail from './manifestwork/ManifestWorkDetail';
import AddOnList from './addons/AddOnList';
import AddOnDetail from './addons/AddOnDetail';

// Top-level OCM sidebar group
registerSidebarEntry({
  parent: null,
  name: 'ocm',
  label: 'OCM',
  icon: 'vaadin:cluster',
});

// Overview
registerSidebarEntry({
  parent: 'ocm',
  name: 'ocm-overview',
  label: 'Overview',
  url: '/ocm/overview',
});

registerRoute({
  path: '/ocm/overview',
  sidebar: 'ocm-overview',
  name: 'ocm-overview',
  exact: true,
  component: () => <Overview />,
});

// Clusters
registerSidebarEntry({
  parent: 'ocm',
  name: 'ocm-clusters',
  label: 'Clusters',
  url: '/ocm/clusters',
});

registerRoute({
  path: '/ocm/clusters',
  sidebar: 'ocm-clusters',
  name: 'ocm-clusters',
  exact: true,
  component: () => <ClusterList />,
});

registerRoute({
  path: '/ocm/clusters/:name',
  sidebar: 'ocm-clusters',
  name: 'ocm-cluster-detail',
  exact: true,
  component: () => <ClusterDetail />,
});

// Placements
registerSidebarEntry({
  parent: 'ocm',
  name: 'ocm-placements',
  label: 'Placements',
  url: '/ocm/placements',
});

registerRoute({
  path: '/ocm/placements',
  sidebar: 'ocm-placements',
  name: 'ocm-placements',
  exact: true,
  component: () => <PlacementList />,
});

registerRoute({
  path: '/ocm/placements/:namespace/:name',
  sidebar: 'ocm-placements',
  name: 'ocm-placement-detail',
  exact: true,
  component: () => <PlacementDetail />,
});

// ManifestWork
registerSidebarEntry({
  parent: 'ocm',
  name: 'ocm-manifestwork',
  label: 'ManifestWork',
  url: '/ocm/manifestwork',
});

registerRoute({
  path: '/ocm/manifestwork',
  sidebar: 'ocm-manifestwork',
  name: 'ocm-manifestwork',
  exact: true,
  component: () => <ManifestWorkList />,
});

registerRoute({
  path: '/ocm/manifestwork/:namespace/:name',
  sidebar: 'ocm-manifestwork',
  name: 'ocm-manifestwork-detail',
  exact: true,
  component: () => <ManifestWorkDetail />,
});

// Add-ons
registerSidebarEntry({
  parent: 'ocm',
  name: 'ocm-addons',
  label: 'Add-ons',
  url: '/ocm/addons',
});

registerRoute({
  path: '/ocm/addons',
  sidebar: 'ocm-addons',
  name: 'ocm-addons',
  exact: true,
  component: () => <AddOnList />,
});

registerRoute({
  path: '/ocm/addons/:name',
  sidebar: 'ocm-addons',
  name: 'ocm-addon-detail',
  exact: true,
  component: () => <AddOnDetail />,
});

// This Cluster (managed/spoke view)
registerSidebarEntry({
  parent: 'ocm',
  name: 'ocm-managed',
  label: 'This Cluster',
  url: '/ocm/managed',
});

registerRoute({
  path: '/ocm/managed',
  sidebar: 'ocm-managed',
  name: 'ocm-managed',
  exact: true,
  component: () => <ManagedOverview />,
});

