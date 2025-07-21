import * as builder from '../../../src/data/builder';
import * as fixtures from './fixtures';
import * as k8s from '@kubernetes/client-node';
import * as sinon from 'sinon';
import { beforeEach } from 'mocha';
import { expect } from 'chai';

suite('Build using the data builder', () => {
	let buildSut: builder.Build;

	beforeEach(() => {
		// stub and inject the various getX methods
		let getUserStub = sinon.stub();
		getUserStub.withArgs(fixtures.k8sUser1.name).returns(fixtures.k8sUser1);
		let getClusterStub = sinon.stub();
		getClusterStub.withArgs(fixtures.k8sCluster1.name).returns(fixtures.k8sCluster1);
		let getContextStub = sinon.stub();
		getContextStub.withArgs(fixtures.k8sContext1.name).returns(fixtures.k8sContext1);

		let configMock = sinon.createStubInstance(k8s.KubeConfig, {
			getUser: getUserStub,
			// @ts-ignore
			getCluster: getClusterStub,
			// @ts-ignore
			getContextObject: getContextStub
		});
		buildSut = new builder.Build(configMock);
	});

	[
		{
			title: 'Successfully build a user from a k8s user',
			argument: fixtures.k8sUser1
		},
		{
			title: 'Successfully build a user from a user name',
			argument: fixtures.k8sUser1.name
		}
	].forEach(value => {
		test(value.title, () => {
			expect(buildSut.user(value.argument)).to.deep.equal(fixtures.connectedUser1);
		});
	});

	test('Build a user from an wrong k8s user name should return undefined', () => {
		expect(buildSut.user('unknownUser')).to.be.undefined;
	});

	[
		{
			title: 'Successfully build a cluster from a k8s cluster',
			argument: fixtures.k8sCluster1

		},
		{
			title: 'Successfully build a cluster from a cluster name',
			argument: fixtures.k8sCluster1.name
		}
	].forEach(value => {
		test(value.title, () => {
			expect(buildSut.cluster(value.argument)).to.deep.equal(fixtures.connectedCluster1);
		});
	});

	test('Build a cluster from an wrong k8s cluster name should return undefined', () => {
		expect(buildSut.cluster('unknownCluster')).to.be.undefined;
	});

	[
		{
			title: 'Successfully build a context from a k8s context',
			argument: fixtures.k8sContext1
		},
		{
			title: 'Successfully build a context from a context name',
			argument: fixtures.k8sContext1.name
		}
	].forEach(value => {
		test(value.title, () => {
			expect(buildSut.context(value.argument)).to.deep.equal(fixtures.connectedContext1);
		});
	});

	test('Build a context from an wrong k8s context name should return undefined', () => {
		expect(buildSut.context('unknownContext')).to.be.undefined;
	});

	test('Build a context with a wrong cluster name should return undefined', () => {
		expect(buildSut.context({ name: 'fakeContext', cluster: 'unknownCluster', user: 'fakeUser'})).to.be.undefined;
	});

	test('Build a context with a wrong user name should return undefined', () => {
		expect(buildSut.context({ name: 'fakeContext', cluster: 'fakeCluster', user: 'unknownUser'})).to.be.undefined;
	});
});
