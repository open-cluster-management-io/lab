import * as chaiAsPromised from 'chai-as-promised';
import * as shellTools from '../../../src/utils/shell';
import * as sinon from 'sinon';
import * as sinonChai from 'sinon-chai';
import { Cluster, ClusterType, buildLocalEnv } from '../../../src/utils/build';
import { use as chaiUse, expect } from 'chai';
import { beforeEach } from 'mocha';

chaiUse(chaiAsPromised);
chaiUse(sinonChai);

suite('Test cases for the build utility functions', () => {
	let shellExecutionStub: sinon.SinonStub;
	let fakeProgressReporter: sinon.SinonStub;

	const dummyHubCluster1: Cluster = {
		name: 'dummyHub1',
		context: 'kind-dummyHub1',
		type: ClusterType.hub
	};

	const dummyHubCluster2: Cluster = {
		name: 'dummyHub2',
		context: 'kind-dummyHub2',
		type: ClusterType.hub
	};

	const dummyManagedCluster1: Cluster = {
		name: 'dummyCluster1',
		context: 'kind-dummyCluster1',
		type: ClusterType.managed
	};

	const dummyManagedCluster2: Cluster = {
		name: 'dummyCluster2',
		context: 'kind-dummyCluster2',
		type: ClusterType.managed
	};

	const fullClusterList = [dummyHubCluster1, dummyManagedCluster1, dummyManagedCluster2];

	const expectedJoinCommand = 'clusteradm join --hub-token xyzxyz --hub-apiserver https://127.0.0.1:1234 --cluster-name';

	const successfulHubInitialization = `The multicluster hub control plane has been initialized successfully!

	You can now register cluster(s) to the hub control plane. Log onto those cluster(s) and run the following command:

		${expectedJoinCommand} <cluster_name>

	Replace <cluster_name> with a cluster name of your choice. For example, cluster1.

	`;

	beforeEach(() => {
		sinon.restore(); // unwrap previously wrapped sinon objects
		shellExecutionStub = sinon.stub(shellTools, 'executeShellCommand'); // stub shell execution utility function
		fakeProgressReporter = sinon.stub(); // mock a fake progress bar
		sinon.stub(console, 'debug'); // silence debug logs
	});

	suite('Testing buildLocalEnv', () => {
		test('When building with no hub cluster, the build should be rejected', async () => {
			// given the requested cluster list contains two managed clusters and no hub cluster
			let clusters = [dummyManagedCluster1, dummyManagedCluster2];
			return Promise.all([
				// then the build should be rejected
				expect(buildLocalEnv(clusters, fakeProgressReporter))
					.to.eventually.be.rejectedWith(
						'OCM extension, required 1 hub and at least 1 managed cluster, found 0 and 2'),
				// then the progress reporter should be incremented fully
				expect(fakeProgressReporter).to.have.been.calledOnceWith(
					{increment: 100 , message: 'required 1 hub and at least 1 managed cluster, found 0 and 2'})
			]);
		});

		test('When building with multiple hub clusters, the build should be rejected', async () => {
			// given the requested cluster list contains two hub clusters
			let clusters = [dummyHubCluster1, dummyHubCluster2, dummyManagedCluster1];
			return Promise.all([
				// then the build should be rejected
				expect(buildLocalEnv(clusters, fakeProgressReporter))
					.to.eventually.be.rejectedWith(
						'OCM extension, required 1 hub and at least 1 managed cluster, found 2 and 1'),
				// then the progress reporter should be incremented full
				expect(fakeProgressReporter).to.have.been.calledOnceWith(
					{increment: 100 , message: 'required 1 hub and at least 1 managed cluster, found 2 and 1'})
			]);
		});

		test('When building with no managed clusters, the build should be rejected', async () => {
			// given the requested cluster list contains two hub clusters
			let clusters = [dummyHubCluster1];
			return Promise.all([
				// then the build should be rejected
				expect(buildLocalEnv(clusters, fakeProgressReporter))
					.to.eventually.be.rejectedWith(
						'OCM extension, required 1 hub and at least 1 managed cluster, found 1 and 0'),
				// then the progress reporter should be incremented full
				expect(fakeProgressReporter).to.have.been.calledOnceWith(
					{increment: 100 , message: 'required 1 hub and at least 1 managed cluster, found 1 and 0'})
			]);
		});

		test('When failed creating kind clusters, the build should be rejected', async () => {
			// given kind will fail creating the clusters
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kind create cluster --name')))
				.rejects('will-always-fail');
			return Promise.all([
				// then the build should be rejected
				expect(buildLocalEnv(fullClusterList, fakeProgressReporter))
					.to.eventually.be.rejectedWith('OCM extension, failed creating kind clusters')
					.then(() => {
						// then track the progress of the progress reporter
						expect(fakeProgressReporter.firstCall.firstArg)
							.to.contain({increment: 0 , message: 'creating 3 kind clusters'});
						expect(fakeProgressReporter.secondCall.firstArg)
							.to.contain({increment: 100 , message: 'failed creating kind clusters'});
					}),
			]);
		});

		test('When failed initializing the hub cluster, the build should be rejected', async () => {
			// given kind will successfully create the clusters
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kind create cluster --name')))
				.resolves();
			// given kubectl context switching will work
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kubectl config use')))
				.resolves();
			// given the hub initialization with clusteradm will fail
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('clusteradm init --use-bootstrap-token')))
				.rejects('will-always-fail');
			return Promise.all([
				// then the build should be rejected
				expect(buildLocalEnv(fullClusterList, fakeProgressReporter))
					.to.eventually.be.rejectedWith('OCM extension, failed initializing the hub cluster')
					.then(() => {
						// then track the progress of the progress reporter
						expect(fakeProgressReporter.firstCall.firstArg)
							.to.contain({increment: 0 , message: 'creating 3 kind clusters'});
						expect(fakeProgressReporter.secondCall.firstArg)
							.to.contain({increment: 20 , message: 'initializing the Hub cluster named dummyHub1'});
						expect(fakeProgressReporter.thirdCall.firstArg)
							.to.contain({increment: 100 , message: 'failed initializing the hub cluster'});
					}),
			]);
		});

		test('When sending the join request fails, the build should be rejected', async () => {
			// given kind will successfully create the clusters
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kind create cluster --name')))
				.resolves();
			// given kubectl context switching will work
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kubectl config use')))
				.resolves();
			// given the hub cluster will be successfully initialized
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('clusteradm init --use-bootstrap-token')))
				.resolves();
			// given the hub cluster will be successfully initialized
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('clusteradm init --use-bootstrap-token')))
				.resolves(successfulHubInitialization);
			// given the join request fails
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith(expectedJoinCommand)))
				.rejects('will-always-fail');
			return Promise.all([
				// then the build should be rejected
				expect(buildLocalEnv(fullClusterList, fakeProgressReporter))
					.to.eventually.be.rejectedWith('OCM extension, failed to issue join requests')
					.then(() => {
						// then track the progress of the progress reporter
						expect(fakeProgressReporter.getCall(0).firstArg)
							.to.contain({increment: 0 , message: 'creating 3 kind clusters'});
						expect(fakeProgressReporter.getCall(1).firstArg)
							.to.contain({increment: 20 , message: 'initializing the Hub cluster named dummyHub1'});
						expect(fakeProgressReporter.getCall(2).firstArg)
							.to.contain({increment: 20 , message: 'issuing join requests for the managed clusters'});
						expect(fakeProgressReporter.getCall(3).firstArg)
							.to.contain({increment: 100 , message: 'failed to issue join requests'});
					}),
			]);
		});

		test('When accepting the join requests fails, the build should be rejected', async () => {
			// given kind will successfully create the clusters
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kind create cluster --name')))
				.resolves();
			// given kubectl context switching will work
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kubectl config use')))
				.resolves();
			// given the hub cluster will be successfully initialized
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('clusteradm init --use-bootstrap-token')))
				.resolves();
			// given the hub cluster will be successfully initialized
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('clusteradm init --use-bootstrap-token')))
				.resolves(successfulHubInitialization);
			// given the managed clusters has successfully issued a join request
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith(expectedJoinCommand)))
				.resolves();
			// given the clusteradm accept command fails
			shellExecutionStub
				.withArgs('clusteradm accept --clusters dummyCluster1,dummyCluster2 --wait')
				.rejects('will-always-fail');
			return Promise.all([
				// then the build should be rejected
				expect(buildLocalEnv(fullClusterList, fakeProgressReporter))
					.to.eventually.be.rejectedWith('OCM extension, failed to accept join requests')
					.then(() => {
						// then track the progress of the progress reporter
						expect(fakeProgressReporter.getCall(0).firstArg)
							.to.contain({increment: 0 , message: 'creating 3 kind clusters'});
						expect(fakeProgressReporter.getCall(1).firstArg)
							.to.contain({increment: 20 , message: 'initializing the Hub cluster named dummyHub1'});
						expect(fakeProgressReporter.getCall(2).firstArg)
							.to.contain({increment: 20 , message: 'issuing join requests for the managed clusters'});
						expect(fakeProgressReporter.getCall(3).firstArg)
							.to.contain({increment: 20 , message: 'accepting the managed clusters join request from the hub cluster'});
						expect(fakeProgressReporter.getCall(4).firstArg)
							.to.contain({increment: 100 , message: 'failed to accept join requests'});
					}),
			]);
		});

		test('When the hub successfully accepted the join requests, the build should be resolved', async () => {
			// given kind will successfully create the clusters
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kind create cluster --name')))
				.resolves();
			// given kubectl context switching will work
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('kubectl config use')))
				.resolves();
			// given the hub cluster will be successfully initialized
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('clusteradm init --use-bootstrap-token')))
				.resolves();
			// given the hub cluster will be successfully initialized
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith('clusteradm init --use-bootstrap-token')))
				.resolves(successfulHubInitialization);
			// given the managed clusters has successfully issued a join request
			shellExecutionStub
				.withArgs(sinon.match((s: string) => s.startsWith(expectedJoinCommand)))
				.resolves();
			// given the clusteradm accept command fails
			shellExecutionStub
				.withArgs('clusteradm accept --clusters dummyCluster1,dummyCluster2 --wait')
				.resolves();
			return Promise.all([
				// then the build should be rejected
				expect(buildLocalEnv(fullClusterList, fakeProgressReporter))
					.to.eventually.equal('OCM extension, successfully created your local environment, have fun')
					.then(() => {
						// then track the progress of the progress reporter
						expect(fakeProgressReporter.getCall(0).firstArg)
							.to.contain({increment: 0 , message: 'creating 3 kind clusters'});
						expect(fakeProgressReporter.getCall(1).firstArg)
							.to.contain({increment: 20 , message: 'initializing the Hub cluster named dummyHub1'});
						expect(fakeProgressReporter.getCall(2).firstArg)
							.to.contain({increment: 20 , message: 'issuing join requests for the managed clusters'});
						expect(fakeProgressReporter.getCall(3).firstArg)
							.to.contain({increment: 20 , message: 'accepting the managed clusters join request from the hub cluster'});
						expect(fakeProgressReporter.getCall(4).firstArg)
							.to.contain({increment: 100 , message: 'successfully created your local environment, have fun'});
					}),
			]);
		});
	});
});
