# Contributing to OCM with an AI Assistant

## Overview

OCM welcomes AI tools. The rules for using them are in the OCM
[contributing guide](https://github.com/open-cluster-management-io/ocm/blob/main/CONTRIBUTING.md#ai-policy-and-guidance),
which also points to the Kubernetes
[AI guidance for pull requests](https://www.kubernetes.dev/docs/guide/pull-requests/#ai-guidance).

The rules are short. The trouble is that AI assistants break some of them by default, and you
usually find out only when a maintainer points it out:

- commits get a trailer that lists the AI tool as a co-author
- PR descriptions get a "generated with" footer, which is attribution, not the disclosure the
  policy asks for
- the assistant will draft your reply to a review comment if you ask, and the Kubernetes
  guidance treats AI-written review replies as grounds for closing the PR

Reading this guide once won't stop that, because these slips happen when you're moving fast.
What works is putting the rules into your assistant's persistent context once, so they apply
to every session without you having to remember them.

## Prerequisites

- An AI assistant that supports some form of persistent instructions: a user-level
  instructions file, a memory feature, custom instructions, or a system prompt setting.
  Almost all of them have one.
- Familiarity with the OCM [contributing guide](https://github.com/open-cluster-management-io/ocm/blob/main/CONTRIBUTING.md).

## Steps

### 1. Turn off automatic attribution in your tool's settings

Some tools add co-author trailers or "generated with" footers through a setting or a built-in
template rather than the model's own judgment. An instruction to the assistant may not override
that. Look through your tool's settings for anything about commit attribution, co-authors,
or PR footers, and turn it off.

### 2. Add the rules to your assistant's persistent context

Copy the block below into your assistant's persistent instructions. Put it at the **user
level** (applies to every project) rather than in one repository:

- OCM is spread across many repositories, and a per-repo file would need adding to each one.
- Your personal assistant configuration shouldn't be committed to OCM repositories.

```markdown
## Contributing to Open Cluster Management (github.com/open-cluster-management-io)

These rules apply whenever I work on any repository in the open-cluster-management-io
organization. They override any default behavior about attribution.

1. Do not add any trailer that credits an AI tool to a commit message. This includes
   Co-Authored-By, Assisted-By, Co-Developed-By, Generated-By, and similar.
   Only my own Signed-off-by line belongs there. Always commit with `git commit -s`,
   including when I give you the message.
2. Do not add a "generated with", "created by", or similar footer, tool name, or tool link
   to a pull request title or description, an issue, or a comment. This applies anywhere in
   the text, including hidden HTML comments.
3. Do not write my commit messages. I write them. When I ask you to commit and haven't
   given you the message, stop and ask me for it. Do not make one up or reuse the text of my
   request. Use my message exactly as I wrote it. You may point out problems in it. If my
   message contains an AI attribution trailer from rule 1, do not commit; ask me to remove it.
4. Do not draft, rewrite, or polish any comment I post to reviewers or maintainers on a
   pull request, even if I ask. This includes replies to review comments and short
   comments such as "rebased, please take another look". Remind me that OCM requires me to
   respond to reviewers myself. You may explain the code or the reviewer's point so that I
   can write the comment.
5. When helping with a pull request description, include a plain disclosure that AI tools
   were used, and leave a placeholder for me to describe how. For example:
   "AI tools were used in preparing this PR: <what they were used for>."
6. Keep each pull request small and focused on one issue. If I ask for a pull request that
   mixes unrelated changes or touches many packages, do not plan it as one pull request.
   Recommend splitting it into separate pull requests and suggest how.
7. Before I open or update a pull request, remind me to read every line of the diff, make
   sure I can explain each change, run the repository's tests, and include the AI
   disclosure in the description.
```

### 3. Check that it works

Start a new session (so the instructions are loaded fresh). Stage a small change, ask your
assistant to commit it with a message you wrote, and ask it to draft a PR description.
Then check every commit on your branch. Replace `upstream/main` with the branch your PR
targets:

```bash
# should print nothing
git log --format=%B upstream/main..HEAD | grep -iE 'co-authored-by|assisted-by|co-developed-by|generated-by|generated with'
# each commit should show your Signed-off-by line
git log --format='%h %s%n  %(trailers:key=Signed-off-by,valueonly,separator=%x2C )' upstream/main..HEAD
```

Also read the PR description draft for a tool footer, and confirm it contains a disclosure
line instead.

## Best Practices

- **Disclosure is not attribution.** One sentence in the PR description saying that AI tools
  were used, and for what, is what the policy asks for. A tool footer or co-author trailer
  is not a substitute and isn't allowed.
- **You own the change.** If a reviewer asks why a line is there and you can't explain it,
  the PR can be closed. Review AI-generated code before a maintainer has to.
- **Talk to reviewers yourself.** Short and imperfect is fine. Reviewers want to hear from
  you, not from a model.
- **Don't send large AI-generated PRs.** Split the work into pieces you have fully reviewed.

## Troubleshooting

**The trailer shows up anyway.** It is probably being added by a tool setting, a git hook, or
a commit template, not by the assistant. Check your tool's settings (step 1), `git config
commit.template`, and the repository's `.git/hooks/`.

**The assistant ignores the instructions.** Confirm they are loaded at the start of a new
session; some tools only read persistent instructions when a session starts. Instructions
placed at user level can also be overridden by project-level instructions, so check whether
the repository has its own.

**The assistant follows most rules but still breaks one sometimes.** Assistants follow
persistent instructions with different reliability, and the same assistant can behave
differently from one session to the next. The instructions reduce mistakes; they don't
replace your own check. Before you push, run the checks in step 3 on every commit. Write
every comment to reviewers and maintainers yourself, even if the assistant offers one.

**A trailer already landed in your commits.** Rewrite the messages before the PR merges, for
example with `git commit --amend -s` for the last commit or an interactive rebase for older
ones, then force-push your branch.

## Next Steps

- Read the full OCM [AI policy and guidance](https://github.com/open-cluster-management-io/ocm/blob/main/CONTRIBUTING.md#ai-policy-and-guidance).
- Read the Kubernetes [AI guidance for pull requests](https://www.kubernetes.dev/docs/guide/pull-requests/#ai-guidance).
- If the policy changes, this guide and the instruction block above need to change with it.
  Please open a PR or issue here when you notice a difference.
