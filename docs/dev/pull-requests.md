# Pull request guidelines

> To accommodate the review process we suggest that PRs are categorically broken up. Ideally each PR addresses only a single issue and does not introduce unrelated changes. Additionally, as much as possible code refactoring and cleanup should be submitted as separate PRs from bug fixes and feature additions.

If the PR is the result of a related GitHub issue, please include `closes: #<issue number>` in the PR’s description in order to auto-close the related issue once the PR is merged. This will also link the ticket and the PR together so that if anyone looks at either in the future, they won’t have any issue trying to find the corresponding ticket/PR as it will be recorded in the sidebar.

If the PR is not the result of an existing issue and it fixes a bug, please provide a detailed description of the bug. For feature addtions, we recommend opening an issue first and have it discussed and agreed upon, before working on it and opening a PR.

Commit messages must follow the [Conventional Commits specification](https://www.conventionalcommits.org/en/v1.0.0/). This will help us to eventually move to automatic generation of changelogs.

If possible, [tick the "Allow edits from maintainers" box](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/allowing-changes-to-a-pull-request-branch-created-from-a-fork) when opening your PR from your fork of ibc-go. This allows us to directly make minor edits / refactors and speeds up the merging process.

If you open a PR on ibc-go, it is mandatory to update the relevant documentation in `/docs`.

## Pull request targeting

Ensure that you base and target your PR on the `main` branch. Please make sure that the PR is made from a branch different than `main`.

All development should be targeted against `main`. Bug fixes which are required for outstanding releases should be backported if the CODEOWNERS decide it is applicable. 

## Pull request review process

All PRs require an approval from at least one CODEOWNER before merge. PRs which cause significant changes require two approvals from CODEOWNERS. When reviewing PRs please use the following review explanations:

- `LGTM` without an explicit approval means that the changes look good, but you haven't pulled down the code, run tests locally and thoroughly reviewed it.
- `Approval` through the GitHub UI means that you understand the code, documentation/spec is updated in the right places, you have pulled down and tested the code locally. In addition:
  - You must also think through anything which ought to be included but is not.
  - You must think through whether any added code could be partially combined (DRYed) with existing code.
  - You must think through any potential security issues or incentive-compatibility flaws introduced by the changes.
  - Naming must be consistent with conventions and the rest of the codebase
  - Code must live in a reasonable location, considering dependency structures (e.g. not importing testing modules in production code, or including example code modules in production code).
  - If you approve of the PR, you are responsible for fixing any of the issues mentioned here and more.
- If you sat down with the PR submitter and did a pairing review please note that in the `Approval`, or your PR comments.
- If you are only making "surface level" reviews, submit any notes as `Comments` without adding a review.

## Pull request merge procedure

- Ensure all GitHub requirements pass.
- Squash and merge pull request.
