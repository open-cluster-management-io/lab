import * as environmentTools from '../../../src/utils/environment';
import * as sinon from 'sinon';
import * as sinonChai from 'sinon-chai';
import * as vscode from 'vscode';
import { use as chaiUse, expect } from 'chai';
import { beforeEach } from 'mocha';

chaiUse(sinonChai);

// Test cases for the the ocm-vscode-extension.verifyTools command
suite('Verify installed developer tools command', () => {
	beforeEach(() => sinon.restore()); // unwrap previously wrapped sinon objects

	test('When all the required tools are installed, the command should end successfully', async () => {
		// wrap a spy around the information box
		let infoBoxSpy = sinon.spy(vscode.window, 'showInformationMessage');
		// given all the required tools are installed
		sinon.stub(environmentTools, 'verifyTools')
			.withArgs(...environmentTools.requiredTools)
			.resolves('whoosh, found all tools');
		// when invoking the command
		await vscode.commands.executeCommand('ocm-vscode-extension.verifyTools');
		// then expect info message to be called
		expect(infoBoxSpy).to.be.calledOnceWith('whoosh, found all tools');
	});

	test('When a tool is missing, the command should fail while providing an installation instructions link', async () => {
		let fakeToolMsg = 'fake-tool is missing, please install';
		let fakeToolUrl = 'http://instructions.to.installation';
		// @ts-ignore stub the error box and resolve to the Install Instructions action
		let errorBoxStub = sinon.stub(vscode.window, 'showErrorMessage').resolves('Install Instructions');
		// stub the open external function for verifying url call
		let openExternalStub = sinon.stub(vscode.env, 'openExternal').resolves();
		// given all the required tools are installed
		sinon.stub(environmentTools, 'verifyTools')
			.withArgs(...environmentTools.requiredTools)
			.rejects([fakeToolMsg, fakeToolUrl]);
		// when invoking the command
		await vscode.commands.executeCommand('ocm-vscode-extension.verifyTools');
		// then expect info message to be called
		expect(errorBoxStub).to.be.calledOnceWith(fakeToolMsg, 'Install Instructions');
		expect(openExternalStub).to.be.calledOnceWith(
			sinon.match(arg =>
				arg['scheme'] === 'http' &&
				arg['authority'] === 'instructions.to.installation' &&
				arg['path'] === '/'));
	});
});
