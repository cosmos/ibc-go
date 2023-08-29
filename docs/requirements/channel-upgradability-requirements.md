# Business requirements

Rather than create a new channel to expand upon the capabilities of an existing channel, channel upgradability enables new features and capabilities to be added to existing channels.

## Problem

Currently, once a channel is opened and the channel handshake is complete, you cannot change or renogociate the semantics of that channel. This means that if you wanted to make a change to a channel affecting the semantics on both channel ends, you would need to open a new channel meaning all previous state in the prior channel would be lost. This is particularly important for channels using the ICS 20 (fungible token transfer) application module because tokens are not fungible between channels.

Upgrading a channel enables upgrading the application module claiming the channel, where the upgrade requires a new packet data structure or adding a middleware at both channel ends. Currently it is possible to make changes to one end of a channel that does not require the counterparty to make changes, for example adding rate limiting middleware. Channel upgradability is solving the problem when changes need to be agreed upon on both sides of the channel.

## Objectives

To enable existing channels to upgrade the application module claiming the channel or add middleware to both ends of an existing channel whilst retaining the state of the channel.

## Scope

A new `ChannelEnd` interface is defined after a channel upgrade, the scope of these upgrades is detailed in the table below.

| Features  | Release |
| --------- | ------- |
| Performing a channel upgrade results in an application module changing from v1 to v2, claiming the same `channelID` and `portID` | v9 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` changing the channel ordering | v9 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` having a new channel version, where the change is needed on both `ChannelEnd`s, for example additional middleware added to the application stack on both sides of the channel, or a change to the packet data or acknowledgement structure | v9 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` modifying the `connectionHops` | v9 |

# User requirements

## Use cases

- Upgrading an existing application module from v1 to v2, e.g. new features could be added to the existing ICS 20 application module which would result in a new version of the module, e.g. enabling different coin types to be sent in a single packet.
- Adding middleware on both sides of an existing channel, e.g. relayer incentivisation middleware, ICS 29, requires middleware to be added to both ends of a channel to incentivise the `recvPacket`, `acknowledgePacket` and `timeoutPacket`.

### Upgrade parameters

The upgrade approval process can be represented in terms of the following parameters:

- The set of port ID, channel ID pairs that specify the channels that can be upgraded. It can be possible to specify a single channel or a set of channels satisfying certain port ID, channel ID conditions. Examples: upgrading can be approved for
  - channels with port ID, channel ID combinations (`transfer`, `channel-0`) & (`transfer`, `channel-1`),
  - all channels for  a specified port ID e.g. `transfer`, or  `icacontroller` (so that all ICA channels on a controller chain, regardless of the owner address appended to the port ID, can be upgraded).
- The upgrade parameters - these must be specified for each channel ID and port ID selected for the upgrade. The upgrade parameters are:
  - channel version
  - channel ordering
  - connection hops
    - If an upgradable parameter is omitted, then it is assumed the parameter must stay the same after the upgrade.
- The timeout until when upgrades can be initiated after approval. This can be specified as:
  - Block height
  - Timestamp  

The upgrade parameters and timestamp must be specified per specific pairs of port ID, channel ID (e.g. for channel (`transfer`, `channel-0`)) or for all channels using a certain port ID (e.g. for all channels on port ID `transfer`). An approval timeout can be left unspecified.

#### Examples

| port ID, channel ID | version | ordering | hops | timeout| use case|
| -- |  --- | ------ | ------ | ------ |------ |
| (`transfer`, `channel-0`) | `ics20-2` |  |  | block height 1000 |Upgrading ics20 to v2 for 1 channel |
| (`icacontroller-cosmos13fx6...`, `channel-2`)|  | `UNORDERED` |  | timestamp 1682288559 | Upgrading single ICA controller channel to be unordered |
| (`transfer`, `*`)| `{"fee_version":"ics29-1", "app_version":"ics20-1"}`|  | | block height 1050 | Upgrade all transfer channels to have ics-29 enabled |
| (`icacontroller-*`, `*`) | `{"fee_version":"ics29-1", "app_version":"ics27-1"}` |  |  | block height 1080 | Upgrade all ICA controller channels to have ics-29 enabled |

Once an approved upgrade succeeds or the timeout window has passed, it is required that a new governance or groups proposal passes to approve a new upgrade to the channel.

### Upgrade Flow

Upgrades can be initiated in two possible ways:

- __Permissionless__: upgrade initiation is previously approved by governance or a groups proposal and a relayer can submit `MsgChannelUpgradeInit`.
- __Permissioned__: upgrade initiation is approved by governance or groups proposal and `MsgChannelUpgradeInit` is automatically submitted on proposal passing.

Assuming that we have two chains, and that both chains approve a compatible upgrade (i.e. the proposed upgrade parameters are identical on both chains and a channel between them is in the upgrade scope), then the following upgrade initiation combinations are possible:

- Permissionless upgrade initiation on both chain A and B.
     In this scenario both chains approve a channel upgrade and a relayer can submit `MsgChannelUpgradeInit` on one side to start the handshake within the timeout period specified in the upgrade approval proposal. Upon successful execution of `MsgChannelUPgradeInit` a relayer can submit `MsgChannelUpgradeTry` on the counterparty.
- Permissionless upgrade on chain A and permissioned upgrade on chain B (or vice versa).
  - Both sides approve a channel upgrade, but one chain automatically initiates the upgrade process by executing `MsgChannelUpgradeInit` and the other side needs a relayer to submit either `MsgChannelUpgradeInit` or `MsgChannelUpgradeTry`. If `MsgChannelUpgradeInit` is executed on both chains, then we would have a crossing hello situation.
- Permissioned upgrade on both chain A and B.
  - The upgrade is approved and automatically initiated on both chains with the submission of `MsgChannelUpgradeInit`. This is a crossing hello scenario.
  - If the upgrade is approved on chain A before being approved on chain B, the initiation of the upgrade would fail. The initiation from the approval on chain B would suceed if it is executed before the approval timeout window on chain A has passed.

A normal flow and crossing hello flow are described below. For the sake of simplicity, the samples below make the following assumptions:

1. The relayer can construct correct messages.
2. There exists a transfer channel between chain A and chain B, with channel ID `channel-0` on both ends.
3. The channel is upgraded to use fee incentivization (i.e. stack fee middleware on top of the transfer application). Both chains' binary has wired up the fee middleware.

Further exception flows can be explored if some of the assumptions above do not apply. For example: if chain B's binary has not wired up the fee middleware and `MsgChannelUpgradeTry` is submitted, the upgrade will be aborted and a cancellation message can be submitted to chain A to restore the channel to its previous state. Or if each chain approves a different channel version, then the upgrade will abort during the negotiation.

### Normal flow

A normal flow for upgrades, with both chains having granted approval for the upgrade:

1. `MsgChannelUpgradeInit` is submitted on chain A proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
2. Execution of `MsgChannelUpgradeInit` succeeds on chain A.
3. Relayer detects execution of `MsgChannelUpgradeInit` and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
4. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
5. Relayer starts flushing in-flight packets from chain A to chain B.
6. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
7. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
8. Chain A disallows port ID `transfer`, channel ID `channel-0` to be further upgraded.
9. Relayer starts flushing in-flight packets from chain B to chain A.
10. Relayer completes packet lifecycle for in-flight packets from chain A to chain B.
11. Relayer completes packet lifecycle for in-flight packets from chain B to chain A.
12. Relayer submits `MsgChannelUpgradeOpen` on chain B.
13. Execution of `MsgChannelUpgradeOpen` succeeds on chain B.
14. Chain B disallows port ID `transfer`, channel ID `channel-0` to be further upgraded.
15. Relayer submits `MsgChannelUpgradeOpen` on chain A.
16. Execution of `MsgChannelUpgradeOpen` succeeds on chain A.

A normal flow is expected when:

- There is permisionless initiation for chain A and B, and a relayer submits `MsgChannelUpgradeInit` on chain A.
- There is permissioned initiation on one chain (chain A in flow example), permissionless on the other chain (chain B in flow example). The proposal passes on chain A and executes `MsgChannelUpgradeInit`.
- There is permissioned initiation on both chains but the proposal passing on chain B initiates the flow before chain A has approved the upgrade, the upgrade initially fails but the proposal subsequently passes on chain A, executing `MsgChannelUpgradeInit` within the timeout window of chain B's approval.

Exception flows:

- `MsgChannelUpgradeInit` is submitted by the relayer on chain A after the timeout for initiation has passed. The upgrade is not possible anymore and a new proposal needs to be submitted and accepted first before a new upgrade is attempted.
- Proposal passes on chain A before it passes on chain B and `MsgChannelUpgradeTry` is rejected on chain B. Then the relayer needs to re-submit `MsgChannelUpgradeTry` after the proposal passes on chain B (assuming that it does before the counterparty upgrade timeout specified on chain A in `MsgChannelUpgradeInit`).
- Proposal on chain B passes after the counterparty upgrade timeout specified on chain A, then upgrade will not go through and state on chain A will be rolled back to `OPEN`. If the upgrade initiation timeout has not passed yet on chain A, then a relayer can submit a new `MsgChannelUpgradeInit` with a new counterparty upgrade timeout.

### Crossing hello flow

The crossing hello flow for upgrades, happens where approval is granted on both chains but the `MsgChannelUpgradeInit` is submitted at a similar time on both chains:

1. Chain A executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
2. Chain B executes `MsgChannelUpgradeInit` to upgrade channel version from `ics20-1` to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
3. Relayer detects execution of `MsgChannelUpgradeInit` on chain A and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`. Since chain B has already executed `MsgChannelUpgradeInit`, we are in a crossing hello scenario.
4. Execution of `MsgChannelUpgradeTry` succeeds on chain B.
5. Relayer starts flushing in-flight packets from chain A to chain B.
6. Relayer detects execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
7. Execution of `MsgChannelUpgradeAck` succeeds on chain A.
8. Chain A disallows port ID `transfer`, channel ID `channel-0` to be further upgraded.
9. Relayer starts flushing in-flight packets from chain B to chain A.
10. Relayer completes packet lifecycle for in-flight packets from chain A to chain B.
11. Relayer completes packet lifecycle for in-flight packets from chain B to chain A.
12. Relayer submits `MsgChannelUpgradeOpen` on chain B.
13. Execution of `MsgChannelUpgradeOpen` succeeds on chain B.
14. Chain B disallows port ID `transfer`, channel ID `channel-0` to be further upgraded.
15. Relayer submits `MsgChannelUpgradeOpen` on chain A.
16. Execution of `MsgChannelUpgradeOpen` succeeds on chain A.

