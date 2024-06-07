---
title: Introduction
sidebar_label: Introduction
sidebar_position: 1
slug: /fee/intro
---

import HighlightTag from '@site/src/components/HighlightTag';
import HighlightBox from '@site/src/components/HighlightBox';

# Introduction

<HighlightTag type="ibc-go" version="v7"/> <HighlightTag type="cosmos-sdk" version="v0.47"/> <HighlightTag type="cosmjs"/> <HighlightTag type="guided-coding"/>

This is a tutorial for wiring up the ICS-29 Fee Middleware to a Cosmos SDK blockchain and a React frontend.

<HighlightBox type="prerequisite" title="Prerequisites">

- Basic Knowledge of [TypeScript](https://www.typescriptlang.org/)
- Basic Knowledge of Cosmos SDK
    - If you are new to Cosmos SDK, we recommend you to go through the first two categories of the [Developer Portal](https://tutorials.cosmos.network/academy/1-what-is-cosmos/)
- Basic Knowledge of [the Fee Middleware module](https://ibc.cosmos.network/main/middleware/ics29-fee/overview)

</HighlightBox>

## Scope

This tutorial will cover creating a Cosmos SDK blockchain with the ICS-29 Fee Middleware wired up to it. It will also cover creating a React frontend that can interact with the blockchain.

<HighlightBox type="learning" title="Learning Goals">

In this tutorial, you will:

- Create an IBC enabled blockchain using the Cosmos SDK.
- Wire up the ICS-29 Fee Middleware to the blockchain.
- Create a React frontend that can interact with the blockchain.
- Run two blockchains locally and connect them using the Hermes relayer.
- Make an incentivized transfer between the two blockchains.

</HighlightBox>
