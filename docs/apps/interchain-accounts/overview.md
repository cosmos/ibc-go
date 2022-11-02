<!--
order: 1
-->

# Overview

Learn about what the Interchain Accounts module is {synopsis}

## What is the Interchain Accounts module?

Interchain Accounts is the Cosmos SDK implementation of the ICS-27 protocol, which enables cross-chain account management built upon IBC.

- How does an interchain account differ from a regular account?

Regular accounts use a private key to sign transactions. Interchain Accounts are instead controlled programmatically by counterparty chains via IBC packets.

## Concepts 

`Host Chain`: The chain where the interchain account is registered. The host chain listens for IBC packets from a controller chain which should contain instructions (e.g. Cosmos SDK messages) for which the interchain account will execute.

`Controller Chain`: The chain registering and controlling an account on a host chain. The controller chain sends IBC packets to the host chain to control the account.

`Interchain Account`: An account on a host chain created using the ICS-27 protocol. An interchain account has all the capabilities of a normal account. However, rather than signing transactions with a private key, a controller chain will send IBC packets to the host chain which signals what transactions the interchain account should execute.

#### Deprecated

`Authentication Module`: A custom IBC application module on the controller chain that uses the Interchain Accounts module API to build custom logic for the creation & management of interchain accounts. For a controller chain to utilize the interchain accounts module functionality, an authentication module is required.

## SDK Security Model

SDK modules on a chain are assumed to be trustworthy.  For example, there are no checks to prevent an untrustworthy module from accessing the bank keeper.

The implementation of ICS-27 in ibc-go uses this assumption in its security considerations.

The implementation assumes other IBC application modules will not bind to ports within the ICS-27 namespace. 
