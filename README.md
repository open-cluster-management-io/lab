![image](assets/ocm-lab-logo.jpg)

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Welcome to the lab repo for [Open Cluster Management (OCM)](https://open-cluster-management.io/).
This repo hosts experimental projects that anyone in the community can try out, provide feedback on, and contribute to.
Feel free to link to these projects from your own websites or repos, gauge interest, and help us improve as we iterate.

Unlike a dedicated labs GitHub org (ie: [argoproj-labs](https://github.com/argoproj-labs)),
this repo hosts all lab projects in one place.
New projects are onboarded via PR and added as subfolders, each governed by its own `OWNERS` file.
Since our community is still small compared to the [arogoproj](https://github.com/argoproj),
keeping everything together avoids unnecessary fragmentation.


For new add-on projects, please use the
[addon-contrib](https://github.com/open-cluster-management-io/addon-contrib) repo.


## Table of Contents
- [Current Projects](#current-projects)
- [Onboarding a New Project](#onboarding-a-new-project)
- [Governance](#governance)
- [Issues for Lab Projects](#issues-for-lab-projects)  
- [PRs for Lab Projects](#prs-for-lab-projects)


## Current Projects

- **dashboard**

  OCM UI Dashboard

- **vscode-extension** (Soon)

  OCM VScode Extension (Soon)

## Onboarding a New Project

To onboard a new lab project:

1. If not already discussed with maintainers, open an issue to propose your idea.
1. Once acknowledged, create a folder named after your project and add your code/docs.
1. Add an `OWNERS` file listing the new project's maintainers.
1. Update the [Current Projects](#current-projects) section in this README with the name and a short description.  
1. Create a PR with a brief project overview and confirm the `OWNERS` file is present.  
1. An OCM maintainer will review and merge the PR.  


## Governance

Once onboarded, each project is fully self governed by its `OWNERS` file,
which defines who can review and merge changes.
Any updates to the project maintainers are made via PR to the `OWNERS` file
and require approval from existing approvers.


## Issues for Lab Projects

When reporting a bug or requesting a feature for an existing lab project,
open an issue with the project folder name at the start of the title (ie: `dashboard - foobar bug`).
The maintainers listed in the project’s `OWNERS` file will triage your issue.


## PRs for Lab Projects

To contribute to an existing lab project,
open a PR with the project folder name at the start of the title (ie: `dashboard - fix foobar bug`).
The maintainers in the `OWNERS` file will review the PR.