The crossing hello flow is expected when:

- There is permissioned initiation on chain A and chain B that starts at a similar time.
- There is permissionless initiation on chain A and chain B but relayers submit `MsgChannelUpgradeInit` on both chains at a similar time.
- There is a combination of permissioned and permisionless initiation on both chains but the `MsgChannelUpgradeInit` is submitted on both chains at a similar time by a relayer and execution after a successful proposal

Exception flows:

- If two different relayers detect execution of `MsgChannelUpgradeInit` on both chain A and chain B and they submit `MsgChannelUpgradeTry` on the counterparty, then the handshake will finish when both chains execute `MsgChannelUpgradeAck` and all in-flight packets has been flushed.
- Proposal on either chain A or chain B passes after the counterparty upgrade timeout specified on `MsgChannelUpgradeInit` has passed. Then `MsgChannelUpgradeTry` will fail on the counterparty and the upgrade can be timed out on the initiating chain and the state of the channel rolled back to `OPEN`. If the upgrade initiation timeout has not passed yet, then it would be possible for a relayer to submit again `MsgChannelUpgradeInit` with a new counterparty upgrade timeout.

# Functional requirements

## Assumptions and dependencies

- Functional relayer infrastructure is required to perform a channel upgrade.
- Chains wishing to successfully upgrade a channel must be using a minimum ibc-go version in the v9 line.
- Chains proposing an upgrade must have the middleware or application module intended to be used in the channel upgrade configured.

