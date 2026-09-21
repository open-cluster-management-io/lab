# AI Assistance for Open Cluster Management

**Status:** Experimental Subproject

## Overview

The `ai-assistance` subproject provides **vendor-neutral AI assistance** content for people building and operating Open Cluster Management (OCM) add-ons and multicluster workflows. This includes prompts and contributor guides that work across different AI tools and platforms.

## Goals

- Provide **vendor-neutral** assistance content compatible with multiple AI platforms (Claude, Cursor, Copilot, Gemini, etc.)
- Help contributors author OCM add-ons, policies, placement patterns, and multicluster workflows
- Maintain portability through markdown and tool-agnostic guidance
- Serve as an upstream home for community AI assistance, aligned with OCM's vendor-neutral principles

## What's Included

- **Prompts** (`prompts/`) - Reusable AI prompts for common OCM development tasks
- **Guides** (`guides/`) - Contributor guides for building OCM components

## Vendor Neutrality

This subproject adheres to OCM's vendor-neutral principles:

- ✅ Portable markdown and tool-agnostic guidance
- ✅ Works across multiple AI platforms and tools
- ✅ No single-vendor skill format required
- ✅ No provider-specific bindings in core content
- ❌ Provider-specific implementations belong outside this subproject

See [VENDOR_NEUTRALITY.md](VENDOR_NEUTRALITY.md) for detailed guidelines.

## Getting Started

### Using the Assistance Content

1. Browse the `prompts/` directory for ready-to-use prompts
2. Check the `guides/` directory for detailed contributor guides
3. Use the content with your preferred AI tool (Claude Code, Cursor, GitHub Copilot, etc.)

### Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:

- Adding new prompts and guides
- Maintaining vendor neutrality
- Testing and validation
- Submission process

## Examples

- **Add-on Development**: Prompts for scaffolding new OCM add-ons
- **Policy Patterns**: Guidance for writing OCM policies
- **Placement Logic**: Assistance with cluster placement and selection
- **Multicluster Workflows**: Patterns for cross-cluster operations

## Documentation

For more information about OCM, visit:

- [OCM Documentation](https://open-cluster-management.io/)
- [OCM Community](https://github.com/open-cluster-management-io/community)
- [OCM Governance](https://github.com/open-cluster-management-io/community/blob/main/GOVERNANCE.md)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../LICENSE) file for details.

## Maintainers

This is an experimental subproject under the OCM community. For questions or issues, please refer to the [OCM Community Guidelines](https://github.com/open-cluster-management-io/community).

## Related Links

- [Proposal Issue #240](https://github.com/open-cluster-management-io/community/issues/240)
- [OCM Lab Repository](https://github.com/open-cluster-management-io/lab)
