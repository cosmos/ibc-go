---
title: Roadmap
sidebar_label: Roadmap
sidebar_position: 10
slug: /ibc/roadmap
---

# Roadmap ibc-go

*Latest update: June 7th, 2024*

This document endeavours to inform the wider IBC community about plans and priorities for work on ibc-go by the team at Interchain GmbH. It is intended to broadly inform all users of ibc-go, including developers and operators of IBC, relayer, chain and wallet applications.

This roadmap should be read as a high-level guide, rather than a commitment to schedules and deliverables. The degree of specificity is inversely proportional to the timeline. We will update this document periodically to reflect the status and plans. For the latest expected release timelines, please check [here](https://github.com/cosmos/ibc-go/wiki/Release-timeline).

## v9.0.0

### ICS20 v2

The transfer application will be updated to add support for [transferring multiple tokens in the same packet](https://github.com/cosmos/ibc/pull/1020) and support for [atomically route tokens series of paths with a single packet](https://github.com/cosmos/ibc/pull/1090). 

## v10.0.0

### ICA v2

This new version of ICS27 will address many of [the pain points with the current design](https://github.com/cosmos/ibc-go/pull/6281), including multiplexing all communication between controller and host through a single channel (instead of each interchain account on the host being associated to a different channel, as it is now).

### Multipacket atomicity

We will refactor the 05-port router to enable atomic sending of multiple packets belonging to different applications.

---

And potentially later on...

#### Multihop channels

Multihop channels specify a way to route messages across a path of IBC enabled blockchains utilizing multiple pre-existing IBC connections. The current IBC protocol defines messaging in a point-to-point paradigm which allows message passing between two directly connected IBC chains, but as more IBC enabled chains come into existence there becomes a need to relay IBC packets across chains because IBC connections may not exist between the two chains wishing to exchange messages. IBC connections may not exist for a variety of reasons which could include economic inviability since connections require client state to be continuously exchanged between connection ends which carries a cost. Please see the [ICS 33 spec](https://github.com/cosmos/ibc/blob/main/spec/core/ics-033-multi-hop/README.md) for more information.

---

This roadmap is also available as a [project board](https://github.com/orgs/cosmos/projects/7/views/25).

For the latest expected release timelines, please check [here](https://github.com/cosmos/ibc-go/wiki/Release-timeline).

For the latest information on the progress of the work or the decisions made that might influence the roadmap, please follow the [Announcements](https://github.com/cosmos/ibc-go/discussions/categories/announcements) category in the Discussions tab of the repository.

---

**Note**: release version numbers may be subject to change.
