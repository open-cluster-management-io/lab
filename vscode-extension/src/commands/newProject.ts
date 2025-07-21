
import * as filesystem from '../utils/filesystem';
import * as vscode from 'vscode';

// create a template project based on the user input
export async function create(): Promise<void> {
	// get template type from the user
	let templateType: string = await vscode.window.showQuickPick(
		filesystem.availableTemplates, {
			placeHolder: `template type, default: ${filesystem.defaultTemplate}`,
		}) || filesystem.defaultTemplate;

	// get the project name from the user
	let projectName: string = await vscode.window.showInputBox({
		placeHolder: `insert project name, default: ${filesystem.defaultProjectName}`,
	}) || filesystem.defaultProjectName;

	// verify inside workspace
	if (vscode.workspace.workspaceFolders === undefined) {
		vscode.window.showWarningMessage(
			'OCM extension, no workspace folder, please open a project or create a workspace');
		return;
	}
	// prepare project folder path
	let workspaceFolder: string = vscode.workspace.workspaceFolders[0].uri.path;
	if (process.platform === 'win32') {
		// the workspaceFolder is a uri path, it includes the initial forward slash
		// behind the scenes, this is used to start at root, but this breaks for windows
		workspaceFolder = workspaceFolder.substring(1);
	}

	// create the project
	filesystem.createProjectFromTemplate(workspaceFolder, projectName, templateType)
		.then((msg: string) => vscode.window.showInformationMessage(msg))
		.catch((msg: string) => vscode.window.showErrorMessage(msg));
}
