<!-- More detailed information about the requirements engineering process can be found at https://github.com/cosmos/ibc-go/wiki/Requirements-engineering -->

# Business requirements

The local host client provides a unified interface to interact with different applications on a single chain.

## Problem

Currently applications, or smart contracts, on a single chain cannot communicate through a single interface. This means a user must interact with each application or smart contract directly which can be cumbersome, especially when a user wants to interact with many applications on one chain.

## Objectives

To provide a single interface in line with IBC application layer semantics, that enables a user to interact with multiple different applications or smart contracts remotely on a given blockchain. 

## Scope

| Features  | Release |
| --------- | ------- |
| To interface with applications or smart contracts on a blockchain through the IBC application layer | v1 |
| To use the localhost client without requiring third party relayer infrastructure | v2 |

# User requirements

## Use cases

### Interacting with Smart Contracts

Many IBC connected blockchains utilise smart contracts, these could be written in rust, solidity or javascript depending on the smart contract platform being used.  Through the localhost client, a user can interact with and call into smart contracts by sending ibc messages from the localhost client to the smart contract. 

On Agoric, a planned use case is for managing delegations and staking with smart contracts on a given chain. MsgDelegate, MsgUndelegate, MsgBeginRedelegate messages would be sent using the localhost client to the staking module. The same interface could be used to manage staking cross chain, sending the same messages through IBC to an interchain account.

### Interoperability Infrastructure

Polymer plans to leverage the localhost client with multiple connections as part of their architecture to connect chains not using Tendermint or the Cosmos SDK to IBC enabled chains. Polymer will act as a hub connecting multiple blockchains together. 

# Functional requirements

## Assumptions and dependencies

1. Chains utilising the localhost client must be using an ibc-go release in the v7 line.
2. The channel behaviour of a localhost client will be as ICS 04 specifies.

## Features

### 1 - Configuration

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 1.01 | The localhost client shall have a client ID of the string `09-localhost`. | ------------ | Drafted | 
| 1.02 | The localhost client shall have a sentinel connection ID of the string `connection-localhost`. | ------------ | Drafted | 
| 1.03 | If more than 1 localhost connection is required, this is possible with a different connection ID. | ------------ | Drafted |
| 1.04 | When the localhost client is initialised the consensus state must be `nil`. | ------------ | Drafted |
| 1.05 | The localhost client can be added to a chain through an upgrade. | ------------ | Drafted |
| 1.06 | A chain can enable the localhost client by initialising the client in the genesis state. | ------------ | Drafted |

### 2 - Operation

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 2.01 | A user of the localhost client can send IBC messages to an application on the same chain. | ------------ | Drafted| 
| 2.02 | A user can use the localhost client through the existing IBC application module interfaces. | ------------ | Drafted| 


# External interface requirements

### 3 - CLI

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ | 
| 3.01 | Existing CLI interfaces used with IBC application modules shall be useable with the localhost client. | ------------ | Drafted | 
