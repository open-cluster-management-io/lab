
import * as shell from 'shelljs';

shell.config.execPath =  String(shell.which('node'));

// execute a command and return a promise of the output as string
export function executeShellCommand(command: string): Promise<string> {
	return new Promise((resolve, reject) => {
		shell.exec(command, (code, stdout, stderr) => {
			if (code === 0) {
				resolve(stdout);
			}
			reject(stderr);
		});
	});
}

// check if a cli tool exists and return a promise
export  function checkToolExists(tool: string): Promise<void> {
	return new Promise((resolve, reject) => {
		if (shell.which(tool)?.code === 0) {
			resolve();
		}
		reject();
	});
}
