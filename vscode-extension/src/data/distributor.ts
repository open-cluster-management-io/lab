import * as loader from './loader';
import { ConnectedContext } from './builder';

// the ManagedCluster kind is used in hub clusters for managing spoke clusters
const hubManagedClusterKind = 'ManagedCluster';
// these are the kinds we expect and support for running on a hub cluster
const hubKinds = ['ManifestWork', 'Placement', 'PlacementDecision', 'ManagedClusterSet', 'ManagedClusterAddOn', 'ClusterManager', 'SubscriptionReport'];
// the Klusterlet kind is used in spoke clusters for communicating with the hub cluster
const spokeKlusterletKind = 'Klusterlet';
// these are the kinds we expect and support for running on a spoke cluster
const spokeKinds = ['AppliedManifestWork', 'SubscriptionStatus'];

// listeners for messages posted here resides in the components package in the react module
// will distribute two types of messages to the parameterized consumer:
// {selectedContext: ...ConnectedContext...} - for every invocation
// { crsDistribution: { kind: X, crs: ...[OcmResource]...}} - for every relevant kind
export async function distributeMessages(selectedContext: ConnectedContext, consumer: CallableFunction): Promise<void> {
	// get the loader instance
	let load = loader.Load.getLoader();

	// retrieve all managedclusters and klusterlets resources
	let managedClusters = await load.getCrs(hubManagedClusterKind);
	let klusterlets = await load.getCrs(spokeKlusterletKind);

	// populate the SelectedContext ui component
	consumer({selectedContext: JSON.stringify(selectedContext)});

	let crsDistributions: Promise<void>[] = [];

	// a cluster with ManagedCluster resources, acts as a hub cluster
	if (managedClusters.length > 0 ){
		// populate the ManagedClusters ui component
		crsDistributions.push(distributeCrs(hubManagedClusterKind, consumer, managedClusters));
		// populate common crs used by the hub cluster
		hubKinds.forEach(kind => crsDistributions.push(distributeCrsForKind(kind, load, consumer)));
	}

	// a cluster with Klusterlet resources, acts as a spoke cluster
	if (klusterlets.length > 0 ){
		// populate the Klusterlets ui component
		crsDistributions.push(distributeCrs(spokeKlusterletKind, consumer, klusterlets));
		// populate common crs used by the spoke cluster
		spokeKinds.forEach(kind => crsDistributions.push(distributeCrsForKind(kind, load, consumer)));
	}

	await Promise.allSettled(crsDistributions);
}

// load crs for a kind, and distribute the crs message to the consumer
async function distributeCrsForKind(kind: string, load: loader.Load, consumer: CallableFunction): Promise<void> {
	let crs = await load.getCrs(kind);
	await distributeCrs(kind, consumer, crs);
}

// distribute the crs message to the consumer
async function distributeCrs(kind: string, consumer: CallableFunction, crs: loader.OcmResource[]): Promise<void> {
	if (crs.length > 0) {
		consumer({ crsDistribution: { kind: kind, crs: JSON.stringify(crs)}});
	}
}
