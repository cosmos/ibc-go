# Business requirements

Rather than create a new channel to expand upon the capabilities of an existing channel, channel upgradability enables new features and capabilities to be added to existing channels.

## Problem

Currently, once a channel is opened and the channel handshake is complete, you cannot change or renegotiate the semantics of that channel. This means that if you wanted to make a change to a channel affecting the semantics on both channel ends, you would need to open a new channel, meaning all previous state in the prior channel would be lost. This is particularly important for channels using the ICS 20 (fungible token transfer) application module because tokens are not fungible between channels.

Upgrading a channel enables upgrading the application module claiming the channel, where the upgrade requires a new packet data structure or adding a middleware at both channel ends. Currently it is possible to make changes to one end of a channel that does not require the counterparty to make changes, for example adding rate limiting middleware. Channel upgradability is solving the problem when changes need to be agreed upon on both sides of the channel.

## Objectives

To enable existing channels to upgrade the application module claiming the channel or add middleware to both ends of an existing channel whilst retaining the state of the channel.

## Scope

A new `ChannelEnd` interface is defined after a channel upgrade, the scope of these upgrades is detailed in the table below.

| Features  | Release |
| --------- | ------- |
| Performing a channel upgrade results in an application module changing from v1 to v2, claiming the same `channelID` and `portID` | v8.1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` changing the channel ordering | v8.1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` having a new channel version, where the change is needed on both `ChannelEnd`s, for example additional middleware added to the application stack on both sides of the channel, or a change to the packet data or acknowledgement structure | v8.1 |
| Performing a channel upgrade results in a channel with the same `channelID` and `portID` modifying the `connectionHops` | v8.1 |

# User requirements

## Use cases

- Upgrading an existing application module from v1 to v2, e.g. new features could be added to the existing ICS 20 application module which would result in a new version of the module, e.g. enabling multiple coin types to be sent in a single packet.
- Adding middleware on both sides of an existing channel, e.g. relayer incentivisation middleware, ICS 29, requires middleware to be added to both ends of a channel to incentivise the `recvPacket`, `acknowledgePacket` and `timeoutPacket`.

### Upgrade parameters

The parameters of a channel that can be modified during a channel upgrade are:

- channel version
- channel ordering
- connection hops

These must be specified for each channel ID and port ID selected for the upgrade. If an upgradable parameter is omitted, then it is assumed the parameter must stay the same after the upgrade.

#### Examples

| port ID, channel ID | version | ordering | hops | use case |
| ------------------- | ------- | -------- | ---- | -------- |
| (`transfer`, `channel-0`) & (`transfer`, `channel-1`)| `ics20-2` |  |  | Upgrading ics20 to v2 for 2 channels |
| (`icacontroller-cosmos13fx6...`, `channel-2`)|  | `UNORDERED` |  | Upgrading single ICA controller channel to be unordered |
| (`transfer`, `channel-1`) | `{"fee_version":"ics29-1", "app_version":"ics20-1"}`|  | | Upgrade single transfer channel to have ics-29 enabled |
| (`icacontroller-cosmos89y4...`, `channel-7`) | `{"fee_version":"ics29-1", "app_version":"ics27-1"}` |  |  | Upgrade an ICA controller channel to have ics-29 enabled |

### Upgrade Flow

Upgrades can be proposed only in a **permissioned** manner: upgrade initiation is approved by governance or groups proposal and `MsgChannelUpgradeInit` is automatically submitted on proposal passing. A permissioned mechanism may also optionally be implemented for the counterparty chain such that governance would need to allow (or prohibit, in the case of using a deny list) upgrades for channels in specific connections. Assuming that the proposed upgrade parameters are identical on both chains, then the following upgrade initiation combinations are possible:

