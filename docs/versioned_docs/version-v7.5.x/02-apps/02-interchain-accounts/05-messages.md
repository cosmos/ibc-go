---
title: Messages
sidebar_label: Messages
sidebar_position: 5
slug: /apps/interchain-accounts/messages
---


# Messages

## `MsgRegisterInterchainAccount`

An Interchain Accounts channel handshake can be initiated using `MsgRegisterInterchainAccount`:

```go
type MsgRegisterInterchainAccount struct {
  Owner        string
  ConnectionID string
  Version      string
}
```

This message is expected to fail if:

- `Owner` is an empty string.
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).

This message will construct a new `MsgChannelOpenInit` on chain and route it to the core IBC message server to initiate the opening step of the channel handshake.

The controller submodule will generate a new port identifier and claim the associated port capability. The caller is expected to provide an appropriate application version string. For example, this may be an ICS-27 JSON encoded [`Metadata`](https://github.com/cosmos/ibc-go/blob/v6.0.0/proto/ibc/applications/interchain_accounts/v1/metadata.proto#L11) type or an ICS-29 JSON encoded [`Metadata`](https://github.com/cosmos/ibc-go/blob/v6.0.0/proto/ibc/applications/fee/v1/metadata.proto#L11) type with a nested application version. 
If the `Version` string is omitted, the controller submodule will construct a default version string in the `OnChanOpenInit` handshake callback.

```go
type MsgRegisterInterchainAccountResponse struct {
  ChannelID string
  PortID string
}
```

The `ChannelID` and `PortID` are returned in the message response.

## `MsgSendTx`

An Interchain Accounts transaction can be executed on a remote host chain by sending a `MsgSendTx` from the corresponding controller chain:

```go
type MsgSendTx struct {
  Owner           string
  ConnectionID    string
  PacketData      InterchainAccountPacketData 
  RelativeTimeout uint64
}
```

This message is expected to fail if:

- `Owner` is an empty string.
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).
- `PacketData` contains an `UNSPECIFIED` type enum, the length of `Data` bytes is zero or the `Memo` field exceeds 256 characters in length.
- `RelativeTimeout` is zero.

This message will create a new IBC packet with the provided `PacketData` and send it via the channel associated with the `Owner` and `ConnectionID`.
The `PacketData` is expected to contain a list of serialized `[]sdk.Msg` in the form of `CosmosTx`. Please note the signer field of each `sdk.Msg` must be the interchain account address. 
When the packet is relayed to the host chain, the `PacketData` is unmarshalled and the messages are authenticated and executed.

```go
type MsgSendTxResponse struct {
  Sequence uint64
}
```

The packet `Sequence` is returned in the message response.

### Queries

It is possible to use [`MsgModuleQuerySafe`](https://github.com/cosmos/ibc-go/blob/v7.5.0/proto/ibc/applications/interchain_accounts/host/v1/tx.proto#L32-L39) to execute a list of queries on the host chain. This message can be included in the list of encoded `sdk.Msg`s of `InterchainPacketData`. The host chain will return on the acknowledgment the responses for all the queries. Please note that only module safe queries can be executed ([deterministic queries that are safe to be called from within the state machine](https://docs.cosmos.network/main/build/building-modules/query-services#calling-queries-from-the-state-machine)). 
 
The queries available from Cosmos SDK are:

```plaintext
/cosmos.staking.v1beta1.Query/Validators,
/cosmos.staking.v1beta1.Query/Validator,
/cosmos.staking.v1beta1.Query/ValidatorDelegations",
/cosmos.staking.v1beta1.Query/ValidatorUnbondingDelegations
/cosmos.staking.v1beta1.Query/Delegation
/cosmos.staking.v1beta1.Query/UnbondingDelegation
/cosmos.staking.v1beta1.Query/DelegatorDelegations
/cosmos.staking.v1beta1.Query/DelegatorUnbondingDelegations
/cosmos.staking.v1beta1.Query/Redelegations
/cosmos.staking.v1beta1.Query/DelegatorValidators
/cosmos.staking.v1beta1.Query/DelegatorValidator
/cosmos.staking.v1beta1.Query/HistoricalInfo
/cosmos.staking.v1beta1.Query/Pool
/cosmos.staking.v1beta1.Query/Params
/cosmos.bank.v1beta1.Query/Balance
/cosmos.bank.v1beta1.Query/AllBalances
/cosmos.bank.v1beta1.Query/SpendableBalances
/cosmos.bank.v1beta1.Query/SpendableBalanceByDenom
/cosmos.bank.v1beta1.Query/TotalSupply
/cosmos.bank.v1beta1.Query/SupplyOf
/cosmos.bank.v1beta1.Query/Params
/cosmos.bank.v1beta1.Query/DenomMetadata
/cosmos.bank.v1beta1.Query/DenomsMetadata
/cosmos.bank.v1beta1.Query/DenomOwners
/cosmos.bank.v1beta1.Query/SendEnabled
/cosmos.auth.v1beta1.Query/Accounts
/cosmos.auth.v1beta1.Query/Account
/cosmos.auth.v1beta1.Query/AccountAddressByID
/cosmos.auth.v1beta1.Query/Params
/cosmos.auth.v1beta1.Query/ModuleAccounts
/cosmos.auth.v1beta1.Query/ModuleAccountByName
/cosmos.auth.v1beta1.Query/AccountInfo
```

The following code block shows an example of how `MsgModuleQuerySafe` can be used to query the account balance of an account on the host chain. The resulting packet data variable is used to set the `PacketData` of `MsgSendTx`.

```go
balanceQuery := banktypes.NewQueryBalanceRequest("cosmos1...", "uatom")
queryBz, err := balanceQuery.Marshal()

// signer of message must be the interchain account on the host
queryMsg := icahosttypes.NewMsgModuleQuerySafe("cosmos2...", []*icahosttypes.QueryRequest{
  {
    Path: "/cosmos.bank.v1beta1.Query/Balance",
    Data: queryBz,
  },
})

bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{queryMsg}, icatypes.EncodingProtobuf)

packetData := icatypes.InterchainAccountPacketData{
  Type: icatypes.EXECUTE_TX,
  Data: bz,
  Memo: "",
}
```

## Atomicity

As the Interchain Accounts module supports the execution of multiple transactions using the Cosmos SDK `Msg` interface, it provides the same atomicity guarantees as Cosmos SDK-based applications, leveraging the [`CacheMultiStore`](https://docs.cosmos.network/main/learn/advanced/store#cachemultistore) architecture provided by the [`Context`](https://docs.cosmos.network/main/learn/advanced/context.html) type. 

This provides atomic execution of transactions when using Interchain Accounts, where state changes are only committed if all `Msg`s succeed.
