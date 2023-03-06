<!-- More detailed information about the requirements engineering process can be found at https://github.com/cosmos/ibc-go/wiki/Requirements-engineering -->

# Business requirements

Rather than create a new channel to expand upon the capabilities of an existing channel, channel upgradability enables new features and capabilities to be added to existing channels. 

## Problem

IBC is designed so that a specific application module will claim the capability of a channel. Currently, once a channel is opened and the channel handshake is complete, you cannot change the application module claiming that channel. This means that if you wanted to upgrade an existing application module or add middleware to both ends of a channel, you would need to open a new channel with these new modules meaning all previous state in the prior channel would be lost. This is particularly important for channels using the ics-20 (fungible token transfer) application module because tokens are not fungible between channels.

## Objectives

To enable existing channels to upgrade the application module claiming the channel or add middleware to both ends of an existing channel whilst retaining the state of the channel. 

## Scope

A new `ChannelEnd` interface is defined after a channel upgrade, the scope of these upgrades is detailed in the table below.

| Features  | Release |
| --------- | ------- |
| Performing a channel upgrade results in an application module changing from v1 to v2, claiming the same `channelID` and `portID` | v1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` changing the ordering from a higher to lower degree of ordering | v1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` having additional middleware added to the application stack on both sides of the channel | v1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` modifying the `connectionHops` | v1 |

# User requirements

## Use cases

Upgrading an existing application module from v1 to v2, e.g. new features could be added to the existing ics-20 application module which would result in a new version of the module.

Adding middleware on both sides of an existing channel, e.g. relayer incentivisation middleware, ics-29, requires middleware to be added to both ends of a channel to incentivise the `recvPacket`, `acknowledgePacket` and `timeoutPacket`.

# Functional requirements

<!-- They should describe as completely as necessary the system's behaviors under various conditions. They describe what the engineers must implement to enable users to accomplish their tasks (user requirements), thereby satisfying the business requirements. Software engineers don't implement business requirements or user requirements. They implement functional requirements, specific bits of system behavior. Each requirement should be uniquely identified with a meaningful tag. -->

## Assumptions and dependencies

<!-- List any assumed factors that could affect the requirements. The project could be affected if these assumptions are incorrect, are not shared, or change. Also identify any dependencies the project has on external factors. -->

- Functional relayer infrastructure is required to perform a channel upgrade.
- Chains wishing to successfully upgrade a channel must be using a minimum ibc-go version in the v8 line.
- Chains proposing an upgrade must have the middleware or application module intended to be used in the channel upgrade configured. 

## Features

<!-- Use a table like the following for the requirements:
| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
-->
### 1 - Configuration

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 1.01 | The parameters for the permitted type of upgrade can be configured on completetion of a successful governance proposal or using the x/group module  | ------------ | Drafted | 
| 1.02 | A type of upgrade can be permitted for all channels with a specific `PortID` or for a subset of channels using this `PortID` | ------------ | Drafted |
| 1.03 | A chain may choose to permit all channel upgrades from counterparties by default   | ------------ | Drafted |  

### 2 - Initiation

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 2.01 | A channel upgrade can only be initiated before the specified timeout period for that type of upgrade| ------------ | Drafted |
| 2.02 | A chain can configure a channel upgrade to be initiated automatically after a successful governance proposal  | ------------ | Drafted |
| 2.03 | After permission is granted for a specific type of upgrade, any relayer can initiate the upgrade | ------------ |Drafted | 
| 2.04 | A channel upgrade can only be initiated when both `ChannelEnd`s are in the `OPEN` state | ------------ | Drafted | 


### 3 - Upgrade Handshake

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 3.01 | The upgrade proposing chain will go from channel state `OPEN` to `INITUPGRADE` after successful execution of the `ChanUpgradeInit` datagram | ------------ | Drafted |
| 3.02 | The upgrade proposing chain channel state will revert to `OPEN` from `INITUPGRADE` if `ChanUpgradeTry` is not successfully executed on the counterparty chain within a specfied timeframe | ------------ | Drafted | 
| 3.03 | If the counterparty chain accepts the upgrade its channel state will go from `OPEN` to `TRYUPGRADE` after successful execution of the `ChanUpgradeTry` datagram | ------------ | Drafted |
| 3.04 | The upgrade proposing chain will go from `INITUPGRADE` to `OPEN` after successful execution of the `ChanUpgradeAck` datagram | ------------ | Drafted |
| 3.05 | A relayer must initiate the `ChanUpgradeAck` datagram | ------------ | Drafted |
| 3.06 | The counterparty chain state will go from `TRYUPGRADE` to `OPEN` after successful execution of the `ChanUpgradeConfirm` datagram | ------------ | Drafted |
| 3.07 | A relayer must initiate the `ChanUpgradeConfirm` datagram | ------------ | Drafted |
| 3.08 | The counterparty chain may reject a proposed channel upgrade and the original channel will be restored| ------------ | Drafted |
| 3.09 | If an upgrade handshake is unsuccessful, the original channel will be restored| ------------ | Drafted |

# External interface requirements

<!-- They describe the interfaces to other software systems, hardware components, and users. Ideally they should state the purpose, format and content of messages used for input and output. -->
| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ |
| 4.01 | There should be a CLI command to query the channel upgrade sequence number  | ------------ | Drafted |

# Non-functional requirements

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ |
| 5.01 | A malicious actor should not be able to compromise the liveness of a channel | ------------ | Drafted|

<!-- Other-than-functional requirements that do not specify what the system does, but rather how well it does those things. For example: quality requirements: performance, security, portability, etc. -->