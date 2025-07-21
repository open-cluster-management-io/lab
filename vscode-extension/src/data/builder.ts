import * as k8s from '@kubernetes/client-node';

// our connected context component is used for encapsulating a kubectl context
export class ConnectedContext {
	readonly kontext: k8s.Context;
	readonly name: string;
    readonly cluster: ConnectedCluster;
    readonly user: ConnectedUser;

    constructor(kontext: k8s.Context, cluster: ConnectedCluster, user: ConnectedUser) {
        this.kontext = kontext;
		this.name = kontext.name;
        this.cluster = cluster;
        this.user = user;
    }
}

// our connected cluster component is used for encapsulating a kubectl cluster
export class ConnectedCluster {
	readonly kluster: k8s.Cluster;
	readonly name: string;
    readonly server: string;

    constructor(cluster: k8s.Cluster) {
        this.kluster = cluster;
		this.name = cluster.name;
        this.server = cluster.server;
    }
}

// our connected user component is used for encapsulating a kubectl user
export class ConnectedUser {
	readonly kuser: k8s.User;
	readonly name: string;

	constructor(user: k8s.User) {
		this.kuser = user;
		this.name = user.name;
	}
}

// a builder class used for building our component
export class Build {
	private config: k8s.KubeConfig;

	constructor(config: k8s.KubeConfig) {
		this.config = config;
	}

	// build a context component
	context(item: string | k8s.Context): ConnectedContext | undefined {
		let connectedContext;
		let kontext = typeof item === 'string' ? this.config.getContextObject(item) : item;
		if (kontext) {
			let cluster = this.cluster(kontext.cluster);
			let user = this.user(kontext.user);
			if (cluster && user) {
				connectedContext = new ConnectedContext(kontext, cluster, user);
			}
		}
		return connectedContext;
	}

	// build a cluster component
	cluster(item: string | k8s.Cluster): ConnectedCluster | undefined{
		let connectedCluster;
		let kluster = typeof item === 'string' ? this.config.getCluster(item) : item;
		if (kluster) {
			connectedCluster = new ConnectedCluster(kluster);
		}
		return connectedCluster;
	}

	// build a user component
	user(item: string | k8s.User): ConnectedUser | undefined {
		let connectedUser;
		let kuser = typeof item === 'string' ? this.config.getUser(item) : item;
		if (kuser) {
			connectedUser = new ConnectedUser(kuser);
		}
		return connectedUser;
	}
}
