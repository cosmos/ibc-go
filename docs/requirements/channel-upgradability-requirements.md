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

Sample use cases:

- Upgrading an existing application module from v1 to v2, e.g. new features could be added to the existing ICS 20 application module which would result in a new version of the module.
- Adding middleware on both sides of an existing channel, e.g. relayer incentivisation middleware, ICS 29, requires middleware to be added to both ends of a channel to incentivise the `recvPacket`, `acknowledgePacket` and `timeoutPacket`.

Upgrades can be initiated in two possible ways:

- __Permissionless__: upgrade initiation is approved by governance or group proposal and a relayer can submit `MsgChannelUpgradeInit`.
- __Permissioned__: upgrade initiation is approved by governance or group proposal and `MsgChannelUpgradeInit` is automatically submitted on proposal passing.

The upgrade approval process can be represented in terms of the following parameters:

- The set of port ID, channel ID pairs that specify the channels that can be upgraded. It can be possible to specifit a single channel or a set of channels satisfiying certain port ID, channel ID conditions. Examples: upgrading can be approved for __a)__ channels with port ID, channel ID combinations (`transfer`, `channel-0`) & (`transfer`, `channel-1`), __b)__ all channels for port ID `transfer`, __c)__ all channels with port ID starting with `icacontroller` (so that all ICA channels on a controller chain, regardless of the owner address appended to the port ID, can be upgraded).
- The parameters that are accepted for the upgrade. Currently they are channel version, channel ordering and connection hops. These parameters should be specified per specific pairs of port ID, channel ID (e.g. for channel (`transfer`, `channel-0`)) or for all channels using a certain port ID (e.g. for all channels on port ID `transfer`). The reason is that it could lead to errors specifying the upgrade parameters for channels on two different ports (e.g. for channels on port ID `transfer` and channels with port ID starting with `icacontroller` it would not make sense to upgrade to verions `ics20-2`, since this version only make sense for transfer channels, but not for interchain accounts channels). If an upgradable parameter is omitted, then we shall assume that the parameter must stay the same after the upgrade.
- The timeout (either as block height or timestamp) until when upgrades can be initiated after approval. This can also be specified per specific pairs of port ID, channel ID (e.g. for channel (`transfer`, `channel-0`)) or for all channels using a certain port ID (e.g. for all channels on port ID `transfer`).

Some concrete examples to illustrate the above:

- Approve upgrading channel (`transfer`, `channel-0`) to version `ics20-2` until block height 1000.
- Approve upgrading channels (`transfer`, `channel-0`) & (`transfer`, `channel-1`) to version `ics20-2` until block height 1000 AND channel (`icacontroller-cosmos13fdx64cc9afrsnjdytk6drnj3mzdku5qsemlz5`, `channel-2`) to ordering `UNORDERED` until time 1682288559.
- Approve upgrading all channels on port ID `transfer` to version `{"fee_version":"ics29-1","app_version":"ics20-1"}` until block height 1050.
- Approve upgrading all channels with port ID starting with `icacontroller` to version `{"fee_version":"ics29-1","app_version":"ics27-1"}` until block height 1080.

Once an approved upgrade succeeds, it is required that a new governance or group proposal passes to approve a new upgrade to the channel.

Assuming that we have two chains, and that both chains approve a compatible upgrade (i.e. the proposed upgrade parameters are identical on both chains and a channel between them is in the upgrade scope), then the following upgrade initiation combinations are possible:

- Permissionless upgrade initiation on both chain A and B.
     In this scenario both chains approve a channel upgrade and a relayer can submit `MsgChannelUpgradeInit` on one side to start the handshake witin the timeout period specified in the upgrade approval proposal. Upon successful execution of `MsgChannelUPgradeInit` a relayer can submit `MsgChannelUpgradeTry` on the counterparty.
- Permissionless upgrade on chain A and permissioned upgrade on chain B.
- Permissioned upgrade on chain A and permissionless upgrade on B.
    In these 2 scenarios, both side approve a channel upgrade, but one chain automatically initiates the upgrade process by executing `MsgChannelUpgradeInit` and the other side needs a relayer to submit either `MsgChannelUpgradeInit` or `MsgChannelUpgradeTry`. If `MsgChannelUpgradeInit` is executed on both chains, then we would have a crossing hello uld be that a crossing hello situation occurs if relayer(s) submit `MsgChannelUpgradeInit` on both chains.
