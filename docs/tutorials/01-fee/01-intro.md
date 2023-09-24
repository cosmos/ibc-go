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

This tutorial for wiring up the ICS-29 Fee Middleware to a Cosmos SDK blockchain and a react frontend.

<HighlightBox type="prerequisite" title="Prerequisites">

- Basic Knowledge of [Go](https://golang.org/doc/tutorial/getting-started)
- Basic Knowledge of [JavaScript](https://developer.mozilla.org/en-US/docs/Web/JavaScript)
- Basic Knowledge of Cosmos SDK
  - If you are new to Cosmos SDK, we recommend you to go through the first two categories of the [Developer Portal](https://tutorials.cosmos.network/academy/1-what-is-cosmos/)

</HighlightBox>

## Scope

This tutorial will cover creating a Cosmos SDK blockchain with the ICS-29 Fee Middleware wired up to it. It will also cover creating a react frontend that can interact with the blockchain.

<HighlightBox type="learning" title="Learning Goals">

In this tutorial, you will:

- Create an IBC enabled blockchain using the Cosmos SDK.
- Wire up the ICS-29 Fee Middleware to the blockchain.
- Create a react frontend that can interact with the blockchain.
- Run two blockchains locally and connect them using the hermes relayer.
- Make an incentivized transfer between the two blockchains.

</HighlightBox>
