import * as buildTools from '../../../src/utils/build';
import * as environmentTools from '../../../src/utils/environment';
import * as sinon from 'sinon';
import * as sinonChai from 'sinon-chai';
import * as vscode from 'vscode';
import { use as chaiUse, expect } from 'chai';
import { YesNo } from '../../../src/commands/createEnvironment';
import { beforeEach } from 'mocha';

chaiUse(sinonChai);

async function sleep(ms: number): Promise<void> {
	return new Promise((res, _rej) => setTimeout(() => res(), ms));
}

// Test cases for the the ocm-vscode-extension.createLocalEnvironment command
suite('Create local environment command', () => {
	const fakeBuildSuccessMsg = 'this is a fake message';
	const buildEnvironmentDelayMS = 2000;

	let quickPickStub: sinon.SinonStub;
	let inputBoxStub: sinon.SinonStub;
	let infoBoxSpy: sinon.SinonSpy;

	const matchDefaultConfigQuickPick = sinon.match(
		op => op['title'] === 'Use default configuration, 1 hub and 2 managed clusters?');

	beforeEach(() => {
		sinon.restore(); // unwrap previously wrapped sinon objects
		quickPickStub = sinon.stub(vscode.window, 'showQuickPick'); // stub the show quick pick
		inputBoxStub = sinon.stub(vscode.window, 'showInputBox'); // stub the input box
		infoBoxSpy = sinon.spy(vscode.window, 'showInformationMessage'); // wrap a spy around the information box
	});

	suite('Using the default configuration @slow', () => {
		let errorBoxSpy: sinon.SinonSpy;

		beforeEach(() => {
			// given the the user will choose to use the default configuration
			quickPickStub.withArgs([YesNo.yes, YesNo.no], matchDefaultConfigQuickPick).resolves(YesNo.yes);
			errorBoxSpy = sinon.spy(vscode.window, 'showErrorMessage'); // wrap a spy around the error box
		});

		test('Selecting to build with the default configuration, the command should be successful', async () => {
			// given all the required tools are installed
			sinon.stub(environmentTools, 'verifyTools').withArgs(...environmentTools.requiredTools).resolves();
			// given the build tool utility function will be resolved with a fake message
			let buildLocalEnvSpy = sinon.stub(buildTools, 'buildLocalEnv').resolves(fakeBuildSuccessMsg);
			// when invoking the command
			vscode.commands.executeCommand('ocm-vscode-extension.createLocalEnvironment');
			await sleep(buildEnvironmentDelayMS); // wait a sec
			// then expect info message to be called
			expect(infoBoxSpy).to.be.calledOnceWith(fakeBuildSuccessMsg);
			// then expect the build environment utility function will be invoked with default configuration
			expect(buildLocalEnvSpy).to.be.calledOnceWith(
				buildTools.defaultClusters,
				sinon.match.func
			);
		});

		test('When the required tools for building the environment are missing, the command should fail', async () => {
			// given kind and clusteradm are missing
			sinon.stub(environmentTools, 'verifyTools').withArgs(...environmentTools.requiredTools).rejects();
			// when invoking the command
			vscode.commands.executeCommand('ocm-vscode-extension.createLocalEnvironment');
			await sleep(buildEnvironmentDelayMS); // wait a sec
			// the expect a failure message
			expect(errorBoxSpy).to.be.calledWith('OCM extension, unable to verify the existence of the required tools');
		});

		test('When the required tools are present but the build fails, the command should fail', async () => {
			// given all the required tools are installed
			sinon.stub(environmentTools, 'verifyTools').withArgs(...environmentTools.requiredTools).resolves();
			// given the build tool utility function will be rejected with a fake message
			sinon.stub(buildTools, 'buildLocalEnv').rejects('oops try again');
			// when invoking the command
			vscode.commands.executeCommand('ocm-vscode-extension.createLocalEnvironment');
			await sleep(buildEnvironmentDelayMS); // wait a sec
			//the expect a failure message
			expect(errorBoxSpy).to.be.calledOnceWith('oops try again');
		});
	});

	suite('Using a custom configuration @slow', () => {
		const fakeHubName = 'my-fake-hub';
		const numManagedClusters = 3;
		const fakeManage1Name = `my-managed-cluster-1`;
		const fakeManage2Name = `my-managed-cluster-2`;
		const fakeManage3Name = `my-managed-cluster-3`;

		let buildLocalEnvSpy: sinon.SinonSpy;

		const matchHubNameInputBox = sinon.match(op =>op['title'] === 'Choose a name for the hub cluster');
		const matchNumManagedInputBox = sinon.match(op => op['title'] === 'Choose the number of managed clusters to be created');
		const matchUseDefaultNameForManagedInputBox = sinon.match(op => op['title'] === 'Name managed clusters by index (clusterX)?');
		const matchCluster1NameInputBox = sinon.match(op => op['title'] === 'Choose a name for managed cluster number 1');
		const matchCluster2NameInputBox = sinon.match(op => op['title'] === 'Choose a name for managed cluster number 2');
		const matchCluster3NameInputBox = sinon.match(op => op['title'] === 'Choose a name for managed cluster number 3');

		beforeEach(() => {
			// given the the user will choose NOT to use the default configuration
			quickPickStub.withArgs([YesNo.yes, YesNo.no], matchDefaultConfigQuickPick).resolves(YesNo.no);
			// given all the required tools are installed
			sinon.stub(environmentTools, 'verifyTools').withArgs(...environmentTools.requiredTools).resolves();
			// given the build tool utility function will be resolved with a fake message
			buildLocalEnvSpy = sinon.stub(buildTools, 'buildLocalEnv').resolves(fakeBuildSuccessMsg);
		});

		test('Create hub cluster with a custom name and 3 managed clusters using defaults names', async () => {
			// given the user will input a custom hub name
			inputBoxStub.withArgs(matchHubNameInputBox).resolves(fakeHubName);
			// given the user will select the amount of desired managed clusters
			inputBoxStub.withArgs(matchNumManagedInputBox).resolves(numManagedClusters);
			// given the user will choose to use the default naming convention for the managed clusters
			quickPickStub.withArgs([YesNo.yes, YesNo.no], matchUseDefaultNameForManagedInputBox).resolves(YesNo.yes);
			// when invoking the command
			vscode.commands.executeCommand('ocm-vscode-extension.createLocalEnvironment');
			await sleep(buildEnvironmentDelayMS); // wait a sec
			// then expect info message to be called
			expect(infoBoxSpy).to.be.calledOnceWith(fakeBuildSuccessMsg);
			// then expect the build environment utility function will be invoked with the expected configuration
			expect(buildLocalEnvSpy).to.be.calledOnceWith(
				[
					{
						name: fakeHubName,
						context: `kind-${fakeHubName}`,
						type: buildTools.ClusterType.hub
					},
					{
						name: 'cluster1',
						context: 'kind-cluster1',
						type: buildTools.ClusterType.managed
					},
					{
						name: 'cluster2',
						context: 'kind-cluster2',
						type: buildTools.ClusterType.managed
					},
					{
						name: 'cluster3',
						context: 'kind-cluster3',
						type: buildTools.ClusterType.managed
					},
				],
				sinon.match.func
			);
		});

		test('Create hub cluster with a custom name and 3 managed clusters using custom names', async () => {
			// given the user will input a custom hub name
			inputBoxStub.withArgs(matchHubNameInputBox).resolves(fakeHubName);
			// given the user will select the amount of desired managed clusters
			inputBoxStub.withArgs(matchNumManagedInputBox).resolves(numManagedClusters);
			// given the user will choose NOT to use the default naming convention for the managed clusters
			quickPickStub.withArgs([YesNo.yes, YesNo.no], matchUseDefaultNameForManagedInputBox).resolves(YesNo.no);
			// given the user will input a custom name for each desired managed cluster
			inputBoxStub.withArgs(matchCluster1NameInputBox).resolves(fakeManage1Name);
			inputBoxStub.withArgs(matchCluster2NameInputBox).resolves(fakeManage2Name);
			inputBoxStub.withArgs(matchCluster3NameInputBox).resolves(fakeManage3Name);
			// when invoking the command
			vscode.commands.executeCommand('ocm-vscode-extension.createLocalEnvironment');
			await sleep(buildEnvironmentDelayMS); // wait a sec
			// then expect info message to be called
			expect(infoBoxSpy).to.be.calledOnceWith(fakeBuildSuccessMsg);
			// then expect the build environment utility function will be invoked with the expected configuration
			expect(buildLocalEnvSpy).to.be.calledOnceWith(
				[
					{
						name: fakeHubName,
						context: `kind-${fakeHubName}`,
						type: buildTools.ClusterType.hub
					},
					{
						name: fakeManage1Name,
						context: `kind-${fakeManage1Name}`,
						type: buildTools.ClusterType.managed
					},
					{
						name: fakeManage2Name,
						context: `kind-${fakeManage2Name}`,
						type: buildTools.ClusterType.managed
					},
					{
						name: fakeManage3Name,
						context: `kind-${fakeManage3Name}`,
						type: buildTools.ClusterType.managed
					},
				],
				sinon.match.func
			);
		});
	});
});
