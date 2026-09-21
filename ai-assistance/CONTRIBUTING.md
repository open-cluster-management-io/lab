# Contributing to AI Assistance

Thank you for your interest in contributing to the OCM AI Assistance subproject! This document provides guidelines for contributing vendor-neutral AI assistance content.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Contribution Guidelines](#contribution-guidelines)
- [Content Types](#content-types)
- [Vendor Neutrality Requirements](#vendor-neutrality-requirements)
- [Submission Process](#submission-process)
- [Review Criteria](#review-criteria)

## Code of Conduct

This project adheres to the [OCM Code of Conduct](../CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

1. **Fork the repository**: Create your own fork of the [lab repository](https://github.com/open-cluster-management-io/lab)
2. **Create a branch**: Use a descriptive branch name (e.g., `ai-assistance/add-addon-prompt`)
3. **Make your changes**: Add or modify content following the guidelines below
4. **Test your content**: Verify that your contribution works across multiple AI platforms
5. **Submit a PR**: Open a pull request with a clear description

## Contribution Guidelines

### General Principles

- **Vendor Neutral**: Content must work across multiple AI platforms (Claude, Cursor, Copilot, Gemini, etc.)
- **Portable**: Use markdown and avoid platform-specific formats
- **Clear and Concise**: Write clear instructions that are easy to understand and follow
- **Tested**: Verify your content works as intended before submitting
- **Well-Documented**: Include context and examples where appropriate

### File Organization

```
ai-assistance/
├── prompts/           # Reusable AI prompts
│   ├── addon-development/
│   ├── policy-authoring/
│   └── placement-logic/
└── guides/            # Contributor guides
    ├── getting-started/
    ├── best-practices/
    └── troubleshooting/
```

## Content Types

### 1. Prompts (`prompts/`)

Reusable prompts for specific OCM development tasks.

**Format:**
```markdown
# [Task Name]

## Purpose
Brief description of what this prompt helps accomplish.

## Context
Background information the AI needs to understand the task.

## Prompt
```
[The actual prompt text]
```

## Expected Output
Description of what the AI should produce.

## Example Usage
Concrete example showing the prompt in action.
```

**Naming Convention**: Use descriptive kebab-case names (e.g., `scaffold-new-addon.md`, `write-placement-policy.md`)

### 2. Guides (`guides/`)

Detailed contributor guides for building OCM components.

**Format:**
```markdown
# [Guide Title]

## Overview
High-level summary of what this guide covers.

## Prerequisites
- List of required knowledge
- Tools needed
- Environment setup

## Steps
1. Step-by-step instructions
2. With code examples
3. And explanations

## Best Practices
Key recommendations and patterns.

## Troubleshooting
Common issues and solutions.

## Next Steps
Where to go from here.
```

## Vendor Neutrality Requirements

All contributions **MUST** adhere to vendor neutrality guidelines:

### ✅ DO

- Use standard markdown format
- Write prompts that work across different AI platforms
- Provide clear, platform-agnostic instructions
- Include examples that don't depend on specific tools
- Focus on OCM concepts and patterns
- Reference official OCM documentation

### ❌ DON'T

- Include platform-specific commands or formats (e.g., Claude-only syntax, Cursor-specific commands)
- Require specific AI tools or versions
- Embed proprietary bindings or integrations
- Reference commercial or vendor-specific features
- Include provider-specific API calls

See [VENDOR_NEUTRALITY.md](VENDOR_NEUTRALITY.md) for detailed guidelines.

## Submission Process

1. **Create your content** following the guidelines above
2. **Test thoroughly**:
   - Try your prompt/guide with at least 2 different AI platforms
   - Verify instructions are clear and complete
   - Check that examples work correctly
3. **Write a clear PR description**:
   - Explain what your contribution adds
   - Describe how you tested it
   - Note any dependencies or prerequisites
4. **Address review feedback** promptly and constructively

## Review Criteria

Contributions will be evaluated based on:

- **Vendor Neutrality**: Works across multiple AI platforms
- **Accuracy**: Technically correct and aligned with OCM best practices
- **Clarity**: Easy to understand and follow
- **Completeness**: Includes necessary context and examples
- **Testing**: Demonstrably tested across platforms
- **Documentation**: Well-documented with clear purpose and usage

## Questions or Issues?

- Open an issue in the [OCM Community repository](https://github.com/open-cluster-management-io/community/issues)
- Reach out on OCM community channels
- Tag maintainers in your PR for guidance

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

---

**Thank you for contributing to OCM AI Assistance!**
