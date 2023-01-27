# Tagging a release

## New major release branch

Pre-requisites for creating a release branch for a new major version:

1. Bump [Go package version](https://github.com/cosmos/ibc-go/blob/main/go.mod#L3).
2. Change all imports. For example: if the next major version is `v3`, then change all imports starting with `github.com/cosmos/ibc-go/v2` to `github.com/cosmos/ibc-go/v3`).

Once the above pre-requisites are satified:

1. Start on `main`.
2. Create the release branch (`release/vX.XX.X`). For example: `release/v3.0.x`.

## New minor release branch

1. Start on the latest release branch in the same major release line. For example: the latest release branch in the `v3` release line is `v3.2.x`.
2. Create branch from the release branch. For example: create branch `release/v3.3.x` from `v3.2.x`.

Post-requisites for both new major and minor release branches:

1. Add branch protection rules to new release branch.
2. Add backport task to [`mergify.yml`](https://github.com/cosmos/ibc-go/blob/main/.github/mergify.yml).
3. Create label for backport (e.g.`backport-to-v3.0.x`).

## Point release procedure

In order to alleviate the burden for a single person to have to cherry-pick and handle merge conflicts of all desired backporting PRs to a point release, we instead maintain a living backport branch, where all desired features and bug fixes are merged into as separate PRs.

### Example

Current release is `v1.0.2`. We then maintain a (living) branch `release/v1.0.x`, given `x` as the next patch release number (currently `v1.0.3`) for the `v1.0` release series. As bugs are fixed and PRs are merged into `main`, if a contributor wishes the PR to be released into the `v1.0.x` point release, the contributor must:

1. Add the `backport-to-v1.0x` label to the PR.
2. Once the PR is merged, the Mergify GitHub application will automatically copy the changes into another branch and open a new PR agains the desired `release/v1.0.x` branch.
3. If the following has not been discussed in the original PR, then update the backport PR's description and ensure it contains the following information:
  - **[Impact]** explanation of how the bug affects users or developers.
  - **[Test Case]** section with detailed instructions on how to reproduce the bug.
  - **[Regression Potential]** section with a discussion how regressions are most likely to manifest, or might manifest even if it's unlikely, as a result of the change. **It is assumed that any backport PR is well-tested before it is merged in and has an overall low risk of regression**. This section should discuss the potential for state breaking changes to occur such as through out-of-gas errors. 

It is the PR's author's responsibility to fix merge conflicts, update changelog entries, and ensure CI passes. If a PR originates from an external contributor, it may be a core team member's responsibility to perform this process instead of the original author. Lastly, it is core team's responsibility to ensure that the PR meets all the backport criteria.

Finally, when a point release is ready to be made:

1. Checkout the release branch (e.g. `release/v1.0.x`).
2. In `CHANGELOG.md`:
  - Ensure changelog entries are verified.
  - Remove any sections of the changelog that do not have any entries (e.g. if the release does not have any bug fixes, then remove the section).
  - Remove the `[Unreleased]` title.
  - Add release version and date of release.
3. Create release in GitHub:
  - Select the correct target branch (e.g. `release/v1.0.x`).
  - Choose a tag (e.g. `v1.0.3`).
  - Write release notes.
  - Check the `This is a pre-release` checkbox if needed (this applies for alpha, beta and release candidates).

### Post-release procedure

- Update [`CHANGELOG.md`](../../CHANGELOG.md) in `main` (remove from the `[Unreleased]` section any items that are part of the release).`
- Put back the `[Unreleased]` section in the release branch (e.g. `release/v1.0.x`) with clean sections for each of the types of changelog entries, so that entries will be added for the PRs that are backported for the next release.
- Update [version matrix](../../RELEASES.md#version-matrix) in `RELEASES.md`: add the new release and remove any tags that might not be recommended anymore.

Additionally, for the first point release of a new major or minor release branch:

- Update the table of supported release lines (and End of Life dates) in [`RELEASES.md`](../../RELEASES.md): add the new release line and remove any release lines that might have become discontinued.
- Update the [list of supported release lines in README.md](../../RELEASES.md#releases), if necessary.
- Update the [e2e compatibility test matrices](https://github.com/cosmos/ibc-go/tree/main/.github/compatibility-test-matrices): add the tag for the new release and remove any tags that might not be recommended anymore.
- Update the manual [e2e `simd`](https://github.com/cosmos/ibc-go/blob/main/.github/workflows/e2e-manual-simd.yaml) and [e2e `icad`](https://github.com/cosmos/ibc-go/blob/main/.github/workflows/e2e-manual-icad.yaml) test workflows:
  - Add the new release and the new `icad` tag.
  - Remove any tags that might not be recommended anymore.
- Bump ibc-go version in [cosmos/interchain-accounts-demo repository](https://github.com/cosmos/interchain-accounts-demo) and create a tag.
- Open a PR to `main` updating the docs site:
  - Add new release branch to [`docs/versions`](../versions) file.
  - Add `label` and `key` to `versions` array in [`config.js`](https://github.com/cosmos/ibc-go/blob/main/docs/.vuepress/config.js#L33).
- After changes to docs site are deployed, check [ibc.cosmos.network](https://ibc.cosmos.network) is updated.
- Open issue in [SDK tutorials repo](https://github.com/cosmos/sdk-tutorials) to update tutorials to the released version of ibc-go.

See [this PR](https://github.com/cosmos/ibc-go/pull/2919) for an example of the involved changes.