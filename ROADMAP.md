# Roadmap ibc-go

_Lastest update: Nov 24, 2021_

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans.

The release tags and timelines are educated guesses based on the information at hand at the moment of updating this document. The release version numbers might change if we need to release security vulnerability patches or urgent bug fixes.

## Q4 - 2021

### Interchain accounts

- Finalize the issues raised during the internal audit.
- Prepare codebase & specification for two external audits.
- Write developer documentation and update demo repository.
- Integration with hermes relayer and end-2-end testing.
- Create release candidate.

### Relayer incentivisation

- Finalize implementation.
- Update specification and write documentation.
- Do internal audit and write issues that may arise.

### Wasm light client

There is an open [PR](https://github.com/cosmos/ibc-go/pull/208) that implements support for Was-based light clients, but it needs to be updated after the finalization of the [ICS28](https://github.com/cosmos/ibc/tree/master/spec/client/ics-008-wasm-client) specification. The PR will also need a final review from ibc-go core team members.

Besides the above mentioned PR, there are [a](https://github.com/cosmos/ibc-go/issues/284) [few](https://github.com/cosmos/ibc-go/issues/285) [issues](https://github.com/cosmos/ibc-go/issues/286) that are also required. These issues will bring the ibc-go implementation in line with [ICS02](https://github.com/cosmos/ibc/tree/master/spec/core/ics-002-client-semantics).

### Release schedule

#### Past

|Release|Milestone|Date|
|-------|---------|----|
|[v1.1.0](https://github.com/cosmos/ibc-go/releases/tag/v1.1.1)||Oct 04, 2021|
|[v1.2.1](https://github.com/cosmos/ibc-go/releases/tag/v1.2.1)||Oct 04, 2021|
|[v2.0.0-rc0](https://github.com/cosmos/ibc-go/releases/tag/v2.0.0-rc0)|[Link](https://github.com/cosmos/ibc-go/milestone/3)|Oct 05, 2021|
|[v1.1.2](https://github.com/cosmos/ibc-go/releases/tag/v1.1.2)||Oct 15, 2021|
|[v1.2.2](https://github.com/cosmos/ibc-go/releases/tag/v1.2.2)||Oct 15, 2021|
|[v1.1.3](https://github.com/cosmos/ibc-go/releases/tag/v1.1.3)||Nov 09, 2021|
|[v1.2.3](https://github.com/cosmos/ibc-go/releases/tag/v1.2.3)||Nov 09, 2021|
|[v2.0.0](https://github.com/cosmos/ibc-go/releases/tag/v2.0.0)|[Link](https://github.com/cosmos/ibc-go/milestone/3)|Nov 09, 2021|

#### Future

|Milestone|Timeline|Notes|
|---------|--------|-----|
|[v2.0.1](https://github.com/cosmos/ibc-go/milestone/11)|H2 Nov||
|[v2.1.0-rc0]()|H2 Dec|Release candidate of Interchain Accounts.|

## Q1 - 2022

### Interchain accounts 

- Work on any issues that may come out of the two external audits.
- Create final release.

### Relayer incentivisation

- Work on issues that may arise from internal audit.
- External audit (issues may arise that we need to work on before release).
- Create release candidate (if needed) and final release.

### Interchain security

-  Testnet testing of [V1](https://github.com/cosmos/gaia/blob/main/docs/interchain-security.md#v1---full-validator-set).

### Technical debt

- [#545](https://github.com/cosmos/ibc-go/issues/545): Remove the `GetTransferAccount` function, since we never use the ICS20 transfer module account (every escrow address is created as a regular account).
- Changes needed to support the migration to SMT storage. This is basically adding a new proof spec that will be used during connection handshake with a chain that has migrated to SMT to verify that the light client of the counterparty chain uses the correct proof specs to be able to verify proofs for that chain.
- And more to be added later!

### Release schedule

#### Future

|Milestone|Timeline|Notes|
|---------|--------|-----|
|[v2.0.2](https://github.com/cosmos/ibc-go/milestone/14)|H2 Jan||
|[v2.1.0](https://github.com/cosmos/ibc-go/milestone/15)|H1 Feb|Final release of Interchain Accounts.|
|[v2.2.0-rc0](https://github.com/cosmos/ibc-go/milestone/16)|H1 Feb|Release candidate of Relayer Incentivisation.|
|[v1.3.0](https://github.com/cosmos/ibc-go/milestone/19)|H2 Feb|Dependencies update: [Cosmos SDK v0.45](https://github.com/cosmos/cosmos-sdk/milestone/46) and [Tendermint v0.35](https://github.com/tendermint/tendermint/releases/tag/v0.35.0).|
|[v2.2.0](https://github.com/cosmos/ibc-go/milestone/17)|H1 Mar|Release of Relayer Incentivisation.|
|[v2.3.0](https://github.com/cosmos/ibc-go/milestone/18)|H1 Mar|Dependencies update: [Cosmos SDK v0.45](https://github.com/cosmos/cosmos-sdk/milestone/46) and [Tendermint v0.35](https://github.com/tendermint/tendermint/releases/tag/v0.35.0).|
|[v3.0.0](https://github.com/cosmos/ibc-go/milestone/13)|H2 Mar|Dependencies update: Golang v1.17 and [Cosmos SDK v1.0](https://github.com/cosmos/cosmos-sdk/milestone/52). This release will support the migration to SMT storage.|

## Q2 - 2022

Scope is still unclear, but it might possibly include the start of the implementation of the [ICS721](https://github.com/cosmos/ibc/pull/615) application.