- Permissioned upgrade on both chain A and B.
    The upgrade is approved and automatically initiated on both chains with the submission of `MsgChannelUpgradeInit`. This is a crossing hello scenario.

A few sample use case flows are described below. For the sake of simplicity, the samples below make the following assumptions:

1. The relayer can construct correct messages.
2. There exists a transfer channel between chain A and chain B, with channel ID `channel-0` on both ends.
3. The channel is upgraded to use fee incentivization (i.e. stack fee middleware on top of the transfer application). Both chains' binary has wired up the fee middleware.

Further exception flows can be explored if some of the assumptions above do not apply. For example: if chain B's binary has not wired up the fee middleware and `MsgChannelUpgradeTry` is submitted, the upgrade will be aborted and a cancellation message can be submitted to chain A to restore the channel to its previous state. Or if each chain approves a different channel version, then the upgrade will abort during the negotiation.

### Permissionless upgrade on chain A and B

Normal flow:

1. Proposal is submitted on chain A to allow upgrade for port ID `transfer`, channel ID `channel-0`.
2. proposal is submitted on chain B to allow upgrade for port ID `transfer`, channel ID `channel-0`.
3. Proposal passes on chain A and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
4. Proposal passes on chain B and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
5. Relayer submits `MsgChannelUpgradeInit` on chain A proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Execution of `MsgChannelUpgradeInit` succeeds on chain A.
7. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
8. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
9. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
10. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
11. Chain A disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance or group proposal).
12. Relayer detects execution of `MsgChannelUpgradeAck` and submits `MsgChannelUpgradeConfirm` on chain B.
13. Execution of `MsgChannelUpgradeConfirm` succeeds on chain B.
14. Chain B disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance or group proposal).

Exception flows:

- Proposal passes on chain A but `MsgChannelUpgradeInit` is submitted by the relayer after the timeout for initiation has passed. Then upgrade is not possible anymore and a new proposal needs to be submitted and accepted first before a new upgrade is attempted.
- Proposal passes on chain A before it passes on chain B and `MsgChannelUpgradeTry` is rejected on chain B. Then the relayer needs to re-submit `MsgChannelUpgradeTry` after the proposal passes on chain B (assuming that it does before the counterparty upgrade timeout specified on chain A in `MsgChannelUpgradeInit`).
- Proposal on chain B passes after the counterparty upgrade timeout specified on chain A, then upgrade will not go through and state on chain A will be rolled back to `OPEN`. If the upgrade initiation timeout has not passed yet on chain A, then a relayer can submit a new `MsgChannelUpgradeInit` with a new counterparty upgrade timeout.

### Permissionless upgrade on chain A and permissioned upgrade on chain B

Normal flow:

1. Proposal is submitted on chain A to allow upgrade for port ID `transfer`, channel ID `channel-0`.
2. Proposal is submitted on chain B to allow upgrade and initiate upgrade for port ID `transfer`, channel ID `channel-0`.
3. Proposal passes on chain A and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
4. Proposal passes on chain B and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain A proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Execution of `MsgChannelUpgradeTry` succeeds on chain A.
7. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain B.
8. Execution of `MsgChannelUpgradeAck` succeeds on chain B.
9. Chain B disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance or group proposal).
10. Relayer detects execution of `MsgChannelUpgradeAck` and submits `MsgChannelUpgradeConfirm` on chain A.
11. Execution of `MsgChannelUpgradeConfirm` succeeds on chain A.
12. Chain A disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance or group proposal).

Exception flows:

