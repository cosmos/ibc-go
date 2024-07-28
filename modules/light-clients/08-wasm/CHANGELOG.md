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

<!-- markdown-link-check-disable-next-line -->
## [v0.3.0+ibc-go-v7.3-wasmvm-v1.5](https://github.com/cosmos/ibc-go/releases/tag/modules%2Flight-clients%2F08-wasm%2Fv0.3.0%2Bibc-go-v7.3-wasmvm-v1.5) - 2024-07-29

### API Breaking

* [\#6924](https://github.com/cosmos/ibc-go/pull/6924) The JSON message API for `VerifyMembershipMsg` and `VerifyNonMembershipMsg` payloads for client contract `SudoMsg` has been updated. The field `path` has been changed to `merkle_path`. This change requires updates to 08-wasm client contracts for integration.

<!-- markdown-link-check-disable-next-line -->
## [v0.2.0+ibc-go-v7.3-wasmvm-v1.5](https://github.com/cosmos/ibc-go/releases/tag/modules%2Flight-clients%2F08-wasm%2Fv0.2.0%2Bibc-go-v7.3-wasmvm-v1.5) - 2024-07-17

### API Breaking

* [\#6644](https://github.com/cosmos/ibc-go/pull/6644) Add `v2.MerklePath` for contract api `VerifyMembershipMsg` and `VerifyNonMembershipMsg` structs. Note, this requires a migration for existing client contracts to correctly handle deserialization of `MerklePath.KeyPath` which has changed from `[]string` to `[][]byte`. In JSON message structures this change is reflected as the `KeyPath` being a marshalled as a list of base64 encoded byte strings. This change supports proving values stored under keys which contain non-utf8 encoded symbols. See migration docs for more details.

### Features

* [#\6231](https://github.com/cosmos/ibc-go/pull/6231) feat: add CLI to broadcast transaction with `MsgMigrateContract`.

### Bug Fixes

* [\#6815](https://github.com/cosmos/ibc-go/pull/6815) Decode to bytes the hex-encoded checksum argument of the `migrate-contract` CLI. 

<!-- markdown-link-check-disable-next-line -->
## [v0.1.1+ibc-go-v7.3-wasmvm-v1.5](https://github.com/cosmos/ibc-go/releases/tag/modules%2Flight-clients%2F08-wasm%2Fv0.1.1%2Bibc-go-v7.3-wasmvm-v1.5) - 2024-04-12

### Dependencies

* [\#6149](https://github.com/cosmos/ibc-go/pull/6149) Bump wasmvm to v1.5.2.

### Bug Fixes

* (cli) [\#5610](https://github.com/cosmos/ibc-go/pull/5610) Register wasm tx cli.

<!-- markdown-link-check-disable-next-line -->
## [v0.1.0+ibc-go-v7.3-wasmvm-v1.5](https://github.com/cosmos/ibc-go/releases/tag/modules%2Flight-clients%2F08-wasm%2Fv0.1.0%2Bibc-go-v7.3-wasmvm-v1.5) - 2023-12-18

### Features

* [\#5079](https://github.com/cosmos/ibc-go/pull/5079) feat: 08-wasm light client proxy module for wasm clients.