## Features

### 1 - Configuration

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 1.01 | The parameters for the permitted type of upgrade can be configured on completion of a successful governance proposal or using the `x/group` module  | TBD | `Drafted` |
| 1.02 | A type of upgrade can be permitted for all channels with a specific `portID` or for a subset of channels using this `portID` | TBD | `Drafted` |
| 1.03 | A chain may choose to permit all channel upgrades of a specific type from counterparties by default | TBD | `Drafted` |
| 1.04 | A chain can specify the channel version, channel ordering and connection hops to be modified in an upgrade | TBD | `Drafted` |
| 1.05 | A chain can specify the timeout period, as a block height or timestamp, for a specific upgrade to be executed by | TBD | `Drafted` |
| 1.06 | A chain may choose to not specify a timeout period for a specific upgrade to be executed by | TBD | `Drafted` |  

### 2 - Initiation

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 2.01 | A channel upgrade can only be initiated before the specified timeout period for that type of upgrade | TBD | `Drafted` |
| 2.02 | A chain can configure a channel upgrade to be initiated automatically after a successful governance proposal | TBD | `Drafted` |
| 2.03 | After permission is granted for a specific type of upgrade, any relayer can initiate the upgrade | TBD |`Drafted` |
| 2.04 | A channel upgrade will be initiated when both `ChannelEnd`s are in the `OPEN` state | TBD | `Drafted` |
| 2.05 | A channel upgrade can be initiated when the counterparty has also executed the `ChanUpgradeInit` datagram with compatible parameters in the case of a crossing hello, when both channel ends are permissioning the upgrade | TBD | `Drafted` |

