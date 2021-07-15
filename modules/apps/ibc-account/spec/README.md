<!--
order: 0
title: "Overview"
parent:
  title: "IBCAccount"
-->

# IBC Account

## Abstract

This document specifies the IBC account module for the Cosmos SDK.

The IBCAccount module manages the creation of IBC Accounts and ICS20 packet handling for the IBC accounts. This module is built based on the [ICS27 specification](https://github.com/cosmos/ics/tree/master/spec/ics-027-interchain-accounts). IBC Accounts allow a remote, IBC-connected **source blockchain** to request an arbitrary transaction to be executed on the **destination blockchain**(the chain which hosts the IBC account) via the IBC account. It should be noted that an IBC account has similar properties to a user account, and are bound to the same restrictions (unbonding periods, redelegation rules, etc).

The current implementation allows the same IBCAccount module on the destination chain to run any of the domiciling blockchain's native transactions that a user account is able to request(i.e. same module can handle 'send', 'stake', 'vote', etc), but the controlling chain/source chain must implement its own logic for controlling the IBC account from its own IBCAccount logic.

## Contents
1. **[Types](03_types.md)**
2. **[Keeper](04_keeper.md)**
3. **[Packets](05_packets.md)**
