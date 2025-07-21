import * as chaiAsPromised from 'chai-as-promised';
import * as shellUtils from '../../../src/utils/shell';
import * as sinon from 'sinon';
import { RequiredTool, verifyTools } from '../../../src/utils/environment';
import { use as chaiUse, expect } from 'chai';
import { beforeEach } from 'mocha';

chaiUse(chaiAsPromised);

suite('Test cases for the environment utility functions', () => {
	beforeEach(() => sinon.restore()); // unwrap previously wrapped sinon objects

	suite('Testing verifyTools', () => {
		// dummy tools used for testing purposes
		const dummyTool1: RequiredTool = {
			'name': 'find-wally',
			'installUrl': 'https://you.can/find/install/instructions/#here'
		};

		const dummyTool2: RequiredTool = {
			'name': 'kill-wally',
			'installUrl': 'https://we.dont/have/install/instructions/#forthat'
		};

		test('When verifying with one existing tool, the function should be successful', async () => {
			// given the shell utility will resolve the request, indicating the tool was found
			sinon.stub(shellUtils, 'checkToolExists').withArgs(dummyTool1.name).resolves();
			// then expect the promise to be resolved with the appropriate message
			return expect(verifyTools(dummyTool1)).to.eventually.be.equal(
				'OCM extension, all tools are accessible, we\'re good to go'
			);
		});

		test('When verifying with one non-existing tool, the function should fail', async () => {
			// given the shell utility will reject the request, indicating the for tool was NOT found
			sinon.stub(shellUtils, 'checkToolExists').withArgs(dummyTool2.name).rejects();
			// @ts-ignore then expect the promise to be rejected with the missing tool info
			return expect(verifyTools(dummyTool2)).to.eventually.be.rejectedWith([
				`OCM extension, ${dummyTool2.name} is missing, please install it`,
				dummyTool2.installUrl
			]);
		});

		test('When verifying with two existing tools, the function should be successful', async () => {
			// given the shell utility will resolve for both requests, indicating the both tools were found
			let checkToolExistsStub = sinon.stub(shellUtils, 'checkToolExists');
			checkToolExistsStub.withArgs(dummyTool1.name).resolves();
			checkToolExistsStub.withArgs(dummyTool2.name).resolves();
			// then expect the promise to be resolved, the and the message consumers to be called accordingly
			return expect(verifyTools(...[dummyTool1, dummyTool2])).to.eventually.be.equal(
				'OCM extension, all tools are accessible, we\'re good to go'
			);
		});

		test('When verifying with two tools, but only one exists, the function should fail', async () => {
			// given the shell utility will reject one request and resolve the other, indicating only one tool was found
			let checkToolExistsStub = sinon.stub(shellUtils, 'checkToolExists');
			checkToolExistsStub.withArgs(dummyTool1.name).rejects();
			checkToolExistsStub.withArgs(dummyTool2.name).resolves();
			// @ts-ignore then expect the promise to be rejected, the and the message consumers to be called accordingly
			return expect(verifyTools(...[dummyTool1, dummyTool2])).to.eventually.be.rejectedWith([
				`OCM extension, ${dummyTool1.name} is missing, please install it`,
				dummyTool2.installUrl
			]);
		});
	});
});
