<!--
order: 0
title: "Overview"
parent:
  title: "Interchain Accounts"
-->

# Interchain Account

## Abstract

This document specifies the ICS 27 Interchain Account module for the Cosmos SDK.

The Interchain Accounts module manages the creation of Interchain Accounts. This module is built based on the [ICS27 specification](https://github.com/cosmos/ibc/tree/master/spec/app/ics-027-interchain-accounts). Interchain Accounts allow a remote, IBC-connected **controller blockchain** to request an arbitrary transaction to be executed on the **host blockchain**(the chain which hosts the IBC account) via the interchain account. It should be noted that an interchain account has similar properties to a user account, and are bound to the same restrictions (unbonding periods, redelegation rules, etc).

The current implementation allows the same interchain account module on the destination chain to run any of the domiciling blockchain's native transactions that a user account is able to request(i.e. same module can handle 'send', 'stake', 'vote', etc), but the controlling chain/source chain must implement its own logic for controlling the interchain account.

## Contents
1. **[Types](03_types.md)**
2. **[Keeper](04_keeper.md)**
3. **[Packets](05_packets.md)**
