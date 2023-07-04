---
title: Roadmap
sidebar_label: Roadmap
sidebar_position: 9
slug: /roadmap/roadmap
---

# Roadmap ibc-go

*Lastest update: July 7, 2022*

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go by the team at Interchain GmbH. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans.

## Q3 - 2022

At a high level we will focus on:

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

For a detail view of each iteration's planned work, please check out our [project board](https://github.com/orgs/cosmos/projects/7).

### Release schedule

#### **July**

We will probably cut at least one more release candidate of v4.0.0 before the final release, which should happen around the end of the month.

For the Rho upgrade of the Cosmos Hub we will also release a new minor version of v3 with SDK 0.46.

#### **August**

In the first half we will probably start cutting release candidates for the 02-client refactor. Final release would most likely come out at the end of the month or beginning of September.

#### **September**

We might cut some pre-releases for the new channel type, and by the end of the month we expect to cut the first alpha pre-release for channel upgradability.

## Q4 - 2022

We will continue the implementation and cut the final release of [channel upgradability](https://github.com/cosmos/ibc/blob/master/spec/core/ics-004-channel-and-packet-semantics/UPGRADES.md). At the end of Q3 or maybe beginning of Q4 we might also work on designing the implementation and scoping the engineering work to add support for [multihop channels](https://github.com/cosmos/ibc/pull/741/files), so that we could start the implementation of this feature during Q4 (but this is still be decided).
