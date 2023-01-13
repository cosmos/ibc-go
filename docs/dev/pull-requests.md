# Pull request guidelines

> To accommodate the review process we suggest that PRs are categorically broken up. Ideally each PR addresses only a single issue and does not introduce unrelated changes. Additionally, as much as possible code refactoring and cleanup should be submitted as separate PRs from bug fixes and feature additions.

If the PR is the result of a related GitHub issue, please include `closes: #<issue number>` in the PR’s description in order to auto-close the related issue once the PR is merged. This will also link the issue and the PR together so that if anyone looks at either in the future, they won’t have any problem trying to find the corresponding issue/PR as it will be recorded in the sidebar.

If the PR is not the result of an existing issue and it fixes a bug, please provide a detailed description of the bug. For feature addtions, we recommend opening an issue first and have it discussed and agreed upon, before working on it and opening a PR.

If possible, [tick the "Allow edits from maintainers" box](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/allowing-changes-to-a-pull-request-branch-created-from-a-fork) when opening your PR from your fork of ibc-go. This allows us to directly make minor edits / refactors and speeds up the merging process.

If you open a PR on ibc-go, it is mandatory to update the relevant documentation in `/docs`.

## Pull request targeting

Ensure that you base and target your PR on the either the `main` branch or the corresponding feature branch where a large body of work is being implemented. Please make sure that the PR is made from a branch different than either `main` or the corresponding feature branch.

All development should be then targeted against `main` or the feature branch. Bug fixes which are required for outstanding releases should be backported if the CODEOWNERS decide it is applicable. 

## Commit Messages

Commit messages should follow the [Conventional Commits specification](https://www.conventionalcommits.org/en/v1.0.0/).

When opening a PR, include the proposed commit message in the PR description.

The commit message type should be one of:

- `feat` / `feature` for feature work.
- `bug` / `fix` for bug fixes.
- `imp` / `improvements` for improvements.
- `doc` / `docs` / `documentation` for any documentation changes.
- `test` / `e2e` for addition or improvements of unit, integration and e2e tests or their corresponding infrastructure.
- `deprecated` for deprecation changes.
- `deps` / `build` for changes to dependencies.
- `chore` / `misc` / `nit` for any miscellaneous changes that don't fit into another category.

**Note**: If any change is breaking, the following format must be used:

- `type` + `(api)!` for api breaking changes, e.g. `fix(api)!: api breaking fix`
- `type` + `(statemachine)!` for state machine breaking changes, e.g. `fix(statemachine)!: state machine breaking fix`

**`api` breaking changes take precedence over `statemachine` breaking changes.**

## Pull request review process

All PRs require an approval from at least one CODEOWNER before merge. PRs which cause significant changes require two approvals from CODEOWNERS. When reviewing PRs please use the following review guidelines:

- `Approval` through the GitHub UI with the following comments:
  - `Concept ACK` means that you agree with the overall proposed concept, but have neither reviewed the code nor tested it.
  - `LGTM` means the above and besides you have superficially reviewed the code without considering how logic affects other parts the codebase.
  - `utACK` (aka. `Untested ACK`) means the above and besides have thoroughly reviewed the code and considered the safety of logic changes, but have not tested it.
  - `Tested ACK` means the above and besides you have tested the code.
- If you are only making "surface level" reviews, submit any notes as `Comments` without submitting an approval.

A thorough review means that:
- You understand the code and make sure that documentation is updated in the right places.
- You must also think through anything which ought to be included but is not.
- You must think through whether any added code could be partially combined (DRYed) with existing code.
- You must think through any potential security issues or incentive-compatibility flaws introduced by the changes.
- Naming must be consistent with conventions and the rest of the codebase.
- Code must live in a reasonable location, considering dependency structures (e.g. not importing testing modules in production code, or including example code modules in production code).

## Pull request merge procedure

- Ensure pull request branch is rebased on target branch.
- Ensure all GitHub requirements pass.
- Set the changelog entry in the commit message for the pull request.
- Squash and merge pull request.
