# Vendor Neutrality Guidelines

## Overview

The OCM AI Assistance subproject maintains **strict vendor neutrality** to align with Open Cluster Management's core principles. All content must be portable and work across multiple AI platforms and tools.

## Core Principles

### 1. Platform Agnostic

Content must work with any AI assistant tool or platform, including but not limited to:

- Claude Code (Anthropic)
- GitHub Copilot (OpenAI/GitHub)
- Cursor (Anysphere)
- Google Gemini Code Assist
- JetBrains AI Assistant
- Amazon CodeWhisperer
- Other general-purpose AI coding assistants

### 2. Format Portability

- Use **standard markdown** for all documentation
- Avoid proprietary or platform-specific formats
- No embedded tool-specific commands or syntax
- No hardcoded references to specific AI models or versions

### 3. Tool Independence

- Content should provide guidance, not tool-specific workflows
- Focus on **what to do**, not which tool to use
- Examples should be conceptual, not tool-specific

## Compliance Guidelines

### ✅ ALLOWED

#### Standard Markdown
```markdown
# OCM Add-on Development Guide

Follow these steps to create a new OCM add-on:

1. Define the add-on manifest
2. Implement the controller logic
3. Set up registration and deployment
```

#### Vendor-Neutral Prompts
```markdown
Create an OCM add-on that monitors cluster health and reports
metrics to a central hub. Include:
- AddOnTemplate resource definition
- Controller implementation
- Health check logic
- Metric reporting
```

#### Platform-Agnostic Instructions
```markdown
Use this prompt with your AI assistant to generate placement policy:

"Generate an OCM Placement resource that selects clusters with 
label 'environment=production' and at least 4 CPU cores available."
```

#### General AI Assistant References
```markdown
When working with your AI coding assistant, provide context about:
- Current cluster architecture
- Desired placement criteria
- Resource constraints
```

### ❌ NOT ALLOWED

#### Platform-Specific Commands
```markdown
# DON'T DO THIS
Use `/ask` in Claude Code to query the codebase...
Press Cmd+K in Cursor to open the chat...
Type @workspace in Copilot to reference files...
```

#### Vendor-Specific Syntax
```markdown
# DON'T DO THIS
<claude_skill>
  <name>ocm-addon</name>
  <prompt>...</prompt>
</claude_skill>
```

#### Hardcoded Tool References
```markdown
# DON'T DO THIS
This guide requires Claude Code 1.5 or later
Compatible with GitHub Copilot Chat only
Designed for Cursor's AI model
```

#### Provider-Specific APIs
```markdown
# DON'T DO THIS
import anthropic
client = anthropic.Client(api_key="...")

# Or
from openai import OpenAI
client = OpenAI(api_key="...")
```

#### Commercial Feature Dependencies
```markdown
# DON'T DO THIS
Requires Claude Code Pro subscription
GitHub Copilot Enterprise license needed
Premium tier access required
```

## Testing Requirements

All contributions must be tested for vendor neutrality:

### Minimum Testing

Test your content with **at least 2 different AI platforms** before submitting:

- [ ] Works with Platform A (e.g., Claude Code)
- [ ] Works with Platform B (e.g., GitHub Copilot)
- [ ] No platform-specific syntax or commands
- [ ] Instructions are clear without tool-specific context

### Testing Checklist

When testing your contribution, verify:

1. **Portability**: Can the content be copy-pasted into any AI assistant?
2. **Clarity**: Are instructions understandable without knowing which tool to use?
3. **Completeness**: Does the content provide enough context on its own?
4. **Functionality**: Does it achieve the intended outcome across platforms?

## Examples

### Good: Vendor-Neutral Prompt

```markdown
# Scaffold OCM Add-on

## Prompt

Create a new OCM add-on with the following specifications:

**Add-on Name**: cluster-monitor
**Purpose**: Monitor cluster resource usage and report to hub
**Components**:
- AddOnTemplate custom resource
- Controller with reconciliation logic
- Health status reporting
- Metric collection every 5 minutes

**Requirements**:
- Use controller-runtime framework
- Follow OCM add-on best practices
- Include proper error handling
- Add unit tests for controller logic

Please generate:
1. AddOnTemplate YAML manifest
2. Controller implementation in Go
3. Health check logic
4. Basic test structure
```

**Why it's good**: Works with any AI assistant, no tool-specific commands, clear requirements.

### Bad: Vendor-Specific Prompt

```markdown
# DON'T DO THIS

Use Claude Code's /generate command to scaffold an add-on.

In the chat, type:
@workspace /generate addon cluster-monitor

Then use Cmd+Shift+P and select "Claude: Refine" to improve the code.
```

**Why it's bad**: References specific tool commands, keyboard shortcuts, and features unique to one platform.

## Borderline Cases

### Documentation Links

**Allowed**: Links to official OCM documentation
```markdown
See the [OCM Add-on Guide](https://open-cluster-management.io/concepts/addon/)
```

**Not Allowed**: Links to tool-specific documentation
```markdown
See the [Claude Code Prompting Guide](https://docs.anthropic.com/claude-code)
```

### Tool Comparisons

**Allowed**: Acknowledging multiple tools exist
```markdown
This prompt works with various AI coding assistants including 
Claude Code, GitHub Copilot, Cursor, and others.
```

**Not Allowed**: Recommending specific tools
```markdown
We recommend using Claude Code for best results.
Cursor works better than other options.
```

### Code Examples

**Allowed**: Standard code with comments
```go
// Controller reconciliation loop for OCM add-on
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Fetch the add-on instance
    // Update status
    // Handle errors
}
```

**Not Allowed**: Code with AI-tool-specific markers
```go
// @cursor: implement reconciliation logic here
// TODO(copilot): add error handling
```

## Review Process

Maintainers will review contributions for vendor neutrality using this checklist:

- [ ] Uses standard markdown format
- [ ] No platform-specific commands or syntax
- [ ] Works across multiple AI platforms (verified by contributor)
- [ ] No proprietary formats or bindings
- [ ] No commercial feature dependencies
- [ ] Examples are tool-agnostic
- [ ] Documentation is clear without tool context
- [ ] Tested on at least 2 platforms

## Questions?

If you're unsure whether your contribution meets vendor neutrality requirements:

1. Review the examples above
2. Test with multiple platforms
3. Ask in your PR description
4. Tag maintainers for guidance

## Updates

These guidelines may be updated as the project evolves. Last updated: 2026-09-21
