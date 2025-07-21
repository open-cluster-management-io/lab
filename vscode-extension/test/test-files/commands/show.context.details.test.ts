/* eslint-disable @typescript-eslint/unbound-method */
import * as distributor from '../../../src/data/distributor';
import * as fixtures from '../data/fixtures';
import * as loader from '../../../src/data/loader';
import * as sinon from 'sinon';
import * as sinonChai from 'sinon-chai';
import * as vscode from 'vscode';
import { afterEach, beforeEach } from 'mocha';
import { expect, use } from 'chai';
import { ConnectedContextWebProvider } from '../../../src/providers/contextWebProvider';

use(sinonChai);

suite('Use the web provider to render the ui panel', () => {
	let sandbox: sinon.SinonSandbox;

	beforeEach(() => {
		sinon.restore();
		sandbox = sinon.createSandbox();
	});

	afterEach(() => {
		// @ts-ignore
		loader.Load.loader = undefined;
		sandbox.restore();
	});

	test('Rendering a panel without selecting a context should not set the context', async () => {
		// stub the show quick pick
		let quickPickStub = sandbox.stub(vscode.window, 'showQuickPick');
		// wrap a spy around the error box
		let errorBoxSpy = sandbox.spy(vscode.window, 'showErrorMessage');
		// mock a loader stubbing the required methods for the test run
		let mockLoad = sandbox.createStubInstance(loader.Load, {
			getContexts: [fixtures.connectedContext1, fixtures.connectedContext2]
		});
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// @ts-ignore given the user will not select a context from selection menu
		quickPickStub.withArgs([fixtures.connectedContext1.name, fixtures.connectedContext2.name]).resolves(undefined);
		// when
		await vscode.commands.executeCommand('ocm-vscode-extension.showContextDetails');
		// then
		expect(mockLoad.setContext).to.have.not.been.called;
		// cleanup
		quickPickStub.restore();
		errorBoxSpy.restore();
	});

	test('Rendering a selected panel for an unreachable cluster should display an error message', async () => {
		// stub the show quick pick
		let quickPickStub = sandbox.stub(vscode.window, 'showQuickPick');
		// wrap a spy around the error box
		let errorBoxSpy = sandbox.spy(vscode.window, 'showErrorMessage');
		// mock a loader stubbing the required methods for the test run
		let mockLoad = sandbox.createStubInstance(loader.Load, {
			getContexts: [fixtures.connectedContext1, fixtures.connectedContext2],
			verifyReachability: Promise.reject('fake rejection message')
		});
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// @ts-ignore stub the quick pick box to return our fake context name
		quickPickStub.withArgs([fixtures.connectedContext1.name, fixtures.connectedContext2.name]).resolves(fixtures.connectedContext1.name);
		// when
		await vscode.commands.executeCommand('ocm-vscode-extension.showContextDetails');
		// then
		expect(mockLoad.setContext).to.have.been.calledOnceWith(fixtures.connectedContext1);
		expect(errorBoxSpy).to.have.been.calledOnceWith('fake rejection message');
		// cleanup
		quickPickStub.restore();
		errorBoxSpy.restore();
	});

	test('Rendering a pre-selected panel (tree view) for an unreachable cluster should display an error message', async () => {
		// wrap a spy around the error box
		let errorBoxSpy = sandbox.spy(vscode.window, 'showErrorMessage');
		// mock a loader stubbing the required methods for the test run
		let mockLoad = sandbox.createStubInstance(loader.Load, {
			verifyReachability: Promise.reject('fake rejection message')
		});
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// when
		await vscode.commands.executeCommand('ocm-vscode-extension.showContextDetails', fixtures.connectedContext2);
		// then
		expect(mockLoad.setContext).to.have.been.calledOnceWith(fixtures.connectedContext2);
		expect(errorBoxSpy).to.have.been.calledOnceWith('fake rejection message');
		// cleanup
		errorBoxSpy.restore();
	});

	test('Successfully rendering a provider should dispose the existing panel, create a new one, and invoke the message distributor', async () => {
		// @ts-ignore create a fake webview panel with a dispose method to act as the previously created panel
		let previousPanel: vscode.WebviewPanel = sandbox.stub();
		previousPanel.dispose = sandbox.fake();
		// create a fake provider for encapsulating the previous panel
		let fakeProvider = sandbox.createStubInstance(ConnectedContextWebProvider);
		// @ts-ignore inject the previous panel to the encapsulating provider
		fakeProvider.panel = previousPanel;
		// @ts-ignore inject the fake provider as the current running provider
		ConnectedContextWebProvider.currentPanel = fakeProvider;
		// mock a loader
		let mockLoad = sandbox.createStubInstance(loader.Load);
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// mock the distributor message posting function
		let distributorMock = sandbox.stub(distributor, 'distributeMessages');
		// when
		await vscode.commands.executeCommand('ocm-vscode-extension.showContextDetails', fixtures.connectedContext1);
		// collect new panel
		let newProvider = ConnectedContextWebProvider.currentPanel;
		// then
		expect(mockLoad.setContext).to.have.been.calledOnceWith(fixtures.connectedContext1);
		expect(previousPanel.dispose).to.have.been.calledOnce;
		// @ts-ignore
		expect(newProvider.panel.title).to.equal('Context Details');
		// @ts-ignore
		expect(newProvider.panel.visible).to.be.true;
		// @ts-ignore
		expect(newProvider.panel.webview.html).to.exist;
		// @ts-ignore
		expect(newProvider.panel.options.retainContextWhenHidden).to.be.true;
		// @ts-ignore
		expect(newProvider.panel.options.enableScripts).to.be.true;
		expect(distributorMock).to.have.been.calledOnceWith(fixtures.connectedContext1, sandbox.match.func);
	});

	test('Disposing the previous panel should dispose the encapsulated panel, dispose all disposables, and remove current provider', async () => {
		// mock a loader
		let mockLoad = sandbox.createStubInstance(loader.Load);
		// @ts-ignore inject a loader mock as the loader's singleton instance
		loader.Load.loader = mockLoad;
		// execute the command for creating the new provider
		await vscode.commands.executeCommand('ocm-vscode-extension.showContextDetails', fixtures.connectedContext1);
		// collect new panel
		let newProvider = ConnectedContextWebProvider.currentPanel;
		// @ts-ignore create a fake webview panel with a dispose method
		let fakePanel: vscode.WebviewPanel = sandbox.stub();
		fakePanel.dispose = sandbox.fake();
		// @ts-ignore inject the fake panel
		newProvider.panel = fakePanel;
		// @ts-ignore create a disposable with a dispose method
		let fakeDisposable: vscode.Disposable = sandbox.stub();
		fakeDisposable.dispose = sandbox.fake();
		// @ts-ignore inject the fake disposable
		newProvider.disposables.push(fakeDisposable);
		// when
		newProvider?.dispose();
		// then
		expect(fakePanel.dispose).to.have.been.calledOnce;
		expect(fakeDisposable.dispose).to.have.been.calledOnce;
		// @ts-ignore
		expect(newProvider.disposables).to.be.empty;
		expect(ConnectedContextWebProvider.currentPanel).to.be.undefined;
	});
});
