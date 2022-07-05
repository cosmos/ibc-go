---
order: 1
---

# Roadmap ibc-go

_Lastest update: March 31, 2022_

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go by the team at Interchain GmbH. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans.

## Q2 - 2022

At a high level we will focus on:

- Finishing the implementation of [relayer incentivisation](https://github.com/orgs/cosmos/projects/7/views/8).
- Finishing the [refactoring of 02-client](https://github.com/cosmos/ibc-go/milestone/16).
- Finishing the upgrade to Cosmos SDK v0.46 and Tendermint v0.35.
- Implementing and testing the changes needed to support the [transtion to SMT storage](https://github.com/cosmos/ibc-go/milestone/21) in the Cosmos SDK.
- Desiging the implementation and scoping the engineering work for [channel upgradability](https://github.com/cosmos/ibc/blob/master/spec/core/ics-004-channel-and-packet-semantics/UPGRADES.md).
- Improving the project's documentation and writing guides for [light client](https://github.com/cosmos/ibc-go/issues/59) and middleware implementation.
- Working on [core backlog issues](https://github.com/cosmos/ibc-go/milestone/8).
- Spending time on expanding and deepening our knowledge of IBC, but also other parts of the Cosmos stack.
- And last, but not least, onboarding new members to the team.

For a detail view of each iteration's planned work, please check out our [project board](https://github.com/orgs/cosmos/projects/7).

### Release schedule

#### **April**

In the first half of the month we will probably cut:

- Alpha/beta pre-releases with the upgrade to SDK 0.46 and Tendermint v0.35.
- [Alpha](https://github.com/cosmos/ibc-go/milestone/5) pre-release with the implementation of relayer incentivisation.

In the second half, and depending on the date of the final release of Cosmos SDK 0.46, we will probably cut the final release with the upgrade to SDK 0.46 and Tendermint v0.35, and also a [beta](https://github.com/cosmos/ibc-go/milestone/23) pre-release with the implementation of relayer incentivisation.

In the second half of the month we also plan to do a second internal audit of the implementation of relayer incentivisation, and issues will most likely will be created from the audit. Depending on the nature and type of the issues we create, those would be released in a second beta pre-release or in a [release candidate](https://github.com/cosmos/ibc-go/milestone/24).

#### **May**

In the first half we will probably start cutting release candidates with relayer incentivisation and the 02-client refactor. Final release would most likely come out at the end of the month or beginning of June.

#### **June**

We will probably cut at the end of the month or beginning of Q3 patch or minor releases on all the supported release lines with the [small features and core improvements](https://github.com/cosmos/ibc-go/milestone/8) that we work on during the quarter.

## Q3 - 2022

We will most likely start the implementation of [channel upgradability](https://github.com/cosmos/ibc/blob/master/spec/core/ics-004-channel-and-packet-semantics/UPGRADES.md). At the end of Q2 or maybe beginning of Q3 we might also work on designing the implementation and scoping the engineering work to add support for [ordered channels that can timeout](https://github.com/cosmos/ibc/pull/636), and we could potentially work on this feature also in Q3.

We will also probably do an audit of the implementation of the [CCV application](https://github.com/cosmos/interchain-security/tree/main/x/ccv) for Interchain Security.

### Release schedule

In this quarter we will make the final release to support the migration to SMT storage.