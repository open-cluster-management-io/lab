import * as environment from '../utils/environment';
import * as vscode from 'vscode';

// verify the required tools exists
export async function verifyTools(): Promise<void> {
	environment.verifyTools(...environment.requiredTools)
		// @ts-ignore
		.then((msg: string) => vscode.window.showInformationMessage(msg))
		.catch((msg: string[]) => vscode.window.showErrorMessage(msg[0], 'Install Instructions')
			.then(answer => {
				if (answer === 'Install Instructions') {
					vscode.env.openExternal(vscode.Uri.parse(msg[1]));
				}
			}));
}
