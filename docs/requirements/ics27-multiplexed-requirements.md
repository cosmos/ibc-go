<!-- More detailed information about the requirements engineering process can be found at https://github.com/cosmos/ibc-go/wiki/Requirements-engineering -->

# Business requirements

The Interchain Accounts feature has enabled control of an account on a host chain remotely from a controller chain. This has opened up many possibilities for more complex cross-chain workflows beyond token transfer alone, but the current architecture is not easy to use for the case where there are on the order of 1000s of host accounts controlled by many controllers. These requirements aim to alleviate current pain points and enhance the usability of the feature.

## Problem

The current pain points for existing ICA users are listed:

- *Account Flexibility* - Currently host accounts can only be of type `BaseAccount`

- *Ordered Channels* - Timeouts and subsequent channel closures are time-consuming and difficult to manage

- *Scalability* - When there are many interchain accounts, and many channels (in the order of 1000s), relayers struggle with the number of queries on startup and performance is compromised. Additionally, account registration being linked to the channel means that multiple packet round trips are required before your host account can execute messages.

- *Workflows requiring tokens in ICA* - there are no atomic guarantees for a workflow containing multiple applications - i.e. a token transfer followed by an ICA message. The transfer could succeed and the ICA message fail, resulting in an incomplete workflow.

- *Lack of default callbacks* - users are still relying on custom auth modules for packet callbacks, the callbacks middleware solves this problem but is an add-on to the application rather than a default

*Whitelisting messages* - this devex is more complicated than having a blacklist

## Objectives

- Enable different account types, to be controlled remotely
- To streamline workflows using token transfer and interchain accounts in combination
- To enable scalable and efficient account creation

## Scope

| Features  | Release |
| --------- | ------- |
| Multi-plexed Interchain Accounts | ibc-go version tbd |

# User requirements

## Use cases

Existing use cases have been detailed in ics27 and ics27 v2 requirements, some other notable use cases used in production are cross chain liquid staking, yield through leveraged lending and cross chain NFT minting.

### Liquid Staking

Chains such as Stride, Quicksilver and pStake control Interchain Accounts on a host chain to stake tokens on behalf of their users and mint representative liquid staking tokens so that users can transact with liquid tokens, rather than a locked staked asset but still earn staking rewards through autocompounding on the Interchain Account. More information on Stride's technical architecture can be found [here](https://github.com/Stride-Labs/stride/tree/main?tab=readme-ov-file#strides-technical-architecture).

### Leveraged Lending

Nolus enables users to borrow assets and use the inflation from staking rewards to repay the interest on the principle. e.g. a user could deposit 10 OSMO as collateral for a 25 OSMO position. When a user opens a position, an Interchain Account is opened on Osmosis and the loan amount is sent to the account with management of the account through the Nolus chain.

### Cross-chain NFT minting

Nomos enable NFTs to be minted on Injective on host accounts controlled from the Xion chain, providing a single interface for users, enabling chain abstraction.

# Functional requirements

## Assumptions and dependencies

- Although having atomic transfer plus action workflows with Interchain Accounts is desirable, it is out of scope for these requirements, as a solution is applicable to applications beyond Interchain Accounts alone. 
Migration of an existing Interchain Account using a prior version of Interchain Accounts that does not satisfy these requirements is not required.
- Assumes use of the `x/accounts` sdk module.

## Features

### 1 - Configuration

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 1.01 | The host chain should accept all message types by default and maintain a blacklist of message types it does not permit | ------------ | ------ |

### 2 - Registration

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 2.01 | The controller of the interchain account must have authority over the account on the host chain to execute messages | -- | ----------- |
| 2.02 | A registered interchain account can be any account type supported by `x/accounts` | ------------ | ------ |

### 3 - Control

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 3.01 | The channel type through which a controller sends transactions to the host should be unordered | ------------ | ------ |
| 3.02 | The message execution order should be determined at the packet level | ------------ | ------ |
| 3.03 | Many controllers can send messages to many host accounts through the same channel | ------------ | ------ |
| 3.04 | The controller of the interchain account should be able to receive information about the balance of the interchain account in the acknowledgment after a transaction was executed by the host | ------------ | ------ |
| 3.05 | The user of the controller should be able to receive all the information contained in the acknowledgment without implementing additional middleware on a per-user basis | ------------ | ------ |
| 3.06 | Callbacks on the packet lifecycle should be supported by default | ------------ | ------ |
| 3.07 | A user can perform module safe queries through a host chain account and return the result in the acknowledgment | ------------ | ------ |  

### 4 - Host execution

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 4.01 | It should be possible to ensure a packet lifecycle from a different application completes before a message from a controller is executed | ------------ | ------ |
| 4.02 | It should be possible for a controller to authorise a host account to execute specific actions on a host chain without needing a packet round trip each time (e.g. auto-compounding) | ------------ | ------ |

# Non-functional requirements

## 5 - Performance

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 5.01 | The number of packet round trips to register an account, load the account with tokens and execute messages on the account should be minimised | ------------ | ------ |
