/* eslint-disable @typescript-eslint/unbound-method */
import * as distributor from '../../../src/data/distributor';
import * as fixtures from './fixtures';
import * as loader from '../../../src/data/loader';
import * as sinon from 'sinon';
import * as sinonChai from 'sinon-chai';
import { afterEach, beforeEach } from 'mocha';
import { expect, use } from 'chai';

use(sinonChai);

suite('Distribute messages using the data distributor', () => {
	let sandbox: sinon.SinonSandbox;
	let mockConsumer: sinon.SinonStub;

	beforeEach(() => {
		sinon.restore();
		sandbox = sinon.createSandbox();
		mockConsumer = sandbox.stub();
	});

	afterEach(() => {
		// @ts-ignore
		loader.Load.loader = undefined;
		sandbox.restore();
	});

	test('When the context is not a hub nor a spoke, should only distribute a context info message', async () => {
		// mock a loader instance
		let mockLoad = sandbox.createStubInstance(loader.Load);
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// given the cluster in context is not a hub (no ManagedCluster crs)
		mockLoad.getCrs.withArgs('ManagedCluster').resolves([]);
		// given the cluster in context is not a spoke (no Klusterlet crs)
		mockLoad.getCrs.withArgs('Klusterlet').resolves([]);
		// when
		await distributor.distributeMessages(fixtures.connectedContext1, mockConsumer);
		// then
		expect(mockConsumer).to.be.calledOnceWith({selectedContext: JSON.stringify(fixtures.connectedContext1)});
	});

	test('When the context is a hub but has no other resources, should distribute only context info and ManagedCluster messages', async () => {
		// mock a loader instance
		let mockLoad = sandbox.createStubInstance(loader.Load);
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// given the cluster in context is a hub (ManagedCluster crs exist)
		mockLoad.getCrs.withArgs('ManagedCluster').resolves([fixtures.ocmCr2Clustered]);
		// given the cluster in context is not a spoke (no Klusterlet crs)
		mockLoad.getCrs.withArgs('Klusterlet').resolves([]);
		// when
		await distributor.distributeMessages(fixtures.connectedContext1, mockConsumer);
		// then
		expect(mockConsumer).to.have.callCount(2);
		expect(mockConsumer).to.be.calledWith({selectedContext: JSON.stringify(fixtures.connectedContext1)});
		expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: 'ManagedCluster', crs: JSON.stringify([fixtures.ocmCr2Clustered])}});
	});

	['ManifestWork', 'Placement', 'PlacementDecision', 'ManagedClusterSet', 'ManagedClusterAddOn', 'ClusterManager', 'SubscriptionReport']
	.forEach(kind => {
		test(`When the context is a hub with existing ${kind} crs, should distribute context info, ManagedCluster, and ${kind} messages`, async () => {
			// mock a loader instance
			let mockLoad = sandbox.createStubInstance(loader.Load);
			// @ts-ignore inject a loader mock as the loader's singleton instance
			loader.Load.loader = mockLoad;
			// given the cluster in context is a hub (ManagedCluster crs exist)
			mockLoad.getCrs.withArgs('ManagedCluster').resolves([fixtures.ocmCr2Clustered]);
			// given the cluster in context is not a spoke (no Klusterlet crs)
			mockLoad.getCrs.withArgs('Klusterlet').resolves([]);
			// given the kind under test has resources
			mockLoad.getCrs.withArgs(kind).resolves([fixtures.ocmCr1Namespaced]);
			// when
			await distributor.distributeMessages(fixtures.connectedContext1, mockConsumer);
			// then
			expect(mockConsumer).to.have.callCount(3);
			expect(mockConsumer).to.be.calledWith({selectedContext: JSON.stringify(fixtures.connectedContext1)});
			expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: 'ManagedCluster', crs: JSON.stringify([fixtures.ocmCr2Clustered])}});
			expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: kind, crs: JSON.stringify([fixtures.ocmCr1Namespaced])}});
		});
	});

	test('When the context is a spoke but has no other resources, should distribute only context info and klusterlet messages', async () => {
		// mock a loader instance
		let mockLoad = sandbox.createStubInstance(loader.Load);
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// given the cluster in context is not a hub (no ManagedCluster crs)
		mockLoad.getCrs.withArgs('ManagedCluster').resolves([]);
		// given the cluster in context is a spoke (Klusterlet crs exists)
		mockLoad.getCrs.withArgs('Klusterlet').resolves([fixtures.ocmCr2Clustered]);
		// when
		await distributor.distributeMessages(fixtures.connectedContext1, mockConsumer);
		// then
		expect(mockConsumer).to.have.callCount(2);
		expect(mockConsumer).to.be.calledWith({selectedContext: JSON.stringify(fixtures.connectedContext1)});
		expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: 'Klusterlet', crs: JSON.stringify([fixtures.ocmCr2Clustered])}});
	});

	['AppliedManifestWork', 'SubscriptionStatus']
	.forEach(kind => {
		test(`When the context is a spoke with existing ${kind} crs, should distribute context info, Klusterlet, and ${kind} messages`, async () => {
			// mock a loader instance
			let mockLoad = sandbox.createStubInstance(loader.Load);
			// @ts-ignore inject a loader mock as the loader's singleton instance
			loader.Load.loader = mockLoad;
			// given the cluster in context is not a hub (ManagedCluster crs exist)
			mockLoad.getCrs.withArgs('ManagedCluster').resolves([]);
			// given the cluster in context is a spoke (no Klusterlet crs)
			mockLoad.getCrs.withArgs('Klusterlet').resolves([fixtures.ocmCr2Clustered]);
			// given the kind under test has resources
			mockLoad.getCrs.withArgs(kind).resolves([fixtures.ocmCr1Namespaced]);
			// when
			await distributor.distributeMessages(fixtures.connectedContext2, mockConsumer);
			// then
			expect(mockConsumer).to.have.callCount(3);
			expect(mockConsumer).to.be.calledWith({selectedContext: JSON.stringify(fixtures.connectedContext2)});
			expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: 'Klusterlet', crs: JSON.stringify([fixtures.ocmCr2Clustered])}});
			expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: kind, crs: JSON.stringify([fixtures.ocmCr1Namespaced])}});
		});
	});

	test('When the context is both a hub and a spoke, should distribute context info, ManagedCluster, and Klusterlet messages', async () => {
		// mock a loader instance
		let mockLoad = sandbox.createStubInstance(loader.Load);
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// given the cluster in context is a hub (no ManagedCluster crs)
		mockLoad.getCrs.withArgs('ManagedCluster').resolves([fixtures.ocmCr2Clustered]);
		// given the cluster in context is a spoke (no Klusterlet crs)
		mockLoad.getCrs.withArgs('Klusterlet').resolves([fixtures.ocmCr2Clustered]);
		// when
		await distributor.distributeMessages(fixtures.connectedContext1, mockConsumer);
		// then
		expect(mockConsumer).to.have.callCount(3);
		expect(mockConsumer).to.be.calledWith({selectedContext: JSON.stringify(fixtures.connectedContext1)});
		expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: 'ManagedCluster', crs: JSON.stringify([fixtures.ocmCr2Clustered])}});
		expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: 'Klusterlet', crs: JSON.stringify([fixtures.ocmCr2Clustered])}});
	});

	test('When no crs found for kind, should not send message for kind', async () => {
		// mock a loader instance
		let mockLoad = sandbox.createStubInstance(loader.Load);
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// given the cluster in context is not a hub (ManagedCluster crs exist)
		mockLoad.getCrs.withArgs('ManagedCluster').resolves([]);
		// given the cluster in context is a spoke (no Klusterlet crs)
		mockLoad.getCrs.withArgs('Klusterlet').resolves([fixtures.ocmCr2Clustered]);
		// given the AppliedManifestWork has no resources
		mockLoad.getCrs.withArgs('AppliedManifestWork').resolves([]);
		// when
		await distributor.distributeMessages(fixtures.connectedContext2, mockConsumer);
		// then
		expect(mockConsumer).to.have.callCount(2);
		expect(mockConsumer).to.be.calledWith({selectedContext: JSON.stringify(fixtures.connectedContext2)});
		expect(mockConsumer).to.be.calledWith({ crsDistribution: { kind: 'Klusterlet', crs: JSON.stringify([fixtures.ocmCr2Clustered])}});
	});
});
