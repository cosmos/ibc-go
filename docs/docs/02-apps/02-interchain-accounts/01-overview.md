---
title: Overview
sidebar_label: Overview
sidebar_position: 1
slug: /apps/interchain-accounts/overview
---


# Overview

:::note Synopsis
Learn about what the Interchain Accounts module is
:::

## What is the Interchain Accounts module?

Interchain Accounts is the Cosmos SDK implementation of the ICS-27 protocol, which enables cross-chain account management built upon IBC.

- How does an interchain account differ from a regular account?

Regular accounts use a private key to sign transactions. Interchain Accounts are instead controlled programmatically by counterparty chains via IBC packets.

## Concepts

`Host Chain`: The chain where the interchain account is registered. The host chain listens for IBC packets from a controller chain which should contain instructions (e.g. Cosmos SDK messages) for which the interchain account will execute.

`Controller Chain`: The chain registering and controlling an account on a host chain. The controller chain sends IBC packets to the host chain to control the account.

`Interchain Account`: An account on a host chain created using the ICS-27 protocol. An interchain account has all the capabilities of a normal account. However, rather than signing transactions with a private key, a controller chain will send IBC packets to the host chain which signals what transactions the interchain account should execute.

`Authentication Module`: A custom application module on the controller chain that uses the Interchain Accounts module to build custom logic for the creation & management of interchain accounts. It can be either an IBC application module using the [legacy API](10-legacy/03-keeper-api.md), or a regular Cosmos SDK application module sending messages to the controller submodule's `MsgServer` (this is the recommended approach from ibc-go v6 if access to packet callbacks is not needed). Please note that the legacy API will eventually be removed and IBC applications will not be able to use them in later releases.

## SDK security model

SDK modules on a chain are assumed to be trustworthy. For example, there are no checks to prevent an untrustworthy module from accessing the bank keeper.

The implementation of ICS-27 in ibc-go uses this assumption in its security considerations.

The implementation assumes other IBC application modules will not bind to ports within the ICS-27 namespace.

## Channel Closure

The provided interchain account host and controller implementations do not support `ChanCloseInit`. However, they do support `ChanCloseConfirm`.
This means that the host and controller modules cannot close channels, but they will confirm channel closures initiated by other implementations of ICS-27.
