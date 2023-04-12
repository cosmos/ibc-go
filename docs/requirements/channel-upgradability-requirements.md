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

A few sample use case flows are described below. For the sake of simplicity, the samples below make the following assumptions:

1. The relayer can construct correct messages.
2. There exists a transfer channel between chain A and chain B, with channel ID `channel-0` on both ends.
3. The channel is upgraded to use fee incentivization (i.e. stack fee middleware on top of the transfer application). Both chains' binary has wired up the fee middleware.

Further exception flows can be explored if some of the asumptions above do not apply. For example: if chain B's binary has not wired up the fee middleware and `MsgChannelUpgradeTry` is submitted, the upgrade will be aborted and a cancellation message can be submitted to chain A to restore the channel to its previous state.

### Pre-approved upgrades on both sides

Normal flow:

1. Governance proposal is submitted on chain A to allow upgrade for port ID `transfer`, channel ID `channel-0`.
2. Governance proposal is submitted on chain B to allow upgrade for port ID `transfer`, channel ID `channel-0`.
3. Governance proposal passes on chain A and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
4. Governance proposal passes on chain B and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
5. Relayer submits `MsgChannelUpgradeInit` on chain A proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Execution of `MsgChannelUpgradeInit` succeeds on chain A.
7. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
8. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
9. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
10. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
11. Chain A disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance).
12. Relayer detects execution of `MsgChannelUpgradeAck` and submits `MsgChannelUpgradeConfirm` on chain A.
13. Execution of `MsgChannelUpgradeConfirms` succeeds on chain B.
14. Chain B disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance).

Exception flows:

##### Chain B has not pre-approved upgrade yet

Pre-steps: 1 to 3 of normal flow.

4. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Execution of `MsgChannelUpgradeTry` fails on chain B since the upgrade is not approved.
6. Governance proposal passes on chain B and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
7. Relayer re-submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.

Post-steps: 8 to 14 of normal flow.

##### Chain B pre-approves upgrade after timeout

Pre-steps: 1 to 3 of normal flow.

4. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Execution of `MsgChannelUpgradeTry` fails on chain B since the upgrade is not approved.
6. Governance proposal passes on chain B and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
5. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Execution of `MsgChannelUpgradeTry` fails on chain B because chain B has moved on after the specified timeout.
7. Relayer submits `MsgChannelUpgradeTimeout` on chain A.
8. Execution of `MsgChannelUpgradeTimeout` on chain A succeeds and channel is restored to previous state.

### Governance-gated upgrade on one side / pre-approved upgrade on another side

Normal flow:

1. Governance proposal is submitted on chain A to upgrade port ID `transfer`, channel ID `channel-0`.
2. Governance proposal is submitted on chain B to allow upgrade for port ID `transfer`, channel ID `channel-0`.
3. Governance proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Governance proposal passes on chain B and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
5. Relayer detects execution of `MsgChannelUpgradeInit` on chain A and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
7. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
8. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
9. Chain A disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance).
10. Relayer detects execution of `MsgChannelUpgradeAck` and submits `MsgChannelUpgradeConfirm` on chain A.
11. Execution of `MsgChannelUpgradeConfirms` succeeds on chain B.
12. Chain B disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance).

### Governance-gated upgrades on both sides

Normal flow:

This is a crossing hellos scenario.

1. Governance proposal is submitted on chain A to upgrade port ID `transfer`, channel ID `channel-0`.
2. Governance proposal is submitted on chain B to upgrade port ID `transfer`, channel ID `channel-0`.
3. Governance proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Governance proposal passes on chain B and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Relayer detects execution of `MsgChannelUpgradeInit` on chain A and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Chain B is already on `INITUPGRADE` state, so this is a crossing hello. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
7. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
8. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
9. Chain A disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance).
10. Relayer detects execution of `MsgChannelUpgradeAck` and submits `MsgChannelUpgradeConfirm` on chain A.
11. Execution of `MsgChannelUpgradeConfirms` succeeds on chain B.
12. Chain B disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance).

Exception flows:

##### Upgrade handshake on chain B needs to be re-tried

Pre-steps: 1 of normal flow.

2. Governance proposal is submitted on chain B to upgrade port ID `transfer`, channel ID `channel-0`.
3. Governance proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Relayer detects execution of `MsgChannelUpgradeInit` on chain A and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Governance proposal on chain B has not passed yet (i.e. `MsgChannelUpgradeInit` has not executed yet) and upgrade has not been pre-aproved, therefore `MsgChannelUpgradeTry` on chain B fails. 
6. Governance proposal passes on chain B and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
7. Relayer re-submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
8. Chain B is now on `INITUPGRADE` state, so this a crossing hello. Execution of `MsgChannelUpgradeTry` succeeds on chain B.

Post-steps: 7 to 12 of normal flow.

##### Upgrade attempt on chain B is initiated after timeout

Pre-steps: 1 of normal flow.

2. Governance proposal is submitted on chain B to upgrade port ID `transfer`, channel ID `channel-0`.
3. Governance proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Execution of `MsgChannelUpgradeTry` fails on chain B because chain B has moved on after the specified timeout.
6. Relayer submits `MsgChannelUpgradeTimeout` on chain A.
7. Execution of `MsgChannelUpgradeTimeout` on chain A succeeds and channel is restored to previous state.

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