- Chain A's governance proposal to initiate the upgrade of (`transfer`, `channel-0`) is accepted and `MsgChannelUpgradeInit` is executed. Chain B's governance proposal to allow upgrades for channels in connection `connection-0` is accepted. When the on-chain parameter with the list of allowed connections is updated, then a relayer can submit `MsgChannelUpgradeTry` on chain B for the counterparty channel (`transfer`, `channel-0`) in `connection-0`.
- Governance of both chain A and chain B submit proposal to initiate the upgrade of (`transfer`, `channel-0`) on chain A and the counterparty channel (`transfer`, `channel-0`) on chain B. The proposal passes and `MsgChannelUpgradeInit` is executed on both chains. This is a crossing hello scenario. A relayer can then continue the handshake submitting `MsgChannelUpgradeTry` on either chain A or chain B.

A normal flow and crossing hello flow are described in more detail below. For the sake of simplicity, the samples below make the following assumptions:

1. The relayer can construct correct messages.
2. There exists a transfer channel between chain A and chain B, with channel ID `channel-0` on both ends.
3. The channel is upgraded to use fee incentivization (i.e. stack fee middleware on top of the transfer application). Both chains' binary has wired up the fee middleware.

Further exception flows can be explored if some of the assumptions above do not apply. For example: if chain B's binary has not wired up the fee middleware and `MsgChannelUpgradeTry` is submitted, the upgrade will be aborted and a cancellation message can be submitted to chain A to restore the channel to its previous state. Or if each chain approves a different channel version, then the upgrade will abort during the negotiation.

### Normal flow

A normal flow for upgrades:

1. Governance proposal with `MsgChannelUpgradeInit` proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}` is submitted on chain A.
2. Governance proposal passes and `MsgChannelUpgradeInit` executes successfully on chain A.
3. Governance of chain B updates the list of allowed connections for channel upgrades, so that `channel-0` is allowed to be upgraded permissionlessly.
4. Relayer submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`.
5. Execution of `MsgChannelUpgradeTry` succeeds on chain B. Chain B specifies a timeout for chain A before which all packets on its side should be flushed.
6. Relayer starts flushing in-flight packets from chain B to chain A.
7. Relayer detects successful execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
8. Execution of `MsgChannelUpgradeAck` succeeds on chain A. Chain A specifies a timeout for chain B before which all packets on its side should be flushed.
9. Relayer starts flushing in-flight packets from chain A to chain B.
10. Relayer completes packet lifecycle for in-flight packets from chain B to chain A.
11. Relayer completes packet lifecycle for in-flight packets from chain A to chain B.
12. Relayer submits `MsgChannelUpgradeConfirm` on chain B.
13. Execution of `MsgChannelUpgradeConfirm` succeeds on chain B.
14. Relayer submits `MsgChannelUpgradeOpen` on chain A.
15. Execution of `MsgChannelUpgradeOpen` succeeds on chain A.

Sample exception flows:

- Relayer submits `MsgChannelUpgradeTry` for a channel whose connection is not in the list of allowed connection. Then the message is rejected, but the channel state on the proposing chain does not need to be reverted.
- Relayer submits `MsgChannelUpgradeTry` with different upgrade parameters to the parameters accepted on the proposing chain. Then the message is rejected, but the channel state on the proposing chain does not need to be reverted.
- During packet flushing if the timeout specified by the counterparty is reached, then the channel will be restored to its initial state, an error receipt will be written and the upgrade will be aborted.

### Crossing hello flow

The crossing hello flow for upgrades happens when a governance proposal on both chains passes and executes `MsgChannelUpgradeInit` with the same upgrade parameters for channel ends that are the counterparty of each other:

