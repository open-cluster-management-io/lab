import * as fse from 'fs-extra';
import * as path from 'path';

export const availableTemplates = ['Git', 'HelmRepo', 'ObjectBucket'];
export const argoTemplates = ['Argo'];

export const defaultTemplate = 'Git';
export const defaultArgoTemplate = 'Argo';
export const defaultProjectName = 'ocm-application';
export const defaultArgoProjectName = 'ocm-argo-application';

// create a template project
export async function createProjectFromTemplate(
	workspaceFolder: string, projectName: string, templateType: string): Promise<string> {

	let projectFolder = path.join(workspaceFolder, projectName); // destination
	let templatesFolder = path.resolve(__dirname, `../../../templates/${templateType}`); // source

	return new Promise((resolve, reject) => {
		// verify project folder doesn't exists
		fse.pathExists(projectFolder)
			.then((exists: boolean) => {
				if (exists) {
					reject(`OCM extension, project folder ${projectName} exists, please use another`);
				} else {
					// create project folder
					fse.ensureDir(projectFolder)
						// copy templates to project folder
						.then(() => fse.copy(templatesFolder, projectFolder)
							.then(() => resolve(`OCM extension, project ${projectName} created`))
							.catch((reason: string) => reject(
								`OCM extension, failed creating project ${projectName}, ${reason}`)
							)
						)
						.catch((reason: string) => reject(
							`OCM extension, failed to create project folder ${projectName}, ${reason}`
						));
				}
			});
	});
}
