---
title: Introduction
sidebar_label: Introduction
sidebar_position: 1
slug: /channel-upgrades/intro
---

import HighlightTag from '@site/src/components/HighlightTag';
import HighlightBox from '@site/src/components/HighlightBox';

# Introduction

<HighlightTag type="ibc-go" version="v8.1"/> <HighlightTag type="cosmos-sdk" version="v0.50"/>

This is a tutorial for upgrading an existing ICS 20 transfer channel to wrap it with the ICS 29 Fee Middleware.

<HighlightBox type="prerequisite" title="Prerequisites">

- Basic Knowledge of Cosmos SDK.
    - If you are new to Cosmos SDK, we recommend you to go through the first two categories of the [Developer Portal](https://tutorials.cosmos.network/academy/1-what-is-cosmos/).
- Basic Knowledge of [the Fee Middleware module](https://ibc.cosmos.network/main/middleware/ics29-fee/overview).
- Basic knowledge of [channel upgrades](https://ibc.cosmos.network/main/ibc/channel-upgrades).

</HighlightBox>

## Scope

This tutorial will cover the process of upgrading an existing ICS 20 transfer channel to add packet incentivization using the Fee Middleware.

<HighlightBox type="learning" title="Learning Goals">

In this tutorial, you will:

- Run two IBC-enabled blockchains locally.
- Open an ICS 20 transfer channel using the Hermes relayer.
- Upgrade the ICS 20 transfer channel to add ICS-29 Fee Middleware.

</HighlightBox>
