<!--
Guiding Principles:

Changelogs are for humans, not machines.
There should be an entry for every single version.
The same types of changes should be grouped.
Versions and sections should be linkable.
The latest version comes first.
The release date of each version is displayed.
Mention whether you follow Semantic Versioning.

Usage:

Change log entries are to be added to the Unreleased section under the
appropriate stanza (see below). Each entry should ideally include a tag and
the Github issue reference in the following format:

* (<tag>) \#<issue-number> message

The issue numbers will later be link-ified during the release process so you do
not have to worry about including a link manually, but you can if you wish.

Types of changes (Stanzas):

"Features" for new features.
"Improvements" for changes in existing functionality.
"Deprecated" for soon-to-be removed features.
"Bug Fixes" for any bug fixes.
"Client Breaking" for breaking CLI commands and REST routes used by end-users.
"API Breaking" for breaking exported APIs used by developers building on SDK.
"State Machine Breaking" for any changes that result in a different AppState given the same genesisState and txList.
Ref: https://keepachangelog.com/en/1.0.0/
-->

# Changelog

## [Unreleased]

### Dependencies

* [\#6828](https://github.com/cosmos/ibc-go/pull/6828) Bump Cosmos SDK to v0.50.9.
* [\#6193](https://github.com/cosmos/ibc-go/pull/6193) Bump `cosmossdk.io/store` to v1.1.0.
* [\#7247](https://github.com/cosmos/ibc-go/pull/7247) Bump CometBFT to v0.38.12.

### API Breaking

* (apps/callbacks)  [\#7000](https://github.com/cosmos/ibc-go/pull/7000) Add base application version to contract keeper callbacks.

### State Machine Breaking

### Improvements

### Features

### Bug Fixes

<!-- markdown-link-check-disable-next-line -->
## [v0.2.0+ibc-go-v8.0](https://github.com/cosmos/ibc-go/releases/tag/modules%2Fapps%2Fcallbacks%2Fv0.2.0%2Bibc-go-v8.0) - 2023-11-15

### Bug Fixes

* [\#4568](https://github.com/cosmos/ibc-go/pull/4568) Include error in event that is emitted when the callback cannot be executed due to a panic or an out of gas error. Packet is only sent if the `IBCSendPacketCallback` returns nil explicitly.

<!-- markdown-link-check-disable-next-line -->
## [v0.1.0+ibc-go-v7.3](https://github.com/cosmos/ibc-go/releases/tag/modules%2Fapps%2Fcallbacks%2Fv0.1.0%2Bibc-go-v7.3) - 2023-08-31

### Features

* [\#3939](https://github.com/cosmos/ibc-go/pull/3939) feat(callbacks): ADR8 implementation.
