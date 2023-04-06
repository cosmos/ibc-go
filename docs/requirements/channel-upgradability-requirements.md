# Business requirements

Rather than create a new channel to expand upon the capabilities of an existing channel, channel upgradability enables new features and capabilities to be added to existing channels. 

## Problem

IBC is designed so that a specific application module will claim the capability of a channel. Currently, once a channel is opened and the channel handshake is complete, you cannot change the application module claiming that channel. This means that if you wanted to upgrade an existing application module or add middleware to both ends of a channel, you would need to open a new channel with these new modules meaning all previous state in the prior channel would be lost. This is particularly important for channels using the ICS 20 (fungible token transfer) application module because tokens are not fungible between channels.

## Objectives

To enable existing channels to upgrade the application module claiming the channel or add middleware to both ends of an existing channel whilst retaining the state of the channel. 

## Scope

A new `ChannelEnd` interface is defined after a channel upgrade, the scope of these upgrades is detailed in the table below.

| Features  | Release |
| --------- | ------- |
| Performing a channel upgrade results in an application module changing from v1 to v2, claiming the same `channelID` and `portID` | v1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` changing the channel ordering from a higher to lower degree of ordering | v1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` having additional middleware added to the application stack on both sides of the channel | v1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` modifying the `connectionHops` | v1 |

# User requirements

## Use cases

Upgrading an existing application module from v1 to v2, e.g. new features could be added to the existing ICS 20 application module which would result in a new version of the module.

Adding middleware on both sides of an existing channel, e.g. relayer incentivisation middleware, ICS 29, requires middleware to be added to both ends of a channel to incentivise the `recvPacket`, `acknowledgePacket` and `timeoutPacket`.

### Governance-gated upgrades on both sides

Governance-gated upgrades on chains A and B: chains A and B both gate the upgrade by a full chain governance or a DAO/technical committee via the groups module.

#### Preconditions

- The relayer can construct correct messages.
- There exists a transfer channel between chain A and chain B, with channel ID `channel-0` on both ends.

#### Postconditions

- If channel upgrade handshake succeeds, then the channel has upgraded to the new parameters.
- If the channel upgrade handshake fails, then the channel remains operational with the previous parameters.

Normal flow:

1. Governance proposal is submitted on chain A to upgrade port ID `transfer`, channel ID `channel-0`.
2. Governance proposal is submitted on chain B to allow upgrade for port ID `transfer`, channel ID `channel-0`.
3. Governance proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Governance proposal passes on chain B and upgrading port ID `transfer`, channel ID `channel-0` will be allowed.
5. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
7. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
8. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
9. Chain A disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance).
10. Relayer detects execution of `MsgChannelUpgradeAck` and submits `MsgChannelUpgradeConfirm` on chain A.
11. Execution of `MsgChannelUpgradeConfirms` succeeds on chain B.
12. Chain B disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance).

Exception flows:

##### Chain B has not approved upgrade yet

Steps 1 to 3 of normal flow remain the same.

4. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Execution of `MsgChannelUpgradeTry` fails on chain B since the upgrade is not approved.
6. Relayer submits `MsgChannelUpgradeTimeout` on chain A.
7. Execution of `MsgChannelUpgradeTimeout` on chain A succeeds and channel is restored to previous state.

Question: Should both chains A and B disallow the upgrade now that it failed because of the timeout? That would mean that new proposals need to pass to attempt the upgrade again.

##### Chain B submits proposal to initiate upgrade

This is a crossing hellos scenario. Step 1 of normal flow remains the same.

2. Governance proposal is submitted on chain B to upgrade port ID `transfer`, channel ID `channel-0`.
3. Governance proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Governance proposal passes on chain B and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Relayer detects execution of `MsgChannelUpgradeInit` on chain A and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Chain B is already on `INITUPGRADE` state, so this a crossing hello. Execution of `MsgChannelUpgradeTry` succeeds on chain B.

Steps 7 to 12 of normal flow remain the same.

##### Upgrade attempt on chain B is initiated after timeout

Step 1 to 2 of normal flow remain the same.

3. Governance proposal passes on chain B and upgrading port ID `transfer`, channel ID `channel-0` will be allowed.
4. Governance proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Execution of `MsgChannelUpgradeTry` fails on chain B because chain B have moved on after the specified timeout.
7. Relayer submits `MsgChannelUpgradeTimeout` on chain A.
8. Execution of `MsgChannelUpgradeTimeout` on chain A succeeds and channel is restored to previous state.

Question: Should both chains A and B disallow the upgrade now that it failed because of the timeout? That would mean that new proposals need to pass to attempt the upgrade again.

# Functional requirements

## Assumptions and dependencies

- Functional relayer infrastructure is required to perform a channel upgrade.
- Chains wishing to successfully upgrade a channel must be using a minimum ibc-go version in the v8 line.
- Chains proposing an upgrade must have the middleware or application module intended to be used in the channel upgrade configured. 

## Features

### 1 - Configuration

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 1.01 | The parameters for the permitted type of upgrade can be configured on completion of a successful governance proposal or using the `x/group` module  | ------------ | Drafted | 
| 1.02 | A type of upgrade can be permitted for all channels with a specific `portID` or for a subset of channels using this `portID` | ------------ | Drafted |
| 1.03 | A chain may choose to permit all channel upgrades from counterparties by default | ------------ | Drafted |  

### 2 - Initiation

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 2.01 | A channel upgrade can only be initiated before the specified timeout period for that type of upgrade | ------------ | Drafted |
| 2.02 | A chain can configure a channel upgrade to be initiated automatically after a successful governance proposal | ------------ | Drafted |
| 2.03 | After permission is granted for a specific type of upgrade, any relayer can initiate the upgrade | ------------ |Drafted | 
| 2.04 | A channel upgrade can only be initiated when both `ChannelEnd`s are in the `OPEN` state | ------------ | Drafted | 


### 3 - Upgrade Handshake

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 3.01 | The upgrade proposing chain will go from channel state `OPEN` to `INITUPGRADE` after successful execution of the `ChanUpgradeInit` datagram | ------------ | Drafted |
| 3.02 | The upgrade proposing chain channel state will revert to `OPEN` from `INITUPGRADE` if `ChanUpgradeTry` is not successfully executed on the counterparty chain within a specified timeframe | ------------ | Drafted | 
| 3.03 | If the counterparty chain accepts the upgrade its channel state will go from `OPEN` to `TRYUPGRADE` after successful execution of the `ChanUpgradeTry` datagram | ------------ | Drafted |
| 3.04 | The upgrade proposing chain will go from `INITUPGRADE` to `OPEN` after successful execution of the `ChanUpgradeAck` datagram | ------------ | Drafted |
| 3.05 | A relayer must initiate the `ChanUpgradeAck` datagram | ------------ | Drafted |
| 3.06 | The counterparty chain state will go from `TRYUPGRADE` to `OPEN` after successful execution of the `ChanUpgradeConfirm` datagram | ------------ | Drafted |
| 3.07 | A relayer must initiate the `ChanUpgradeConfirm` datagram | ------------ | Drafted |
| 3.08 | The counterparty chain may reject a proposed channel upgrade and the original channel will be restored | ------------ | Drafted |
| 3.09 | If an upgrade handshake is unsuccessful, the original channel will be restored | ------------ | Drafted |

# External interface requirements

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ |
| 4.01 | There should be a CLI command to query the channel upgrade sequence number | ------------ | Drafted |

# Non-functional requirements

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ |
| 5.01 | A malicious actor should not be able to compromise the liveness of a channel | ------------ | Drafted|
