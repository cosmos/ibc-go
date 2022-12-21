# Business requirements

> **TL;DR**: There is no general incentivization mechanism to reward relayers for their service ensuring Interchain liveness.

## Problem

The Interchain dream will never scale unless there is a clear way to incentivize relayers. An interface that can be easily adopted by any applications is needed, without precluding chains that do not use tokens.

Though IBC does not depend on relayer operators for transaction verification, the relayer infrastructure ensures liveness of the Interchain network — operators listen for packets sent through channels opened between chains, and perform the vital service of ferrying these packets (and proof of the transaction on the sending chain/receipt on the receiving chain) to the clients on each side of the channel. Though relaying is permissionless and completely decentralized and accessible, it does come with operational costs. Running full nodes to query transaction proofs and paying for transaction fees associated with IBC packets are two of the primary cost burdens which have driven the overall discussion on a general incentivization mechanism for relayers.

## Objectives

Provide a general fee payment design that can be adopted by any ICS application protocol as middleware.

## Scope

| Features  | Release |
| --------- | ------- |
| Incentivize timely delivery of a packet (successfully submit `MsgRecvPacket`). | v1 |
| Incentivize relaying acknowledgement for a packet (successfully submit `MsgAcknowledgement`). | v1 | 
| Incentivize relaying timeout for a packet when the timeout has expired before packet is delivered (successfully submit `MsgTimeout`). | v1 |
| Enable permissionless or permissioned relaying. | v1 |
| Allow opt-in for chains (an application with fee support on chain A could connect to the couterparty application without fee support on chain B). | v1 |

# User requirements

## Use cases

- A packet already sent may be incentivized for one or more of the messages `MsgRecvPacket`, `MsgAcknowledgement`, `MsgTimeout`.
- The next packet to be sent may be incentivized for one or more of the messages `MsgRecvPacket`, `MsgAcknowledgement`, `MsgTimeout`.

# Functional requirements

## Assumptions

1. Relayer addresses should not be forgeable.

## Known limitations

1. Without channel upgradability it is not possible to use fee incentivization on an existing channel.

## Terminology

See section [Definitions](https://github.com/cosmos/ibc/tree/main/spec/app/ics-029-fee-payment#definitions) in ICS 29 spec.

## Features

### 1 - Configuration

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 1.02 | A chain shall have the ability to export the fee middleware genesis state. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/genesis_test.go#L69) | `Verified` | 
| 1.03 | A chain shall have the ability to initialize the fee middleware genesis state. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/genesis_test.go#L9) | `Verified` | 

### 2 - Payee registration

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 2.01 | A relayer shall have the ability to register for a channel an optional payee address to which fees shall be distributed for successful relaying `MsgAcknowledgement` and `MsgTimeout`. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L23) | `Verified` |
| 2.02 | The payee address shall only be registered if the channel exist. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L28) | `Verified` |
| 2.03 | The payee address shall only be registered if the channel is fee enabled. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L35) | `Verified` |
| 2.04 | The payee address shall only be registered if the address is a valid `Bech32` address. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L42) | `Verified` |
| 2.05 | The payee address shall only be registered if the address is not blocked. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L49) | `Verified` |

### 3 - Counterparty payee registration

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 3.01 | A relayer shall have the ability to register a counterparty payee to which fees shall be distributed for successful relaying of `MsgRecvPacket`. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L102) | `Verified` |
| 3.02 | The counterparty payee address shall only be registered if the channel exists. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L115) | `Verified` |
| 3.03 | The counterparty payee address shall only be registered if the channel is fee enabled. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L122) | `Verified` |
| 3.04 | The counterparty payee address may be an arbitrary string (since the counterparty chain may not be Cosmos-SDK based). | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L42) | `Verified` |

### 4 - Fee incentivization

#### Synchronously

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 4.01 | If a channel is fee-enabled, the next sequence packet may be incentivized. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L178) | `Verified` |
| 4.02 | If the fee module is not locked, the next sequence packet may be incentivized. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L210) | `Verified` |
| 4.03 | The next sequence packet may be incentivized for some or all of `MsgRecvPacket`, `MsgAcknowledgement`, `MsgTimeout`. | Fees for any of the messages may be zero, but [not all of them](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/types/fee.go#L87). | `Verified` | 
| 4.04 | A blocked account address shall not be allowed to incentivize the next sequence packet. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L239) | `Verified` |
| 4.05 | A non-valid `Bech32` account address shall not be allowed to incentivize the next sequence packet. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L225) | `Verified` |
| 4.06 | The next sequence packet may be incentivized by multiple account addresses. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L183) | `Verified` |
| 4.07 | The next sequence packet shall only be incentivized with valid coin denominations. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L246-L266) | `Verified` |

#### Aynchronously

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 4.08 | Only packets that have been sent may be incentivized. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L376) | `Verified` | 
| 4.09 | A packet sent may be incentivized for some or all of `MsgRecvPacket`, `MsgAcknowledgement`, `MsgTimeout`. | Fees for any of the messages may be zero, but [not all of them](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/types/fee.go#L87). | `Verified` |
| 4.10 | Only packets that have not gone through the packet life cycle (i.e. have not been acknowledged or timed out yet) may be incentivized. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L382-L410) | `Verified` |
| 4.11 | If a channel exists, a packet sent may be incentivized. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L365) |
| 4.12 | If a channel is fee-enabled, a packet sent may be incentivized. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L357) | `Verified` |
| 4.13 | If the fee module is not locked, a packet sent may be incentivized. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L350) | `Verified` |
| 4.14 | A non-valid `Bech32` account address shall not be allowed to incentivize a packet sent. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L412) | `Verified` |
| 4.15 | An account that does not exist shall not be allowed to incentivize a packet sent. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L419) | `Verified` |
| 4.16 | A blocked account shall not be allowed to incentivize a packet sent. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/msg_server_test.go#L426) | `Verified` | 