### 3 - Upgrade Handshake

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 3.01 | The upgrade proposing chain channel state will remain `OPEN` after successful or unsuccessful execution of the `ChanUpgradeInit` datagram | TBD | `Drafted` |
| 3.02 | If the counterparty chain accepts the upgrade its channel state will go from `OPEN` to `FLUSHING` after successful execution of the `ChanUpgradeTry` datagram, initiated by a relayer | TBD | `Drafted` |
| 3.03 | A relayer must initiate the `ChanUpgradeAck` datagram on the upgrade proposing chain, on successful execution the channel state will go from `OPEN` to `FLUSHING`| TBD | `Drafted` |
| 3.04 | Once in-flight packets have been flushed, the channel state shall change from `FLUSHING` to `FLUSHCOMPLETE` | TBD | `Drafted` |
| 3.05 | A relayer must initiate the `ChanUpgradeConfirm` datagram on the counterparty to inform of the timeout period of the counterparty | TBD | `Drafted` |
| 3.06 | Successful execution of the `ChanUpgradeConfirm` datagram when the channel state is `FLUSHCOMPLETE` changes the channel state to `OPEN` | TBD | `Drafted` |
| 3.07 | If the channel state is `FLUSHING` when the `ChanUpgradeConfirm` datagram is called, `ChanUpgradeOpen` is later called to change the state to `OPEN` | TBD | `Drafted` |
| 3.08 | When both channel ends are in the `FLUSHCOMPLETE` state, a relayer can submit the `ChanUpgradeOpen` datagram to move the channel state to `OPEN` | TBD | `Drafted` |
| 3.09 | The counterparty chain may reject a proposed channel upgrade and the original channel will be restored | TBD | `Drafted` |
| 3.10 | If an upgrade handshake is unsuccessful, the original channel will be restored | TBD | `Drafted` |
| 3.11 | A relayer can submit the `ChanUpgradeCancel` to cancel an upgrade which will successfully execute if the counterparty wrote an `ErrorReciept`| TBD | `Drafted` |

# External interface requirements

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 4.01 | There should be a CLI command to query the channel upgrade sequence number | TBD | `Drafted` |

# Non-functional requirements

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 5.01 | A malicious actor should not be able to compromise the liveness of a channel | TBD | `Drafted` |
