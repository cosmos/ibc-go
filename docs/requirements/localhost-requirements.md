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
| To interface with applications or smart contracts on a blockchain through the ibc application layer | v1 |
| To use the localhost client without requiring third party relayer infrastructure | v2 |


# User requirements

## Use cases

### Interacting with Smart Contracts

Many IBC connected blockchains utilise smart contracts, these could be written in rust, solidity or javascript depending on the smart contract platform being used.  Through the localhost client, a user can interact with and call into smart contracts by sending ibc messages from the localhost client to the smart contract. 

On Agoric, a planned use case is for managing delegations and staking with smart contracts on a given chain. MsgDelegate, MsgUndelegate, MsgBeginRedelegate messages would be sent using the localhost client to the staking module. The same interface could be used the manage staking cross chain, sending the same messages through IBC to an interchain account.

### Interoperability Infrastructure

Polymer plan to leverage the localhost client with multiple connections as part of their architecture to connect chains not using tendermint or the cosmos sdk to IBC enabled chains. Polymer will act as a hub connecting multiple blockchains together. 


# Functional requirements

<!-- They should describe as completely as necessary the system's behaviors under various conditions. They describe what the engineers must implement to enable users to accomplish their tasks (user requirements), thereby satisfying the business requirements. Software engineers don't implement business requirements or user requirements. They implement functional requirements, specific bits of system behavior. Each requirement should be uniquely identified with a meaningful tag. -->

## Assumptions and dependencies

1. Chains utilising the localhost client must be using an ibc-go release in the v7 line
2. The channel behaviour of a localhost client will be as ics 004 specifies
<!-- List any assumed factors that could affect the requirements. The project could be affected if these assumptions are incorrect, are not shared, or change. Also identify any dependencies the project has on external factors. -->

## Features
### 1 - Configuration
| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 1.01 | The localhost client shall have a clientID of the string "09-localhost" | ------------ | Drafted | 
| 1.02 | The localhost client shall have a sentinel connectionID of the string "connection-localhost"| ------------ | Drafted | 
| 1.03| If more than 1 localhost connection is required, this is possible with a different connectionID | ------------ | Drafted |
| 1.04 | When the localhost client is initialised the consensus state must be nil | ------------ | Drafted |
| 1.05| The localhost client can be added to a chain through an upgrade | ------------ | Drafted |
| 1.06| A chain can enable the localhost client by initialising the client in the genesis state | ------------ | Drafted |

### 2 - Operation
| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 2.01 | A user of the localhost client can send ibc messages to an application on the same chain | ------------ | Drafted| 
| 2.02 | A user can use the localhost client through the existing ibc application module interfaces | ------------ | Drafted| 


# External interface requirements

### 3 - CLI
| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ | 
| 3.01 | Existing CLI interfaces used with ibc application modules shall be useable with the localhost client | ------------ | Drafted | 

<!-- They describe the interfaces to other software systems, hardware components, and users. Ideally they should state the purpose, format and content of messages used for input and output. -->

# Non-functional requirements

The user should not be explicitly concerned with the latency of transactions they send through a localhost client interface.
<!-- Other-than-functional requirements that do not specify what the system does, but rather how well it does those things. For example: quality requirements: performance, security, portability, etc. -->