- Proposal passes on chain B after a relayer has submitted `MsgChannelUpgradeInit` on chain A. This is a crossing hello scenario, since chain B has also automatically executed `MsgChannelUpgradeInit`. Relayer can submit `MsgChannelUpgradeTry` on chain B to continue the handshake.
- Proposal passes on chain B after the initiation upgrade on chain A has passed, then when a relayer attempts to submit `MsgChannelUpgradeTry` on chain A, the message will fail, and the upgrade will be aborted and the state of the channel on chain B will be reverted to `OPEN`.
- Proposal on chain A passes after the counterparty upgrade timeout specified on chain B, then upgrade will not go through and state on chain B will be rolled back to `OPEN`. If the upgrade initiation timeout has not passed yet on chain A, then a relayer can submit a new `MsgChannelUpgradeInit` on chain B with a new counterparty upgrade timeout.

### Permissioned upgrade on chain A and permissionless upgrade on chain B

Normal flow:

1. Proposal is submitted on chain A to allow upgrade and initiate upgrade for port ID `transfer`, channel ID `channel-0`.
2. Proposal is submitted on chain B to allow upgrade for port ID `transfer`, channel ID `channel-0`.
3. Proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Proposal passes on chain B and it allows port ID `transfer`, channel ID `channel-0` to be upgraded.
5. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
6. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
7. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
8. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
9. Chain A disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance or group proposal).
10. Relayer detects execution of `MsgChannelUpgradeAck` and submits `MsgChannelUpgradeConfirm` on chain B.
11. Execution of `MsgChannelUpgradeConfirm` succeeds on chain B.
12. Chain B disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance or group proposal).

Exception flows:

- Proposal passes on chain B after a relayer has submitted `MsgChannelUpgradeInit` on chain A. This is a crossing hello scenario, since chain B has also automatically executed `MsgChannelUpgradeInit`. Relayer can submit `MsgChannelUpgradeTry` on chain B to continue the handshake.
- Proposal passes on chain B after the initiation upgrade on chain A has passed, then when a relayer attempts to submit `MsgChannelUpgradeTry` on chain A, then the message will fail, the upgrade will be aborted and the state of the channel on chain B will be reverted to `OPEN`.
- Proposal on chain A passes after the counterparty upgrade timeout specified on chain B, then upgrade will not go through and state on chain B will be rolled back to `OPEN`. If the upgrade initiation timeout has not passed yet on chain B, then a relayer can submit a new `MsgChannelUpgradeInit` on chain A with a new counterparty upgrade timeout.

### Permissioned upgrade on chain A and B

Normal flow:

1. Proposal is submitted on chain A to allow upgrade and initiate upgrade for port ID `transfer`, channel ID `channel-0`.
2. Proposal is submitted on chain B to allow upgrade and initiate upgrade for port ID `transfer`, channel ID `channel-0`.
3. Proposal passes on chain A and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Proposal passes on chain B and:
    a. it allows port ID `transfer`, channel ID `channel-0` to be upgraded;
    b. it executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Relayer detects execution of `MsgChannelUpgradeInit` on chain A and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`. Since chain B has already executed `MsgChannelUpgradeInit`, the we are in a crossing hello scenario.
6. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
7. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
8. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
9. Chain A disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance or group proposal).
10. Relayer detects execution of `MsgChannelUpgradeAck` and submits `MsgChannelUpgradeConfirm` on chain B.
11. Execution of `MsgChannelUpgradeConfirm` succeeds on chain B.
12. Chain B disallows port ID `transfer`, channel ID `channel-0` to be upgraded (now that upgrade has completed, then we can disallow future upgrades until approved again by governance or group proposal).

Exception flows:

- If two different relayers detect execution of `MsgChannelUpgradeInit` on both chain A and chain B and they submit `MsgChannelUpgradeTry` on the counterparty, then the handshake will finish when both chains execute `MsgChannelUpgradeAck`.
- Proposal on either chain A or chain B passes after the counterparty upgrade timeout specified on `MsgChannelUpgradeInit` has passed. Then `MsgChannelUpgradeTry` will fail on the counterparty and the upgrade can be timed out on the initiating chain and the state of the channel rolled back to `OPEN`. If the upgrade initiation timeout has not passed yet, then it would be possible for a relayer to submit again `MsgChannelUpgradeInit` with a new counterparty upgrade timeout.

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
