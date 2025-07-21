import * as chaiAsPromised from 'chai-as-promised';
import * as chaiThings from 'chai-things';
import * as fse from 'fs-extra';
import * as path from 'path';
import * as sinon from 'sinon';
import * as sinonChai from 'sinon-chai';
import * as vscode from 'vscode';
import * as yaml from 'js-yaml';
import { use as chaiUse, expect } from 'chai';
import { beforeEach } from 'mocha';

chaiUse(chaiAsPromised);
chaiUse(sinonChai);
chaiUse(chaiThings);

interface ExpectedTemplate {
	channelType: string,
	verifySubscription: CallableFunction
}

async function sleep(ms: number): Promise<void> {
	return new Promise((resolve, _reject) => setTimeout(() => resolve(), ms));
}

// Test cases for the the ocm-vscode-extension.ocmNewProject command
suite('Create a new argo project command', () => {
	let quickPickStub: sinon.SinonStub;
	let projectCreationDelayMS = 500;

	// expected template files
	const expectedTemplateFiles = [
		'00-namespaces.yaml',
		'README.md',
		'application-set.yaml',
		'argo-configmap.yaml',
		'role.yaml',
		'placement.yaml',
		'rolebinding.yaml'
	];


	beforeEach(() => {
		sinon.restore(); // unwrap previously wrapped sinon objects
		quickPickStub = sinon.stub(vscode.window, 'showQuickPick'); // stub the show quick pick
	});



	test('Successfully create a project with the default name and type', async () => {
		// wrap a spy around the information box
		let infoBoxSpy = sinon.spy(vscode.window, 'showInformationMessage');
		// given the default path
		let projectFolder: string = path.resolve(__dirname, '../../../../test/test-workspace/ocm-argo-application');
		// given the path doesn't already exists
		await fse.remove(projectFolder);
		// given the user will not input a project name (type enter)
		sinon.stub(vscode.window, 'showInputBox').resolves('');
		// when invoking the command
		await vscode.commands.executeCommand('ocm-vscode-extension.argoProject');
		await sleep(projectCreationDelayMS); // wait a sec
		// the grab the application-set resource file
		let applicationSet = yaml.load(await fse.readFile(`${projectFolder}/application-set.yaml`, 'utf-8'));
		return Promise.all([
			// then expect the following
			expect(fse.pathExists(projectFolder)).to.eventually.be.true,
			expect(fse.readdir(projectFolder)).to.eventually.have.members(expectedTemplateFiles),
			expect(infoBoxSpy).to.have.been.calledOnceWith('OCM extension, project ocm-argo-application created'),
			expect(applicationSet).to.have.property('kind').that.equals('ApplicationSet'),
			expect(applicationSet).to.have.property('spec').that.has.a.property('template'),
			expect(applicationSet).to.have.property('spec').that.has.a.property('generators')
		]);
	});

	test('Fail creating a new project when the folder already exists', async () => {
		// wrap a spy over vscode's error message box
		let errorBoxSpy = sinon.spy(vscode.window, 'showErrorMessage');
		// given the following project name and path
		let projectNameInput = 'existing-folder-name';
		let projectFolder: string = path.resolve(__dirname, `../../../../test/test-workspace/${projectNameInput}`);
		// given the folder already exists (with no files in it)
		await fse.emptyDir(projectFolder);
		// given the user will input the project name as the existing folder
		sinon.stub(vscode.window, 'showInputBox').resolves(projectNameInput);
		// when invoking the command
		await vscode.commands.executeCommand('ocm-vscode-extension.argoProject');
		await sleep(projectCreationDelayMS); // wait a sec
		return Promise.all([
			// then expect the following
			expect(fse.pathExists(projectFolder)).to.eventually.be.true,
			expect(fse.readdir(projectFolder)).to.eventually.be.empty,
			expect(errorBoxSpy).to.have.been.calledWith(
				`OCM extension, project folder ${projectNameInput} exists, please use another`
			)
		]);
	});

	test('Fail creating a new project when not in a workspace', async () => {
		// wrap a spy over vscode's warning message box
		let warningBoxSpy = sinon.spy(vscode.window, 'showWarningMessage');
		// given the following project name and path
		let projectNameInput = 'non-existing-folder-name';
		let projectFolder: string = path.resolve(__dirname, `../../../../test/test-workspace/${projectNameInput}`);
		// given the path doesn't already exists
		await fse.remove(projectFolder);
		// given the user will input the project name as the existing folder
		sinon.stub(vscode.window, 'showInputBox').resolves(projectNameInput);
		// given the workspace api will return undefined workspaceFolders
		sinon.stub(vscode.workspace, 'workspaceFolders').value(undefined);
		// when invoking the command
		await vscode.commands.executeCommand('ocm-vscode-extension.argoProject');
		return Promise.all([
			// then expect the following
			expect(fse.pathExists(projectFolder)).to.eventually.be.false,
			expect(warningBoxSpy).to.have.been.calledWith(
				'OCM extension, no workspace folder, please open a project or create a workspace')
		]);
	});
});




