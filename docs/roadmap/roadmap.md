---
order: 1
---

# Roadmap ibc-go

<<<<<<< HEAD
_Lastest update: July 7, 2022_
=======
_Lastest update: December 21st, 2022_
>>>>>>> main

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go by the team at Interchain GmbH. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans. For the latest expected release timelines, please check [here](https://github.com/cosmos/ibc-go/wiki/Release-timeline).

<<<<<<< HEAD
## Q3 - 2022
=======
## v7.0.0
>>>>>>> main

### 02-client refactor

<<<<<<< HEAD
### Features

- Releasing [v4.0.0](https://github.com/cosmos/ibc-go/milestone/26), which includes the ICS-29 Fee Middleware module.
- Finishing and releasing the [refactoring of 02-client](https://github.com/cosmos/ibc-go/milestone/16). This refactor will make the development of light clients easier.
- Starting the implementation of channel upgradability (see [epic](https://github.com/cosmos/ibc-go/issues/1599) and [alpha milestone](https://github.com/cosmos/ibc-go/milestone/29)) with the goal of cutting an alpha1 pre-release by the end of the quarter. Channel upgradability will allow chains to renegotiate an existing channel to take advantage of new features without having to create a new channel, thus preserving all existing packet state processed on the channel.
- Implementing the new [`ORDERED_ALLOW_TIMEOUT` channel type](https://github.com/cosmos/ibc-go/milestone/31) and hopefully releasing it as well. This new channel type will allow packets on an ordered channel to timeout without causing the closure of the channel.

### Testing and infrastructure

- Adding [automated e2e tests](https://github.com/cosmos/ibc-go/milestone/32) to the repo's CI.

### Documentation and backlog

- Finishing and releasing the upgrade to Cosmos SDK v0.46.
- Writing the [light client implementation guide](https://github.com/cosmos/ibc-go/issues/59).
- Working on [core backlog issues](https://github.com/cosmos/ibc-go/milestone/28).
- Depending on the timeline of the Cosmos SDK, implementing and testing the changes needed to support the [transtion to SMT storage](https://github.com/cosmos/ibc-go/milestone/21).

We have also received a lot of feedback to improve Interchain Accounts and we might also work on a few things, but will depend on priorities and availability.
=======
This refactor will make the development of light clients easier. The ibc-go implementation will finally align with the spec and light clients will be required to set their own client and consensus states. This will allow more flexibility for light clients to manage their own internal storage and do batch updates. See [ADR 006](../architecture/adr-006-02-client-refactor.md) for more information.
>>>>>>> main

Follow the progress with the [beta](https://github.com/cosmos/ibc-go/milestone/25) and [RC](https://github.com/cosmos/ibc-go/milestone/27) milestones or in the [project board](https://github.com/orgs/cosmos/projects/7/views/14).

### Upgrade Cosmos SDK v0.47

<<<<<<< HEAD
#### **July**

We will probably cut at least one more release candidate of v4.0.0 before the final release, which should happen around the end of the month. 

For the Rho upgrade of the Cosmos Hub we will also release a new minor version of v3 with SDK 0.46.

#### **August**

In the first half we will probably start cutting release candidates for the 02-client refactor. Final release would most likely come out at the end of the month or beginning of September.

#### **September**

We might cut some pre-releases for the new channel type, and by the end of the month we expect to cut the first alpha pre-release for channel upgradability.

## Q4 - 2022

We will continue the implementation and cut the final release of [channel upgradability](https://github.com/cosmos/ibc/blob/master/spec/core/ics-004-channel-and-packet-semantics/UPGRADES.md). At the end of Q3 or maybe beginning of Q4 we might also work on designing the implementation and scoping the engineering work to add support for [multihop channels](https://github.com/cosmos/ibc/pull/741/files), so that we could start the implementation of this feature during Q4 (but this is still be decided).
=======
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

We will add support for Wasm light clients. The first Wasm client developed with ibc-go/v7 02-client refactor and stored as Wasm bytecode will be the GRANDPA light client used for Cosmos <> Substrate IBC connections. This feature will be used also for a NEAR light client in the future.

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
>>>>>>> main
