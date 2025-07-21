import * as chaiAsPromised from 'chai-as-promised';
import * as shell from 'shelljs';
import * as sinon from 'sinon';
import { use as chaiUse, expect } from 'chai';
import { checkToolExists, executeShellCommand } from '../../../src/utils/shell';
import { beforeEach } from 'mocha';

chaiUse(chaiAsPromised);

suite('Test cases for the shell utility functions', () => {
	beforeEach(() => sinon.restore()); // unwrap any previously wrapped sinon objects

	suite('Testing checkToolExists', () => {
		test('When checking with an existing tool, the function should be successful', async () => {
			// @ts-ignore given the command in question exists
			sinon.stub(shell, 'which').withArgs('existing-tool').returns({code: 0});
			// then expect the promise to be resolved
			return expect(checkToolExists('existing-tool')).to.eventually.be.fulfilled;
		});

		test('When checking with a non-existing tool, the function should fail', async () => {
			// given the command in question doesn't exist
			sinon.stub(shell, 'which').withArgs('non-existing-tool').returns(null);
			// then expect the promise to be rejected
			return expect(checkToolExists('non-existing-tool')).to.eventually.be.rejected;
		});
	});

	suite('Testing executeShellCommand', () => {
		test('When executing a successful command, the function should be successful', async () => {
			let dummyCommand = 'dummy-successful-command';
			let dummyStdout = 'all is good in the hood';
			// given the dummy success execution will return for the dummy command
			sinon.stub(shell, 'exec').withArgs(dummyCommand, sinon.match.func).yields(0, dummyStdout, null);
			// then expect the promise to be resolved with the standard output from the succeeded execution
			return expect(executeShellCommand(dummyCommand)).to.eventually.be.equal(dummyStdout);
		});

		test('When executing a failed command, the function should fail', async () => {
			let dummyCommand = 'dummy-failed-command';
			let dummyStderr = 'oh my god they killed kenny';
			// given the dummy failed execution will return for the dummy command
			sinon.stub(shell, 'exec').withArgs(dummyCommand, sinon.match.func).yields(999, null, dummyStderr);
			// then expect the promise to be rejected with the error output from the failed execution
			return expect(executeShellCommand(dummyCommand)).to.eventually.be.rejectedWith(dummyStderr);
		});
	});
});