1. Governance proposal with `MsgChannelUpgradeInit` proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}` is submitted on chain A.
2. Governance proposal passes and `MsgChannelUpgradeInit` executes successfully on chain A.
3. Governance proposal with `MsgChannelUpgradeInit` proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}` is submitted on chain B.
4. Governance proposal passes and `MsgChannelUpgradeInit` executes successfully on chain B.
5. Relayer detects execution of `MsgChannelUpgradeInit` on chain A and submits `MsgChannelUpgradeTry` on chain B proposing to upgrade channel version to `{"fee_version":"ics29-1","app_version":"ics20-1"}`. Since chain B has already executed `MsgChannelUpgradeInit`, we are in a crossing hello scenario.
6. Execution of `MsgChannelUpgradeTry` succeeds on chain B. Chain B specifies a timeout for chain A before which all packets on its side should be flushed.
7. Relayer starts flushing in-flight packets from chain B to chain A.
8. Relayer detects successful execution of `MsgChannelUpgradeTry` and submits `MsgChannelUpgradeAck` on chain A.
9. Execution of `MsgChannelUpgradeAck` succeeds on chain A. Chain A specifies a timeout for chain B before which all packets on its side should be flushed.
10. Relayer starts flushing in-flight packets from chain A to chain B.
11. Relayer completes packet lifecycle for in-flight packets from chain B to chain A.
12. Relayer completes packet lifecycle for in-flight packets from chain A to chain B.
13. Relayer submits `MsgChannelUpgradeConfirm` on chain B.
14. Execution of `MsgChannelUpgradeConfirm` succeeds on chain B.
15. Relayer submits `MsgChannelUpgradeOpen` on chain A.
16. Execution of `MsgChannelUpgradeOpen` succeeds on chain A.

Sample exception flows:

- If two different relayers detect execution of `MsgChannelUpgradeInit` on both chain A and chain B and they submit `MsgChannelUpgradeTry` on the counterparty, then the handshake will finish `MsgChannelUpgradeOpen` after both chains execute `MsgChannelUpgradeAck` and all in-flight packets have been flushed (i.e. it is not needed to execute `MsgChannelUpgradeConfirm`).

# Functional requirements

## Assumptions and dependencies

- Functional relayer infrastructure is required to perform a channel upgrade.
- Chains wishing to successfully upgrade a channel must be using a minimum ibc-go version of v8.1.
- Chains proposing an upgrade must have the middleware or application module intended to be used in the channel upgrade configured.

## Features

### 1 - Configuration

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 1.01 | An on-chain parameter keeps a list of all connection IDs (e.g. [`connection-0`, `connection-1`]) for which channels are allowed to be upgraded for an upgrade proposed on a counterparty chain | TBD | `Deferred` |
| 1.02 | The on-chain parameter of connection IDs can only be updated by an authorized actor (e.g. governance) | TBD | `Deferred` |

