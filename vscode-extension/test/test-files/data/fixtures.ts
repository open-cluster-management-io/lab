import * as builder from '../../data/builder';
import * as k8s from '@kubernetes/client-node';
import * as loader from '../../data/loader';


/* ################################
########## Fake Clusters ##########
################################ */
export const k8sCluster1: k8s.Cluster = {
	name: 'fakeCluster1',
	server: 'https://my-fake-first-server:6443',
	skipTLSVerify: false
};

export const k8sCluster2: k8s.Cluster = {
	name: 'fakeCluster2',
	server: 'https://my-fake-second-server:6443',
	skipTLSVerify: false
};

export const connectedCluster1: builder.ConnectedCluster = {
	kluster: k8sCluster1,
	name: k8sCluster1.name,
	server: k8sCluster1.server,
};

export const connectedCluster2: builder.ConnectedCluster = {
	kluster: k8sCluster2,
	name: k8sCluster2.name,
	server: k8sCluster2.server,
};

/* ################################
########### Fake Users ############
################################ */
export const k8sUser1: k8s.User = {
	name: 'fakeUser1'
};

export const k8sUser2: k8s.User = {
	name: 'fakeUser2'
};

export const connectedUser1: builder.ConnectedUser = {
	kuser: k8sUser1,
	name: k8sUser1.name,
};

export const connectedUser2: builder.ConnectedUser = {
	kuser: k8sUser2,
	name: k8sUser2.name,
};

/* ################################
########## Fake Contexts ##########
################################ */
export const k8sContext1: k8s.Context = {
	name: 'fakeContext1',
	cluster: k8sCluster1.name,
	user: k8sUser1.name
};

export const k8sContext2: k8s.Context = {
	name: 'fakeContext2',
	cluster: k8sCluster2.name,
	user: k8sUser2.name
};

export const connectedContext1: builder.ConnectedContext = {
	kontext: k8sContext1,
	name: k8sContext1.name,
	cluster: connectedCluster1,
	user: connectedUser1
};

export const connectedContext2: builder.ConnectedContext = {
	kontext: k8sContext2,
	name: k8sContext2.name,
	cluster: connectedCluster2,
	user: connectedUser2
};

/* ################################
############ Fake CRDS ############
################################ */
export const k8sCrd1Namespaced: k8s.V1CustomResourceDefinition = {
	metadata: {
		name: 'fakecrd1s.fake.group.open-cluster-management'
	},
	spec: {
		names: {
			kind: 'FakeCrd1',
			listKind: 'FakeCrd1List',
			plural: 'fakecrd1s',
			singular: 'fakecrd1'
		},
		group: 'fake.group.open-cluster-management',
		scope: 'Namespaced',
		versions: [
			{
				name: 'v1beta1',
				served: false,
				storage: false,
				schema: {}
			},
			{
				name: 'v1beta2',
				served: true,
				storage: false,
				deprecated: true,
				deprecationWarning: 'please use v1',
				schema: {}
			},
			{
				name: 'v1',
				served: true,
				storage: true,
				schema: {}
			}
		]
	}
};

export const k8sCrd2Clustered: k8s.V1CustomResourceDefinition = {
	metadata: {
		name: 'fakecrd2s.another.fake.group.open-cluster-management'
	},
	spec: {
		names: {
			kind: 'FakeCrd2',
			listKind: 'FakeCrd2List',
			plural: 'fakecrd2s',
			singular: 'fakecrd2'
		},
		group: 'another.fake.group.open-cluster-management',
		scope: 'Cluster',
		versions: [
			{
				name: 'v1beta1',
				served: true,
				storage: false,
				schema: {}
			},
			{
				name: 'v1beta2',
				served: true,
				storage: true,
				schema: {}
			},
		]
	}
};

export const ocmCrd1Namespaced: loader.OcmResourceDefinition = {
	krd: k8sCrd1Namespaced,
	// @ts-ignore
	name: k8sCrd1Namespaced.metadata.name,
	plural: k8sCrd1Namespaced.spec.names.plural,
	namespaced: true,
	kind: k8sCrd1Namespaced.spec.names.kind,
	version: 'v1',
	group: k8sCrd1Namespaced.spec.group
};

export const ocmCrd2Clustered: loader.OcmResourceDefinition = {
	krd: k8sCrd2Clustered,
	// @ts-ignore
	name: k8sCrd2Clustered.metadata.name,
	plural: k8sCrd2Clustered.spec.names.plural,
	namespaced: false,
	kind: k8sCrd2Clustered.spec.names.kind,
	version: 'v1beta2',
	group: k8sCrd2Clustered.spec.group
};

/* ################################
############ Fake CRS #############
################################ */
export const k8sNamespace: k8s.V1Namespace = {
	metadata: {
		name: 'my-fake-app-namespace'
	}
};

export const k8sCr1Namespaced = {
	apiVersion: `${k8sCrd1Namespaced.spec.group}/v1`,
	kind: k8sCrd1Namespaced.spec.names.kind,
	metadata: {
		name: 'my-fake-namespaced-resource',
		namespace: k8sNamespace.metadata?.name
	},
	spec: {},
	status: {}
};


export const k8sCr2Clustered = {
	apiVersion: `${k8sCrd2Clustered.spec.group}/v1beta2`,
	kind: k8sCrd2Clustered.spec.names.kind,
	metadata: {
		name: 'my-fake-clustered-resource'
	},
	spec: {},
	status: {}
};

export const ocmCr1Namespaced: loader.OcmResource = {
	kr: k8sCr1Namespaced,
	crd: ocmCrd1Namespaced,
	name: k8sCr1Namespaced.metadata.name,
	namespace: k8sNamespace.metadata?.name
};

export const ocmCr2Clustered: loader.OcmResource = {
	kr: k8sCr2Clustered,
	crd: ocmCrd2Clustered,
	name: k8sCr2Clustered.metadata.name,
	namespace: undefined
};
