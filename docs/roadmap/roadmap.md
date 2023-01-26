---
order: 1
---

# Roadmap ibc-go

_Lastest update: October 3, 2022_

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go by the team at Interchain GmbH. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans.

## v6.0.0

### Interchain Accounts improvements

These improvements tackle two shortcomings of the current implementation of ICS-27:

- The lack of a default/standard underlying app (also known as _authentication module_), which created a bottleneck for chains looking to integrate the controller submodule.
- The need to separate application and authentication concerns.

The changes will make integrating the controller chain functionality easier (see [ADR 009](../architecture/adr-009-v6-ics27-msgserver.md) for more information), and with the release of Cosmos SDK v0.46.x, they will enable any Cosmos SDK module (e.g `x/auth`, `x/gov`, `x/group`) to register interchain accounts and send transactions on their behalf. 

For more details about these changes, please read [this blog post](https://medium.com/the-interchain-foundation/ibc-go-v6-changes-to-interchain-accounts-and-how-it-impacts-your-chain-806c185300d7).

### Other features/improvements

- ICS-20:
  - Adding a `metadata` field in `FungibleTokenPacketData` (implementation of [this](https://github.com/cosmos/ibc/pull/842) spec update).
  - Adding the sequence number of the packet sent to `MsgTransferResponse`.
- 04-channel:
  - Simplify `SendPacket` API (implementation of [this](https://github.com/cosmos/ibc/pull/731) spec update).

Follow the progress with the [`v6.0.0` milestone](https://github.com/cosmos/ibc-go/milestone/35) or the [project board](https://github.com/orgs/cosmos/projects/7/views/23).

## v7.0.0

### 02-client refactor

This refactor will make the development of light clients easier. The ibc-go implementation will finally align with the spec and light clients will be required to set their own client and consensus states. This will allow more flexibility for light clients to manage their own internal storage and do batch updates. See [ADR 006](../architecture/adr-006-02-client-refactor.md) for more information.

Follow the progress with the [beta](https://github.com/cosmos/ibc-go/milestone/25) and [RC](https://github.com/cosmos/ibc-go/milestone/27) milestones or in the [project board](https://github.com/orgs/cosmos/projects/7/views/14).

### Localhost connection

This feature will add support for applications on a chain to communicate with applications on the same chain using the existing standard interface to communicate with applications on remote chains.

For more details, see the design proposal and discussion [here](https://github.com/cosmos/ibc-go/discussions/2191). Issues need still to be created and will be added to the [`v7.0.0` milestone](https://github.com/cosmos/ibc-go/milestone/34).

### Upgrade SDK v0.47.x

Follow the progress with the [milestone](https://github.com/cosmos/ibc-go/milestone/36).

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
