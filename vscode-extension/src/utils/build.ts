
import * as shellTools from './shell';

const blankLines = new RegExp(/(^[ \t]*\r?\n)/, "gm");

export enum ClusterType {
	hub, managed
}

export interface Cluster {
	name: string,
	context: string
	type: ClusterType
}

export interface ProgressReport {
	increment: number,
	message: string
}

export interface ProgressReporter {
	(r: ProgressReport): void
}

export const defaultClusters: Cluster[] = [
	{
		name: 'hub',
		context: 'kind-hub',
		type: ClusterType.hub
	},
	{
		name: 'cluster1',
		context: 'kind-cluster1',
		type: ClusterType.managed
	},
	{
		name: 'cluster2',
		context: 'kind-cluster2',
		type: ClusterType.managed
	},
];

// create a kind cluster, fulfilled with stdout/stderr
function createKindCluster(cluster: Cluster):  Promise<string> {
	console.debug(`creating a kind cluster named ${cluster.name}`);
	return shellTools.executeShellCommand(`kind create cluster --name ${cluster.name}`);
}

// create kind clusters
async function createKindClusters(clusters: Cluster[], reporter: ProgressReporter): Promise<void|string> {
	console.debug(`creating ${clusters.length} kind clusters`);
	reporter({increment: 0 , message: `creating ${clusters.length} kind clusters`});
	let clusterPromises = clusters.map(cluster => createKindCluster(cluster));
	await Promise.allSettled(clusterPromises)
		.then((results: PromiseSettledResult<string>[]) => {
			if (results.filter((r) => r.status === 'rejected').length > 0) {
				return Promise.reject('failed creating kind clusters');
			}
		});
}

// initialize the hub cluster, resolves with the join command, rejects with the error message
async function initializeHubCluster(hubCluster: Cluster, reporter: ProgressReporter): Promise<string> {
	console.debug(`initializing the hub cluster named ${hubCluster.name}`);
	reporter({increment: 20 , message: `initializing the Hub cluster named ${hubCluster.name}`});
	return shellTools.executeShellCommand(`kubectl config use ${hubCluster.context}`)
		.then(() => shellTools.executeShellCommand('clusteradm init --use-bootstrap-token'))
		.then((stdout: string) => stdout.replace(blankLines, '').split(/\r?\n/)[2].trim())
		.catch(() => Promise.reject('failed initializing the hub cluster'));
}

// issue a join command from a managed cluster using a join command
async function issueJoinRequest(managedCluster: Cluster, joinCmd: string): Promise<string> {
	console.debug(`issuing a join request from the managed cluster named ${managedCluster.name}`);
	let fixedJoinCmd = joinCmd.replace('<cluster_name>', managedCluster.name).trim();
	return shellTools.executeShellCommand(`kubectl config use ${managedCluster.context}`)
		.then(() => shellTools.executeShellCommand(`${fixedJoinCmd} --force-internal-endpoint-lookup --wait`));
}

// issue join requests from the managed clusters to the hub
async function sendJoinRequests(
		joinCmd: string, managedClusters: Cluster[], reporter: ProgressReporter): Promise<void|string> {

	console.debug(`issuing join requests for ${managedClusters.length} managed clusters`);
	reporter({increment: 20 , message: 'issuing join requests for the managed clusters'});
	return managedClusters.reduce(
		(previousPromise, currentPromise) =>
			previousPromise.then(() => issueJoinRequest(currentPromise, joinCmd)),
			Promise.resolve('initial value')
	)
	.then(() => Promise.resolve())
	.catch(() => Promise.reject('failed to issue join requests'));
}

// approve join request made to the hub cluster by the managed clusters
async function acceptJoinRequest(hubCluster: Cluster, requesters: string) : Promise<string> {
	console.debug(`accepting join requests from ${requesters}`);
	return shellTools.executeShellCommand(`kubectl config use ${hubCluster.context}`)
		.then(() => shellTools.executeShellCommand(`clusteradm accept --clusters ${requesters} --wait`));
}

// accept the issued join commands by the managed clusters from the hub cluster
async function acceptAllJoinRequests(
	hubCluster: Cluster, managedClusters: Cluster[], reporter: ProgressReporter): Promise<string> {

	console.debug(`accepting ${managedClusters.length} on behalf of the ${hubCluster.name}`);
	reporter({increment: 20 , message: 'accepting the managed clusters join request from the hub cluster'});
	return acceptJoinRequest(hubCluster, managedClusters.map(mc => mc.name).join())
		.then(() => Promise.resolve('successfully created your local environment, have fun'))
		.catch(() => Promise.reject('failed to accept join requests'));
}

// log, report, and fulfil the build process
function fulfilBuild(msg: string, reporter: ProgressReporter, fulfil: (s: string) => void): void {
	console.debug(msg);
	reporter({increment: 100 , message: msg});
	fulfil(`OCM extension, ${msg}`);
}

// starts a local OCM kind env and return a promise
export async function buildLocalEnv(clusters: Cluster[], reporter: ProgressReporter): Promise<string> {
	return new Promise((resolve, reject) => {
		//Verify Clusters Info
		let hubClusters = clusters.filter(c => c.type === ClusterType.hub);
		let managedClusters = clusters.filter(c => c.type === ClusterType.managed);

		if (hubClusters.length !== 1 || managedClusters.length < 1) {
			let errMsg = `required 1 hub and at least 1 managed cluster, found ${hubClusters.length} and ${managedClusters.length}`;
			fulfilBuild(errMsg, reporter, reject);
		} else {
			let hubCluster = hubClusters[0];
			createKindClusters(clusters, reporter) // create the kind clusters
			.then(() => initializeHubCluster(hubCluster, reporter))  // initialize the hub cluster
			.then((joinCmd: string) => sendJoinRequests(joinCmd, managedClusters, reporter)) // send join requests
			.then(() => acceptAllJoinRequests(hubCluster, managedClusters, reporter)) // accept join requests
			.then((msg: string) => fulfilBuild(msg, reporter, resolve)) // resolve build
			.catch((msg: string) => fulfilBuild(msg, reporter, reject)); // reject build
		}
	});
}
