import * as path from 'path';
import { runTests } from '@vscode/test-electron';

async function main(): Promise<void> {
	try {
		let extensionDevelopmentPath = path.resolve(__dirname, '../', '../');
		let extensionTestsPath = path.join(__dirname, './suite');
		let testWorkspace = path.join(extensionDevelopmentPath, "test", "test-workspace");
		await runTests({
			extensionDevelopmentPath,
			extensionTestsPath,
			launchArgs: [testWorkspace]
		});
	} catch (err) {
		console.error('Failed to run tests');
		process.exit(1);
	}
}

main();
