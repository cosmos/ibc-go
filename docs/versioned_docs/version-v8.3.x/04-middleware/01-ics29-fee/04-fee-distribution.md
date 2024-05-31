---
title: Fee Distribution
sidebar_label: Fee Distribution
sidebar_position: 4
slug: /middleware/ics29-fee/fee-distribution
---

# Fee distribution

:::note Synopsis
Learn about payee registration for the distribution of packet fees. The following document is intended for relayer operators.
:::

:::note

## Pre-requisite readings

- [Fee Middleware](01-overview.md)

:::

Packet fees are divided into 3 distinct amounts in order to compensate relayer operators for packet relaying on fee enabled IBC channels.

- `RecvFee`: The sum of all packet receive fees distributed to a payee for successful execution of `MsgRecvPacket`.
- `AckFee`: The sum of all packet acknowledgement fees distributed to a payee for successful execution of `MsgAcknowledgement`.
- `TimeoutFee`: The sum of all packet timeout fees distributed to a payee for successful execution of `MsgTimeout`.

## Register a counterparty payee address for forward relaying

As mentioned in [ICS29 Concepts](01-overview.md#concepts), the forward relayer describes the actor who performs the submission of `MsgRecvPacket` on the destination chain.
Fee distribution for incentivized packet relays takes place on the packet source chain.

> Relayer operators are expected to register a counterparty payee address, in order to be compensated accordingly with `RecvFee`s upon completion of a packet lifecycle.

The counterparty payee address registered on the destination chain is encoded into the packet acknowledgement and communicated as such to the source chain for fee distribution.
**If a counterparty payee is not registered for the forward relayer on the destination chain, the escrowed fees will be refunded upon fee distribution.**

### Relayer operator actions

A transaction must be submitted **to the destination chain** including a `CounterpartyPayee` address of an account on the source chain.
The transaction must be signed by the `Relayer`.

Note: If a module account address is used as the `CounterpartyPayee` but the module has been set as a blocked address in the `BankKeeper`, the refunding to the module account will fail. This is because many modules use invariants to compare internal tracking of module account balances against the actual balance of the account stored in the `BankKeeper`. If a token transfer to the module account occurs without going through this module and updating the account balance of the module on the `BankKeeper`, then invariants may break and unknown behaviour could occur depending on the module implementation. Therefore, if it is desirable to use a module account that is currently blocked, the module developers should be consulted to gauge to possibility of removing the module account from the blocked list.

```go
type MsgRegisterCounterpartyPayee struct {
  // unique port identifier
  PortId string
  // unique channel identifier
  ChannelId string
  // the relayer address
  Relayer string
  // the counterparty payee address
  CounterpartyPayee string
}
```

> This message is expected to fail if:
>
> - `PortId` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators).
> - `ChannelId` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).
> - `Relayer` is an invalid address (see [Cosmos SDK Addresses](https://github.com/cosmos/cosmos-sdk/blob/main/docs/learn/beginner/03-accounts.md#addresses)).
> - `CounterpartyPayee` is empty.

See below for an example CLI command:

```bash
simd tx ibc-fee register-counterparty-payee transfer channel-0 \
  cosmos1rsp837a4kvtgp2m4uqzdge0zzu6efqgucm0qdh \
  osmo1v5y0tz01llxzf4c2afml8s3awue0ymju22wxx2 \
  --from cosmos1rsp837a4kvtgp2m4uqzdge0zzu6efqgucm0qdh
```

## Register an alternative payee address for reverse and timeout relaying

As mentioned in [ICS29 Concepts](01-overview.md#concepts), the reverse relayer describes the actor who performs the submission of `MsgAcknowledgement` on the source chain.
Similarly the timeout relayer describes the actor who performs the submission of `MsgTimeout` (or `MsgTimeoutOnClose`) on the source chain.

> Relayer operators **may choose** to register an optional payee address, in order to be compensated accordingly with `AckFee`s and `TimeoutFee`s upon completion of a packet life cycle.

If a payee is not registered for the reverse or timeout relayer on the source chain, then fee distribution assumes the default behaviour, where fees are paid out to the relayer account which delivers `MsgAcknowledgement` or `MsgTimeout`/`MsgTimeoutOnClose`.

### Relayer operator actions

A transaction must be submitted **to the source chain** including a `Payee` address of an account on the source chain.
The transaction must be signed by the `Relayer`.

Note: If a module account address is used as the `Payee` it is recommended to [turn off invariant checks](https://github.com/cosmos/ibc-go/blob/v7.0.0/testing/simapp/app.go#L727) for that module.

```go
type MsgRegisterPayee struct {
  // unique port identifier
  PortId string
  // unique channel identifier
  ChannelId string
  // the relayer address
  Relayer string
  // the payee address
  Payee string
}
```

> This message is expected to fail if:
>
> - `PortId` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators).
> - `ChannelId` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).
> - `Relayer` is an invalid address (see [Cosmos SDK Addresses](https://github.com/cosmos/cosmos-sdk/blob/main/docs/learn/beginner/03-accounts.md#addresses)).
> - `Payee` is an invalid address (see [Cosmos SDK Addresses](https://github.com/cosmos/cosmos-sdk/blob/main/docs/learn/beginner/03-accounts.md#addresses)).

See below for an example CLI command:

```bash
simd tx ibc-fee register-payee transfer channel-0 \
  cosmos1rsp837a4kvtgp2m4uqzdge0zzu6efqgucm0qdh \
  cosmos153lf4zntqt33a4v0sm5cytrxyqn78q7kz8j8x5 \
  --from cosmos1rsp837a4kvtgp2m4uqzdge0zzu6efqgucm0qdh
```
