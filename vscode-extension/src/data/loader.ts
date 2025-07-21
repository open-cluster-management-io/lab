import * as builder from './builder';
import * as k8s from '@kubernetes/client-node';

const OCM_GROUP = 'open-cluster-management';

// our ocm resource component is used for encapsulating an ocm-related k8s resource
export class OcmResource {
	readonly kr: any;
	readonly crd: OcmResourceDefinition;
	readonly name: string;
	readonly namespace?: string;

	constructor(kr: any, crd: OcmResourceDefinition, namespace?: string) {
		this.kr = kr;
		this.crd = crd;
		this.name = kr.metadata.name;
		this.namespace = namespace;
	}
}

// our ocm resource definition component is used for encapsulating an ocm-related k8s resource definition
export class OcmResourceDefinition {
	readonly krd: k8s.V1CustomResourceDefinition;
	readonly name: string;
	readonly plural: string;
	readonly namespaced: boolean;
	readonly kind: string;
	readonly version: string;
	readonly group: string;

	constructor(krd: k8s.V1CustomResourceDefinition) {
		this.krd = krd;
		this.name = krd.metadata?.name as string;
		this.plural = krd.spec.names.plural;
		this.namespaced = krd.spec.scope === 'Namespaced';
		this.kind = krd.spec.names.kind;
		this.version = krd.spec.versions.find(v => v.storage)?.name as string;
		this.group = krd.spec.group;
	}
}

// use this class for obtaining our loader singleton used for accessing k8s via kubectl configuration
export class Load {
	private static loader: Load;
	private config: k8s.KubeConfig;
	private build: builder.Build;
	private extApi?: k8s.ApiextensionsV1Api;
	private objApi?: k8s.CustomObjectsApi;
	private coreApi?: k8s.CoreV1Api;

	// serve (and build) our loader instance
	static getLoader(): Load {
		if (!Load.loader) {
			let config = new k8s.KubeConfig();
			let build = new builder.Build(config);
			Load.loader = new Load(config, build);
		}
		return Load.loader;
	}

	private constructor(config: k8s.KubeConfig, build: builder.Build) {
		this.config = config;
		this.build = build;
		this.refresh();
	}

	// use the refresh function to load the configuration from kube-config and refresh our api clients
	refresh(): void {
		this.config.loadFromDefault();
		this.refreshClients();
	}

	// refresh our api clients for existing contexts
	private refreshClients(): void {
		if (this.getContext()) {
			this.extApi = this.config.makeApiClient(k8s.ApiextensionsV1Api);
			this.objApi = this.config.makeApiClient(k8s.CustomObjectsApi);
			this.coreApi = this.config.makeApiClient(k8s.CoreV1Api);
		}
	}

	// verify the cluster listed in the current context is reachable
	async verifyReachability(): Promise<void> {
		try {
			let r = await this.coreApi?.listNode();
			if (!r) {
				return Promise.reject('Cluster is not accessible');
			}
			if (r.response.statusCode !== 200) {
				return Promise.reject(`Cluster is not accessible, ${String(r.response.statusCode)}`);
			}
		} catch (e: any) {
			return Promise.reject(`Cluster is not accessible, ${String(e)}`);
		}
	}

	// set the current context and refresh the api client accordingly
	setContext(context: string | builder.ConnectedContext): void {
		this.config.setCurrentContext(typeof context === 'string' ? context : context.name);
		this.refreshClients();
	}

	// get the current context if one configured
	getContext(): builder.ConnectedContext | undefined {
		return this.build.context(this.config.currentContext);
	}

	// get all contexts from the loaded kube-config
	getContexts(): builder.ConnectedContext[] {
		return this.config.contexts
			.map(kontext => this.build.context(kontext))
			.filter((context): context is builder.ConnectedContext => !!context);
	}

	// get a cluster from the loaded kube-config
	getCluster(name: string): builder.ConnectedCluster | undefined {
		return this.build.cluster(name);
	}

	// get all clusters from the loaded kube-config
	getClusters(): builder.ConnectedCluster[] {
		return this.config.clusters
			.map(kluster => this.build.cluster(kluster))
			.filter((cluster): cluster is builder.ConnectedCluster => !!cluster);
	}

	// get a user from the loaded kube-config
	getUser(name: string): builder.ConnectedUser | undefined {
		return this.build.user(name);
	}

	// get all users from the loaded kube-config
	getUsers(): builder.ConnectedUser[] {
		return this.config.users
			.map(kuser => this.build.user(kuser))
			.filter((user): user is builder.ConnectedUser => !!user);
	}

	// fetch a crd for a kind
	async getCrd(kind: string): Promise<OcmResourceDefinition | undefined> {
		return (await this.getCrds()).filter(c => c.kind === kind)[0];
	}

	// fetch all crds for the open-cluster-management group
	async getCrds(): Promise<OcmResourceDefinition[]> {
		let retCrds: OcmResourceDefinition[] = [];
		let response = await this.extApi?.listCustomResourceDefinition();
		if (response && response.response.statusCode === 200) {
			retCrds.push(...response.body.items.filter(item => item.spec.group.includes(OCM_GROUP)).map(crd => new OcmResourceDefinition(crd)));
		}
		return retCrds;
	}

	// get all crs for a crd
	async getCrs(item: string | OcmResourceDefinition): Promise<OcmResource[]> {
		let retRes: OcmResource[] = [];
		let crd = typeof item === 'string' ? await this.getCrd(item) : item;
		if (crd) {
			let crs = crd.namespaced ? await this.getNamespacedCrs(crd) : await this.getClusterCrs(crd);
			retRes.push(...crs);
		}
		return retRes;
	}

	// get clustered crs for a clustered crd
	private async getClusterCrs(crd: OcmResourceDefinition): Promise<OcmResource[]> {
		let retRes: OcmResource[] = [];
		let response = await this.objApi?.listClusterCustomObject(crd.group, crd.version, crd.plural);
		if (response && response.response.statusCode === 200) {
			// @ts-ignore
			retRes.push(...response.body.items.map(item => new OcmResource(item, crd)));
		}
		return retRes;
	}

	// get namespaced crs for a namespaced crd
	private async getNamespacedCrs(crd: OcmResourceDefinition): Promise<OcmResource[]> {
		let retRes: OcmResource[] = [];
		let nsResponse = await this.coreApi?.listNamespace();
		if (nsResponse && nsResponse.response.statusCode === 200) {
			let objectPromises: Promise<void>[] = [];
			nsResponse.body.items.forEach(async nsitem => {
				let namespace = nsitem.metadata?.name as string;
				let objectPromise = this.objApi?.listNamespacedCustomObject(crd.group, crd.version, namespace, crd.plural)
					.then(objResponse => {
						// @ts-ignore
						if (objResponse && objResponse.response.statusCode === 200 && objResponse.body.items.length > 0) {
							// @ts-ignore
							retRes.push(...objResponse.body.items.map(item => new OcmResource(item, crd, namespace)));
						}
					});
				if (objectPromise) {
					objectPromises.push(objectPromise);
				}
			});
			await Promise.all(objectPromises);
		}
		return retRes;
	}
}
