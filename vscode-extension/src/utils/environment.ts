import * as shell from './shell';

export interface RequiredTool {
	name: string,
	installUrl: string
}

export const requiredTools: RequiredTool[] = [
	{
		'name': 'kubectl',
		'installUrl': 'https://kubernetes.io/docs/tasks/tools/#kubectl'
	},
	{
		'name': 'clusteradm',
		'installUrl': 'https://github.com/open-cluster-management-io/clusteradm#install-the-clusteradm-command-line'
	},
	{
		'name': 'kind',
		'installUrl': 'https://kind.sigs.k8s.io/docs/user/quick-start/#installation'
	}
];

// verify the the existence of the required tools in the environment's shell
// will be resolved with a string or rejected with a string[]
export async function verifyTools(...tools: RequiredTool[]): Promise<string|string[]> {
	let executionPromises: Promise<void | string>[] = tools.map(
		tool => shell.checkToolExists(tool.name).catch(
			() => Promise.reject([`OCM extension, ${tool.name} is missing, please install it`, tool.installUrl])
		)
	);
	return Promise.all(executionPromises)
		.then(() => Promise.resolve('OCM extension, all tools are accessible, we\'re good to go'));
}