### 2 - Initiation

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 2.01 | An upgrade initiated by an authorized actor (e.g. governance) is always allowed | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/keeper/msg_server_test.go#L952-L969) | `Verified` |
| 2.02 | A chain can configure a channel upgrade to be initiated automatically after a successful governance proposal | Provided by Cosmos SDK `x/gov` | `Verified` |
| 2.03 | After permission is granted for channels in a given connection to be upgraded, any relayer can continue the upgrade proposed on a counterparty chain | Only `MsgChannelUpgradeInit` is permissioned |`Verified` |
| 2.04 | A channel upgrade will be initiated when both `ChannelEnd`s are in the `OPEN` state | Acceptance tests in [`ChanUpgradeInit`](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade_test.go#L63-L69) and [`ChanUpgradeTry`](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade_test.go#L167-L173)| `Verified` |
| 2.05 | In the case of a crossing hello, a channel upgrade can be initiated when the counterparty has also executed the `ChanUpgradeInit` datagram with compatible parameters in the case of a crossing hello | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade_test.go#L141-L148) | `Verified` |

### 3 - Upgrade Handshake

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 3.01 | The upgrade proposing chain's channel state will remain `OPEN` after successful or unsuccessful execution of the `ChanUpgradeInit` datagram | [Channel state does not change in `WriteUpgradeInitChannel`](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade.go#L66) | `Verified` |
| 3.02 | If the counterparty chain accepts the upgrade its channel state will go from `OPEN` to `FLUSHING` after successful execution of the `ChanUpgradeTry` datagram, initiated by a relayer | [Move to `FLUSHING` in `startFlushing`](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade.go#L219) | `Verified` |
| 3.03 | A relayer must initiate the `ChanUpgradeAck` datagram on the upgrade proposing chain, on successful execution the channel state will go from `OPEN` to `FLUSHING`| [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade_test.go#L826) | `Verified` |
| 3.04 | The `ChanUpgradeAck` datagram informs the chain of the timeout period specified by the counterparty for the upgrade process to complete | [Stored in `WriteUpgradeAckChannel`](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade.go#L380) | `Verified` |
| 3.05 | When channel is in `FLUSHING` state, no new packets are allowed to be sent | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/packet_test.go#L232-L247) | `Verified` |
| 3.06 | When channel is in `FLUSHING` state, it is only allowed to receive packets with sequence number smaller or equal than the sequence number of the last packet sent on the counterparty when the channel moved to `FLUSHING` state | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/packet_test.go#L406-L427) | `Verified` |
| 3.07 | When channel is in `FLUSHING` state and packets are acknowledged or timed out, if the counterparty-specified timeout is reached, then the channel will be restored to its initial state, an error receipt will be written and the upgrade will be aborted | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/packet_test.go#L978-L1028) | `Verified` |
| 3.08 | Once in-flight packets have been flushed, the channel state shall change from `FLUSHING` to `FLUSHCOMPLETE` | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/packet_test.go#L924-L977) | `Verified` |
| 3.09 | A relayer must initiate the `ChanUpgradeConfirm` datagram on the counterparty to inform of the timeout period specified by the counterparty for the upgrade process to complete | [Stored in `WriteUpgradeConfirmChannel`](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade.go#L480) | `Verified` |
| 3.10 | Successful execution of the `ChanUpgradeConfirm` datagram when the channel state is `FLUSHCOMPLETE` changes the channel state to `OPEN` | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/keeper/msg_server_test.go#L1557-L1596) | `Verified` |
| 3.11 | If the channel state is `FLUSHING` when the `ChanUpgradeConfirm` datagram is called, `ChanUpgradeOpen` is later called to change the state to `OPEN` | [Move to `OPEN` in `WriteUpgradeOpenChannel`](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade.go#L627) | `Verified` |
| 3.12 | When both channel ends are in the `FLUSHCOMPLETE` state, a relayer can submit the `ChanUpgradeOpen` datagram to move the channel state to `OPEN` | [Move to `OPEN` in `WriteUpgradeOpenChannel`](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade.go#L627) | `Verified` |
| 3.13 | The counterparty chain may reject a proposed channel upgrade and the original channel on the proposing chain will be restored | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade_test.go#L215-L229) | `Verified` |
| 3.14 | If an upgrade handshake is unsuccessful, the original channel will be restored | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade_test.go#L2512-L2516) | `Verified` |
| 3.15 | A relayer can submit the `ChanUpgradeCancel` to cancel an upgrade which will successfully execute if the counterparty wrote an `ErrorReceipt`| [There must be an error receipt proof](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/upgrade.go#L652-L655) | `Verified` |
| 3.16 | If the `ChanUpgradeCancel` datagram is submitted by an authorized actor (e.g. governance) then the upgrade will be canceled without requiring the counterparty to write an `ErrorReceipt`| [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/keeper/msg_server_test.go#L2192-L2241) | `Verified` |
| 3.17 | An upgrade should be timed out if the chain has not completed flushing all pending packets within the timeout specified by the counterparty | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/packet_test.go#L978-L1028) | `Verified` |

# External interface requirements

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 4.01 | There should be a gRPC query to retrieve the upgrade fields and timeout stored for a given port ID and channel ID | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/grpc_query_test.go#L1849-L1854) | `Verified` |
| 4.02 | There should be a gRPC query to retrieve the upgrade error receipt stored for a given port ID and channel ID | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/core/04-channel/keeper/grpc_query_test.go#L1755-L1769) | `Verified` |

# Non-functional requirements

| ID | Description | Verification | Status |
| -- | ----------- | ------------ | ------ |
| 5.01 | A malicious actor should not be able to compromise the liveness of a channel | Verified during security audit | `Verified` |
