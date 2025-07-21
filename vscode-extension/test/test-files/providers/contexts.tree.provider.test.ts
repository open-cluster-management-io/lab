/* eslint-disable @typescript-eslint/unbound-method */
import * as contextsTreeProvider from '../../../src/providers/contextsTreeProvider';
import * as fixtures from '../data/fixtures';
import * as loader from '../../../src/data/loader';
import * as sinon from 'sinon';
import { expect, use } from 'chai';
import { beforeEach } from 'mocha';
import chaiExclude from 'chai-exclude';

use(chaiExclude);

suite('Load the tree provider', () => {
	let loadStub: loader.Load;
	let providerSut: contextsTreeProvider.ConnectedContextsTreeProvider;

	beforeEach(() => {
		// mock the loader and override the various methods with fakes and stubs
		loadStub = sinon.createStubInstance(loader.Load, {
			getContexts: [fixtures.connectedContext1, fixtures.connectedContext2],
			getCrds: Promise.resolve([fixtures.ocmCrd1Namespaced, fixtures.ocmCrd2Clustered]),
			getCrs: Promise.resolve([fixtures.ocmCr1Namespaced, fixtures.ocmCr2Clustered]) // irl two diff kinds of crs won't return together
		});
		// instantiate the provider subject under test
		providerSut = new contextsTreeProvider.ConnectedContextsTreeProvider(loadStub);
	});

	test('Instantiation of the provider should refresh the load', () => {
		expect(loadStub.refresh).to.have.been.calledOnce;
	});

	test('Hitting refresh on the provider should refresh the load', () => {
		providerSut.refresh();
		// twice bc the first call was for instantiation
		expect(loadStub.refresh).to.have.been.calledTwice;
	});

	test('Retrieving a specific element should return itself unmodified', () => {
		let fakeElement = sinon.fake();
		// @ts-ignore
		expect(providerSut.getTreeItem(fakeElement)).to.equal(fakeElement);
	});

	test('Retrieving top level children elements should return the contexts', async () => {
		let treeContexts = await providerSut.getChildren();
		expect(treeContexts).to.have.lengthOf(2);
		expect(treeContexts[0]).excluding('iconPath').to.deep.equal({
			collapsibleState: 1,
			label: fixtures.connectedContext1.name,
			context: fixtures.connectedContext1,
			tooltip: fixtures.connectedContext1.cluster.name,
			command: {
				title: 'Context Info',
				command: 'ocm-vscode-extension.showContextDetails',
				arguments: [fixtures.connectedContext1]
			}
		});
		expect(treeContexts[1]).excluding('iconPath').to.deep.equal({
			collapsibleState: 1,
			label: fixtures.connectedContext2.name,
			context: fixtures.connectedContext2,
			tooltip: fixtures.connectedContext2.cluster.name,
			command: {
				title: 'Context Info',
				command: 'ocm-vscode-extension.showContextDetails',
				arguments: [fixtures.connectedContext2]
			}
		});
	});

	test('Retrieving children for a context element should return the crds', async () => {
		let treeCrds = await providerSut.getChildren(new contextsTreeProvider.TreeContext(fixtures.connectedContext1));
		expect(loadStub.setContext).to.have.been.calledOnceWith(fixtures.connectedContext1);
		expect(loadStub.getCrds).to.have.been.calledOnceWith();
		expect(treeCrds).to.have.lengthOf(2);
		expect(treeCrds[0]).excluding('iconPath').to.deep.equal({
			collapsibleState: 1,
			label: fixtures.ocmCrd1Namespaced.kind,
			crd: fixtures.ocmCrd1Namespaced,
			tooltip: fixtures.ocmCrd1Namespaced.group
		});
		expect(treeCrds[1]).excluding('iconPath').to.deep.equal({
			collapsibleState: 1,
			label: fixtures.ocmCrd2Clustered.kind,
			crd: fixtures.ocmCrd2Clustered,
			tooltip: fixtures.ocmCrd2Clustered.group
		});
	});

	test('Retrieving children for a crd element should return the crs', async () => {
		let treeCrs = await providerSut.getChildren(new contextsTreeProvider.TreeCrd(fixtures.ocmCrd1Namespaced));
		expect(loadStub.getCrs).to.have.been.calledOnceWith(fixtures.ocmCrd1Namespaced);
		expect(treeCrs).to.have.lengthOf(2);
		expect(treeCrs[0]).to.deep.equal({
			collapsibleState: 0,
			label: fixtures.ocmCr1Namespaced.name,
			cr: fixtures.ocmCr1Namespaced,
			tooltip: fixtures.ocmCr1Namespaced.crd.version
		});
		expect(treeCrs[1]).to.deep.equal({
			collapsibleState: 0,
			label: fixtures.ocmCr2Clustered.name,
			cr: fixtures.ocmCr2Clustered,
			tooltip: fixtures.ocmCr2Clustered.crd.version
		});
	});
});
