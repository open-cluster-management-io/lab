import * as argoProject from './commands/argoProject';
import * as createEnvironment from './commands/createEnvironment';
import * as newProject from './commands/newProject';
import * as verifyEnvironment from './commands/verifyEnvironment';
import * as vscode from 'vscode';
import { ConnectedContext } from './data/builder';
import { ConnectedContextWebProvider } from './providers/contextWebProvider';
import { ConnectedContextsTreeProvider } from './providers/contextsTreeProvider';
import { Load } from './data/loader';

export function activate(extensionContext: vscode.ExtensionContext): void {
	let dataLoader = Load.getLoader();
	let connectedContextsTreeProvider = new ConnectedContextsTreeProvider(dataLoader);
	extensionContext.subscriptions.push(
		vscode.commands.registerCommand(
			'ocm-vscode-extension.ocmNewProject', () => newProject.create()),
		vscode.commands.registerCommand(
			'ocm-vscode-extension.argoProject', () => argoProject.create()),	
		vscode.commands.registerCommand(
			'ocm-vscode-extension.verifyTools', () => verifyEnvironment.verifyTools()),
		vscode.commands.registerCommand(
			'ocm-vscode-extension.createLocalEnvironment', () => createEnvironment.createLocalEnvironment()),
		vscode.window.registerTreeDataProvider(
			'ocm-vscode-extension.connectedContextsView', connectedContextsTreeProvider),
		vscode.commands.registerCommand(
			'ocm-vscode-extension.connectedContextsView.refresh', () => connectedContextsTreeProvider.refresh()),
		vscode.commands.registerCommand(
			'ocm-vscode-extension.showContextDetails',
			(selectedContext: ConnectedContext) => ConnectedContextWebProvider.render(extensionContext.extensionUri, selectedContext))
	);
}
