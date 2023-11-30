---
title: Contracts
sidebar_label: Contracts
sidebar_position: 7
slug: /ibc/light-clients/wasm/contracts
---

# Contracts

Learn about the expected behaviour of Wasm light client contracts and the between with `08-wasm`. {synopsis}

## API

The `08-wasm` light client proxy performs calls to the Wasm light client via the Wasm VM. The calls require as input JSON-encoded payload messages that fall in the three categories described in the next sections. 

## `InstantiateMessage`

This is the message sent to the contract's `instantiate` entry point. It contains the bytes of the protobuf-encoded client and consensus states of the underlying light client, both provided in [`MsgCreateClient`](https://github.com/cosmos/ibc-go/blob/v7.2.0/proto/ibc/core/client/v1/tx.proto#L25-L37). Please note that the bytes contained within the JSON message are represented as base64-encoded strings.

```go
type InstantiateMessage struct {
	ClientState    []byte `json:"client_state"`
	ConsensusState []byte `json:"consensus_state"`
	Checksum       []byte `json:"checksum"
}
```

The Wasm light client contract is expected to store the client and consensus state in the corresponding keys of the client-prefixed store.

## `QueryMsg`

`QueryMsg` acts as a discriminated union type that is used to encode the messages that are sent to the contract's `query` entry point. Only one of the fields of the type should be set at a time, so that the other fields are omitted in the encoded JSON and the payload can be correctly translated to the corresponding element of the enumeration in Rust.

```go
type QueryMsg struct {
  Status               *StatusMsg               `json:"status,omitempty"`
  ExportMetadata       *ExportMetadataMsg       `json:"export_metadata,omitempty"`
  TimestampAtHeight    *TimestampAtHeightMsg    `json:"timestamp_at_height,omitempty"`
  VerifyClientMessage  *VerifyClientMessageMsg  `json:"verify_client_message,omitempty"`
  CheckForMisbehaviour *CheckForMisbehaviourMsg `json:"check_for_misbehaviour,omitempty"`
}
```

```rust
#[cw_serde]
pub enum QueryMsg {
  Status(StatusMsg),
  ExportMetadata(ExportMetadataMsg),
  TimestampAtHeight(TimestampAtHeightMsg),
  VerifyClientMessage(VerifyClientMessageRaw),
  CheckForMisbehaviour(CheckForMisbehaviourMsgRaw),
}
```

To learn what it is expected from the Wasm light client contract when processing each message, please read the corresponsing section of the [Light client developer guide](../01-developer-guide/01-overview.md):

- For `StatusMsg`, see the section [`Status` method](../01-developer-guide/02-client-state.md#status-method).
- For `ExportMetadataMsg`, see the section [Genesis metadata](../01-developer-guide/08-genesis.md#genesis-metadata).
- For `TimestampAtHeightMsg`, see the section [`GetTimestampAtHeight` method](../01-developer-guide/02-client-state.md#gettimestampatheight-method).
- For `VerifyClientMessageMsg`, see the section [`VerifyClientMessage`](../01-developer-guide/04-updates-and-misbehaviour.md#verifyclientmessage).
- For `CheckForMisbehaviourMsg`, see the section [`CheckForMisbehaviour` method](../01-developer-guide/02-client-state.md#checkformisbehaviour-method).

## `SudoMsg`

`SudoMsg` acts as a discriminated union type that is used to encode the messages that are sent to the contract's `sudo` entry point. Only one of the fields of the type should be set at a time, so that the other fields are omitted in the encoded JSON and the payload can be correctly translated to the corresponding element of the enumeration in Rust.

The `sudo` entry point is able to perform state-changing writes in the client-prefixed store.

```go
type SudoMsg struct {
  UpdateState                 *UpdateStateMsg                 `json:"update_state,omitempty"`
  UpdateStateOnMisbehaviour   *UpdateStateOnMisbehaviourMsg   `json:"update_state_on_misbehaviour,omitempty"`
  VerifyUpgradeAndUpdateState *VerifyUpgradeAndUpdateStateMsg `json:"verify_upgrade_and_update_state,omitempty"`
  VerifyMembership            *VerifyMembershipMsg            `json:"verify_membership,omitempty"`
  VerifyNonMembership         *VerifyNonMembershipMsg         `json:"verify_non_membership,omitempty"`
  MigrateClientStore          *MigrateClientStoreMsg          `json:"migrate_client_store,omitempty"`
}
```

```rust
#[cw_serde]
pub enum SudoMsg {
  UpdateState(UpdateStateMsgRaw),
  UpdateStateOnMisbehaviour(UpdateStateOnMisbehaviourMsgRaw),
  VerifyUpgradeAndUpdateState(VerifyUpgradeAndUpdateStateMsgRaw),
  VerifyMembership(VerifyMembershipMsgRaw),
  VerifyNonMembership(VerifyNonMembershipMsgRaw),
  MigrateClientStore(MigrateClientStoreMsgRaw),
}
```

To learn what it is expected from the Wasm light client contract when processing each message, please read the corresponsing section of the [Light client developer guide](../01-developer-guide/01-overview.md):

- For `UpdateStateMsg`, see the section [`UpdateState`](../01-developer-guide/04-updates-and-misbehaviour.md#updatestate).
- For `UpdateStateOnMisbehaviourMsg`, see the section [`UpdateStateOnMisbehaviour`](../01-developer-guide/04-updates-and-misbehaviour.md#updatestateonmisbehaviour).
- For `VerifyUpgradeAndUpdateStateMsg`, see the section [`GetTimestampAtHeight` method](../01-developer-guide/05-upgrades.md#implementing-verifyupgradeandupdatestate).
- For `VerifyMembershipMsg`, see the section [`VerifyMembership` method](../01-developer-guide/02-client-state.md#verifymembership-method).
- For `VerifyNonMembershipMsg`, see the section [`VerifyNonMembership` method](../01-developer-guide/02-client-state.md#verifynonmembership-method).
- For `MigrateClientStoreMsg`, see the section [Implementing `CheckSubstituteAndUpdateState`](../01-developer-guide/07-proposals.md#implementing-checksubstituteandupdatestate).

### Migration

The `08-wasm` proxy light client exposes the `MigrateContract` RPC endpoint that can be used to migrate a given Wasm light client contract (specified by the client identifier) to a new Wasm byte code (specified by the hash of the byte code). The expected use case for this RPC endpoint is to enable contracts to migrate to new byte code in case the current byte code is found to have a bug or vulnerability. The Wasm byte code that contracts are migrated have to be uploaded beforehand using `MsgStoreCode` and must implement the `migrate` entry point. See section[`MsgMigrateContract`](./04-messages.md#msgmigratecontract) for information about the request messsage for this RPC endpoint. 

## Expected behaviour

The `08-wasm` proxy light client modules expects the following behaviour from the Wasm light client contracts when executing messages that perform state-changing writes:

- The contract must not delete the client state from the store.
- The contract must not change the client state to a client state of another type.
- The contract must not change the checksum in the client state.

Any violation of these rules will result in an error returned from `08-wasm` that will abort the transaction.