### 5 - Fee distribution

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 5.01 | Fee distribution shall occurr on the source chain from which packets originate. | Either in [`OnAcknowledgementPacket`](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/ibc_middleware.go#L288) or in [`OnTimeoutPacket`](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/ibc_middleware.go#L330). | `Verified` | 

#### `OnAcknowledgementPacket`

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 5.02 | Fees for successful relaying of `MsgAcknowledgement` shall be distributed to the relayer address (or its associated payee address if one has been registered). | If a payee address exists for the relayer address, then [fees are distributed to the payee address](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/ibc_middleware.go#L288); otherwise [fees are distributed to the relayer address](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/ibc_middleware.go#L277). | `Verified` |
| 5.03 | Fees for successful relaying of `MsgRecvPacket` shall be distributed to the payee address of the relayer address. | [Fees are distributed to the payee address registered on the counterparty chain](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow.go#L93) if it is a valid `Bech32` address and is not a blocked address. | `Verified` |

#### `OnTimeoutPacket`

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 5.05 | Fees for successful relaying of `MsgTimeout` shall be distributed to the relayer address (or its associated payee address if one has been registered). | If a payee address exists for the relayer address, then [fees are distributed to the payee address](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/ibc_middleware.go#L330); otherwise [fees are distributed to the relayer address](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/ibc_middleware.go#L319). | `Verified` |

### 6 - Fee refunding

#### `OnAcknowledgementPacket`

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 6.01 | On successful processing of `MsgAcknowledgement`, fees for successful relay of `MsgTimeout` shall be refunded. | [Fees are refunded is the refund address is a valid `Bech32` address](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow.go#L103). | `Verified` | 
| 6.02 | On successful processing of `MsgAcknowledgement`, if fees for successful relay of `MsgRecvPacket` cannot be distributed, then they should be refunded. | [Fees are refunded](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow.go#L96) if the payee address registered on the counterparty chain for the relayer is either [invalid](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow_test.go#L105) or a [blocked address](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow_test.go#L120). | `Verified` |

#### `OnTimeoutPacket`

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 6.03 | On successful processing of `MsgTimeout`, fees for successful relay of `MsgRecvPacket` shall be refunded. | [Fees are refunded is the refund address is a valid `Bech32` address](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow.go#L146). | `Verified` |
| 6.04 | On successful processing of `MsgTimeout`, fees for successful relay of `MsgAcknowledgement` shall be refunded. | [Fees are refunded is the refund address is a valid `Bech32` address](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow.go#L149). | `Verified` | 

#### Channel closure

| ID  | Description | Verification | Status |
| --- | ----------- | ------------ | ------ |
| 6.04 | Packet fees are refunded on channel closure. |  Both on [`OnChanCloseInit`](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/ibc_middleware.go#L182)  and [`OnChanCloseConfirm`](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/ibc_middleware.go#L207). | `Verified` |  

# Non-functional requirements

## 7 - Security

| ID | Description | Verification | Status |  
| -- | ----------- | ------------ | ------ |      
| 7.01 | If the escrow account does not have sufficient funds while distributing fees in `OnAcknowledgementPacket`, the fee module shall become locked. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow_test.go#L85) | `Verified` |
| 7.02 | If the escrow account does not have sufficient funds while distributing fees in `OnTimeoutPacket`, the fee module shall become locked. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow_test.go#L236) | `Verified` |
| 7.03 | If the escrow account does not have sufficient funds while refunding fees in channel closure, the fee module shall become locked. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/keeper/escrow_test.go#L377) | `Verified` |

# External interface requirements

## 8 - CLI

### Query

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 8.01 | There shall be a CLI command available to query for the incentivization of an unrelayed packet by port ID, channel ID and packet sequence. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#L64) | `Verified` |
| 8.02 | There shall be a CLI command to query for the incentivization of all unrelayed packets on all open channels. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#L64) | `Verified` |
| 8.03 | There shall be a CLI command available to query for the total fees for `MsgRecvPacket` for a given port ID, channel ID and packet sequence. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#L105) | `Verified` |
| 8.04 | There shall be a CLI command available to query for the total fees for `MsgAcknowledgement` for a given port ID, channel ID and packet sequence. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#L151) | `Verified` |
| 8.05 | There shall be a CLI command available to query for the total fees for `MsgTimeout` for a given port ID, channel ID and packet sequence. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#L197) | `Verified` |
| 8.06 | There shall be a CLI command available to query for the payee address registered for a relayer for a specific channel ID. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#243) | `Verified` |
| 8.07 | There shall be a CLI command available to query for the counterparty payee address registered for a relayer for a specific channel ID. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#282) | `Verified` |
| 8.08 | There shall be a CLI command available to query if a channel is fee-enabled for a port ID and channel ID. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#L362) | `Verified` |
| 8.09 | There shall be a CLI command available to query for all the unrelayed incentivized packets in a channel by a port ID and channel ID. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/query.go#L397) | `Verified` | 

### Transaction

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 8.10 | There shall be a CLI command available to register a payee address for a channel by port ID and channel ID. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/tx.go#L26) | `Verified` |
| 8.11 | There shall be a CLI command available to register a counterparty payee address for a channel by port ID and channel ID. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/tx.go#L51) | `Verified` |
| 8.12 | There shall be a CLI command available to incentivize a packet by port ID, channel ID and packet sequence. | [CLI](https://github.com/cosmos/ibc-go/blob/v4.0.0/modules/apps/29-fee/client/cli/tx.go#L76) | `Verified` |