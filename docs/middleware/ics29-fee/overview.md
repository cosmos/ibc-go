<!--
order: 1
-->

# Overview

Learn about what the Fee Middleware module is, and how to build custom modules that utilize the Fee Middleware functionality {synopsis}

## What is the Fee Middleware module?

IBC does not depend on relayer operators for transaction verification. However, the relayer infrastructure ensures liveness of the Interchain network — operators listen for packets sent through channels opened between chains, and perform the vital service of ferrying these packets (and proof of the transaction on the sending chain/receipt on the receiving chain) to the clients on each side of the channel.

Though relaying is permissionless and completely decentralized and accessible, it does come with operational costs. Running full nodes to query transaction proofs and paying for transaction fees associated with IBC packets are two of the primary cost burdens which have driven the overall discussion on **a general, in-protocol incentivization mechanism for relayers**.

Initially, a [simple proposal](https://github.com/cosmos/ibc/pull/577/files) was created to incentivize relaying on ICS20 token transfers on the destination chain. However, the proposal was specific to ICS20 token transfers and would have to be reimplemented in this format on every other IBC application module.

After much discussion, the proposal was expanded to a [general incentivisation design](https://github.com/cosmos/ibc/tree/master/spec/app/ics-029-fee-payment) that can be adopted by any ICS application protocol as [middleware](../../ibc/middleware/develop.md).

## Concepts

ICS29 fee payments in this middleware design are built on the assumption that sender chains are the source of incentives — the chain on which packets are incentivized is the chain that distributes fees to relayer operators. However, as part of the IBC packet flow, messages have to be submitted on both sender and destination chains. This introduces the requirement of a mapping of relayer operator's addresses on both chains.

> To achieve the stated requirements, the **fee middleware module has two main groups of functionality**:

- Registering of relayer addresses associated with each party involved in relaying the packet on the source chain. This registration process can be automated on start up of relayer infrastructure and happens only once, not every packet flow.

  This is described in the [Fee distribution section](../ics29-fee/fee-distribution.md).

- Escrowing fees by any party which will be paid out to each rightful party on completion of the packet lifecycle.

  This is described in the [Fee messages section](../ics29-fee/msgs.md).

We complete the introduction by giving a list of definitions of relevant terminolgy.

`Forward relayer`: The relayer that submits the `MsgRecvPacket` message for a given packet (on the destination chain).

`Reverse relayer`: The relayer that submits the `MsgAcknowledgement` message for a given packet (on the source chain).

`Timeout relayer`: The relayer that submits the `MsgTimeout` or `MsgTimeoutOnClose` messages for a given packet (on the source chain).

`Payee`: The account address on the source chain to be paid on completion of the packet lifecycle. The packet lifecycle on the source chain completes with the receipt of a `MsgTimeout`/`MsgTimeoutOnClose` or a `MsgAcknowledgement`.

`Counterparty payee`: The account address to be paid on completion of the packet lifecycle on the destination chain. The package lifecycle on the destination chain completes with a successful `MsgRecvPacket`.

`Refund address`: The address of the account paying for the incentivization of packet relaying. The account is refunded timeout fees upon successful acknowledgement. In the event of a packet timeout, both acknowledgement and receive fees are refunded.

## Known Limitations

The first version of fee payments middleware will only support incentivisation of new channels, however, channel upgradeability will enable incentivisation of all existing channels.
