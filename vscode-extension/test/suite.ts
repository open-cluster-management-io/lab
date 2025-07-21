import * as Mocha from 'mocha';
import * as glob from 'glob';
import * as path from 'path';

function wrapCoverage(): any {
	// eslint-disable-next-line @typescript-eslint/no-var-requires
	let nyc = new (require('nyc'))({
		all: true,
		cwd: path.join(__dirname, '..', '..'),
		exclude: [".vscode-test"],
		hookRequire: true,
		exitOnError: true,
	});

	nyc.reset();
	nyc.wrap();
}

export function run(): Promise<void> {
	wrapCoverage();

	let mochaOpts: Mocha.MochaOptions = {
		ui: 'tdd',
		color: true,
		slow: 1500,
		timeout: 5000,
		bail: true,
		fullTrace: true,
	};

	if ('true' === process?.env?.QUICK_TEST) {
		mochaOpts.fgrep = '@slow';
		mochaOpts.invert = true;
	}

	let mocha = new Mocha(mochaOpts);
	let testsRoot = path.join(__dirname, 'test-files');

	return new Promise((resolve, reject) => {
		glob('**/**.test.js', { cwd: testsRoot }, (err: Error | null, files: string[]) => {
			if (err) {
				return reject(err);
			}

			files.forEach(f => mocha.addFile(path.resolve(testsRoot, f)));

			try {
				mocha.run(failures => {
					if (failures > 0) {
						reject(new Error(`${failures} tests failed.`));
					} else {
						resolve();
					}
				});
			} catch (err) {
				console.error(err);
				reject(err);
			}
		});
	});
}
