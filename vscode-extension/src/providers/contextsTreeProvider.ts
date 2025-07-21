import * as builder from '../data/builder';
import * as loader from '../data/loader';
import * as path  from 'path';
import * as vscode from 'vscode';

const LAUNCH_WEBVIEW_TITLE = 'Context Info';
const LAUNCH_WEBVIEW_CMD = 'ocm-vscode-extension.showContextDetails';
const ICON_CONTEXT = 'k8s';
const ICON_CRD = 'ocm';

type TreeElement = TreeContext | TreeCrd | TreeCr;

// tree view element for encapsulating a context representing a cluster and a user
export class TreeContext extends vscode.TreeItem {
	readonly context: builder.ConnectedContext;

	constructor(context: builder.ConnectedContext) {
		super(context.name, vscode.TreeItemCollapsibleState.Collapsed);
		this.context = context;
		this.tooltip =  context.cluster.name;
		// this command launces the webview with the encapsulated context as an argument
		this.command = {
			title: LAUNCH_WEBVIEW_TITLE,
			command: LAUNCH_WEBVIEW_CMD,
			arguments: [context]
		};
	}

	iconPath = {
		light: path.join(__dirname, '..', '..', '..', 'images', 'light', `${ICON_CONTEXT}.svg`),
		dark: path.join(__dirname, '..', '..', '..', 'images', 'dark', `${ICON_CONTEXT}.svg`),
	};
}

// tree view element for encapsulating a custom resource definition
export class TreeCrd extends vscode.TreeItem {
	readonly crd: loader.OcmResourceDefinition;

	constructor(crd: loader.OcmResourceDefinition) {
		super(crd.kind, vscode.TreeItemCollapsibleState.Collapsed);
		this.crd = crd;
		this.tooltip = crd.group;
	}

	iconPath = {
		light: path.join(__dirname, '..', '..', '..', 'images', 'light', `${ICON_CRD}.svg`),
		dark: path.join(__dirname, '..', '..', '..', 'images', 'dark', `${ICON_CRD}.svg`),
	};
}

// tree view element for encapsulating a custom resource
export class TreeCr extends vscode.TreeItem {
	readonly cr: loader.OcmResource;

	constructor(cr: loader.OcmResource) {
		super(cr.name, vscode.TreeItemCollapsibleState.None);
		this.cr = cr;
		this.tooltip = cr.crd.version;
	}
}

// implementation of tree view, provides contexts as highest entity:
// contexts > crds > crs
export class ConnectedContextsTreeProvider implements vscode.TreeDataProvider<TreeElement> {
	private load: loader.Load;

	private _onDidChangeTreeData: vscode.EventEmitter<TreeElement | undefined | null | void> = new vscode.EventEmitter<TreeElement | undefined | null | void>();
	readonly onDidChangeTreeData: vscode.Event<TreeElement | undefined | null | void> = this._onDidChangeTreeData.event;
  

	constructor(load: loader.Load) {
		this.load = load;
		this.refresh();
	}

	refresh(): void {
		this.load.refresh();
		this._onDidChangeTreeData.fire();
	}

	getTreeItem(element: TreeElement): TreeElement {
		return element;
	}

	// (highest) contexts > crds > crs (lowest)
	async getChildren(element?: TreeElement): Promise<TreeElement[]> {
		let elements: TreeElement[] = [];
		if (element) {
			if (element instanceof TreeContext) {
				// children of a contexts are crds
				this.load.setContext(element.context);
				let crds = await this.load.getCrds();
				elements.push(...crds.map(crd => new TreeCrd(crd)));
			}
			if (element instanceof TreeCrd) {
				// children of an crds are crs
				let crs = await this.load.getCrs(element.crd);
				elements.push(...crs.map(cr => new TreeCr(cr)));
			}
		} else {
			// top level children are the contexts
			let contexts = this.load.getContexts();
			elements.push(...contexts.map(context => new TreeContext(context)));
		}
		return elements;
	}
}
