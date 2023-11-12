---
title: Roadmap
sidebar_label: Roadmap
sidebar_position: 9
slug: /ibc/roadmap
---

# Roadmap ibc-go

*Lastest update: September 12th, 2023*

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go by the team at Interchain GmbH. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans. For the latest expected release timelines, please check [here](https://github.com/cosmos/ibc-go/wiki/Release-timeline).

## v8.0.0

Follow the progress with the [milestone](https://github.com/cosmos/ibc-go/milestone/38).

This release main additions are:

- Upgrade to Cosmos SDK v0.50.
- [Migration of gov proposals from v1beta1 to v1](https://github.com/cosmos/ibc-go/issues/1282).
- [Migration of params to be self managed](https://github.com/cosmos/ibc-go/issues/2010).

## 08-wasm/v0.1.0

Follow the progress with the [milestone](https://github.com/cosmos/ibc-go/milestone/40).

The first release of this new module will add support for Wasm light clients. The first Wasm client developed on top of ibc-go/v7 02-client refactor and stored as Wasm bytecode will be the GRANDPA light client used for Cosmos x Substrate IBC connections. This feature will be used also for a NEAR light client in the future.

This feature has been developed by Composable and Strangelove.

## v9.0.0

### Channel upgradability

Channel upgradability will allow chains to renegotiate an existing channel to take advantage of new features without having to create a new channel, thus preserving all existing packet state processed on the channel. This feature will enable, for example, the adoption of existing channels of features like [path unwinding](https://github.com/cosmos/ibc/discussions/824) or fee middleware.

Follow the progress with the [alpha milestone](https://github.com/cosmos/ibc-go/milestone/29) or the [project board](https://github.com/orgs/cosmos/projects/7/views/17).

---

This roadmap is also available as a [project board](https://github.com/orgs/cosmos/projects/7/views/25).

For the latest expected release timelines, please check [here](https://github.com/cosmos/ibc-go/wiki/Release-timeline).

For the latest information on the progress of the work or the decisions made that might influence the roadmap, please follow the [Announcements](https://github.com/cosmos/ibc-go/discussions/categories/announcements) category in the Discussions tab of the repository.

---

**Note**: release version numbers may be subject to change.
