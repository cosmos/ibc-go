---
title: Roadmap
sidebar_label: Roadmap
sidebar_position: 9
slug: /roadmap/roadmap
---

# Roadmap ibc-go

*Lastest update: December 21st, 2022*

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go by the team at Interchain GmbH. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans. For the latest expected release timelines, please check [here](https://github.com/cosmos/ibc-go/wiki/Release-timeline).

## v7.0.0

### 02-client refactor

This refactor will make the development of light clients easier. The ibc-go implementation will finally align with the spec and light clients will be required to set their own client and consensus states. This will allow more flexibility for light clients to manage their own internal storage and do batch updates. See [ADR 006](/architecture/adr-006-02-client-refactor) for more information.

Follow the progress with the [beta](https://github.com/cosmos/ibc-go/milestone/25) and [RC](https://github.com/cosmos/ibc-go/milestone/27) milestones or in the [project board](https://github.com/orgs/cosmos/projects/7/views/14).

### Upgrade Cosmos SDK v0.47

Follow the progress with the [milestone](https://github.com/cosmos/ibc-go/milestone/36).

### Add `authz` support to 20-transfer

Authz goes cross chain: users can grant permission for their tokens to be transferred to another chain on their behalf. See [this issue](https://github.com/cosmos/ibc-go/issues/2431) for more details.

## v7.1.0

Because it is so important to have an ibc-go release compatible with the latest Cosmos SDK release, a couple of features will take a little longer and be released in [v7.1.0](https://github.com/cosmos/ibc-go/milestone/37).

### Localhost connection

This feature will add support for applications on a chain to communicate with applications on the same chain using the existing standard interface to communicate with applications on remote chains. This is a powerful UX improvement, particularly for those users interested in interacting with multiple smart contracts on a single chain through one interface.

For more details, see the design proposal and discussion [here](https://github.com/cosmos/ibc-go/discussions/2191).

A special shout out to Strangelove for their substantial contribution on this feature.

### Support for Wasm light clients

We will add support for Wasm light clients. The first Wasm client developed with ibc-go/v7 02-client refactor and stored as Wasm bytecode will be the GRANDPA light client used for Cosmos x Substrate IBC connections. This feature will be used also for a NEAR light client in the future.

This feature was developed by Composable and Strangelove but will be upstreamed into ibc-go.

## v8.0.0

### Channel upgradability

Channel upgradability will allow chains to renegotiate an existing channel to take advantage of new features without having to create a new channel, thus preserving all existing packet state processed on the channel.

Follow the progress with the [alpha milestone](https://github.com/cosmos/ibc-go/milestone/29) or the [project board](https://github.com/orgs/cosmos/projects/7/views/17).

### Path unwinding

This feature will allow tokens with non-native denoms to be sent back automatically to their native chains before being sent to a final destination chain. This will allow tokens to reach a final destination with the least amount possible of hops from their native chain.

For more details, see this [discussion](https://github.com/cosmos/ibc/discussions/824).

---

This roadmap is also available as a [project board](https://github.com/orgs/cosmos/projects/7/views/25).

For the latest expected release timelines, please check [here](https://github.com/cosmos/ibc-go/wiki/Release-timeline).

For the latest information on the progress of the work or the decisions made that might influence the roadmap, please follow our [engineering updates](https://github.com/cosmos/ibc-go/wiki/Engineering-updates).

---

**Note**: release version numbers may be subject to change.
