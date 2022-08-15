---
name: Release tracker
about: Create an issue to track release progress

---

<!-- < < < < < < < < < < < < < < < < < < < < < < < < < < < < < < < < < ☺ 
v                            ✰  Thanks for opening an issue! ✰    
v    Before smashing the submit button please review the template.
v    Word of caution: poorly thought-out proposals may be rejected 
v                     without deliberation 
☺ > > > > > > > > > > > > > > > > > > > > > > > > > > > > > > > > >  -->

## Milestones

<!-- Links to alpha, beta, RC or final milestones -->

## IBC spec compatibility

<!-- Version of the IBC spec that this release is compatible with -->

## QA

### Backwards compatibility

<!-- List of tests that need be performed with previous
versions of ibc-go to guarantee that no regression is introduced -->

- [ ] Fungible token transfers over a non-incentivised channel works with a counterparty chain running each previous major version.
- [ ] Fungible token transfers over an incentivised channel works with a counterparty chain running each previous major version that supports ICS-29 Fee Middleware.
- [ ] Interchain Accounts over a non-incentivised channel works with a counterparty chain running each previous major version that supports ICS-27 Interchain Accounts over non-incentivised channels.
- [ ] Interchain Accounts over an incentivised channel works with a counterparty chain running each previous major version that supports ICS-27 Interchain Accounts over incentivised channels.

## Migration 

<!-- Link to migration document -->

## Checklist

<!-- Remove any items that are not applicable. -->

- [ ] Bump [go package version](https://github.com/cosmos/ibc-go/blob/main/go.mod#L3).
- [ ] Change all imports starting with `github.com/cosmos/ibc-go/v{x}` to `github.com/cosmos/ibc-go/v{x+1}`.
- [ ] Branch off main to create release branch in the form  of `release/vx.y.z`.
- [ ] Add branch protection rules to new release branch.
- [ ] Add backport task to [`mergify.yml`](https://github.com/cosmos/ibc-go/blob/main/.github/mergify.yml)
- [ ] Upgrade ibc-go version in [ibctest](https://github.com/strangelove-ventures/ibctest).
- [ ] Check Swagger is up-to-date.

## Post-release checklist

- [ ] Update [`CHANGELOG.md`](https://github.com/cosmos/ibc-go/blob/main/CHANGELOG.md)
- [ ] Update the table of supported release lines (and End of Life dates) in [`RELEASES.md`](https://github.com/cosmos/ibc-go/blob/main/RELEASES.md).
- [ ] Update [version matrix](https://github.com/cosmos/ibc-go/blob/main/RELEASES.md#version-matrix) in `RELEASES.md`.
- [ ] Update docs site:
  - [ ] Add new release branch to [`docs/versions`](https://github.com/cosmos/ibc-go/blob/main/docs/versions) file.
  - [ ] Add `label` and `key` to `versions` array in [`config.js`](https://github.com/cosmos/ibc-go/blob/main/docs/.vuepress/config.js#L33).
- [ ] After changes to docs site are deployed, check [ibc.cosmos.network](https://ibc.cosmos.network) is updated.

____

#### For Admin Use

- [ ] Not duplicate issue
- [ ] Appropriate labels applied
- [ ] Appropriate contributors tagged/assigned
