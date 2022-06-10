<!--
order: 1
-->

# Overview

Learn about what the Fee Middleware module is, and how to build custom modules that utilize the Fee Middleware functionality {synopsis}

## What is the Fee Middleware module?

IBC does not depend on relayer operators for transaction verification. However, the relayer infrastructure ensures liveness of the Interchain network — operators listen for packets sent through channels opened between chains, and perform the vital service of ferrying these packets (and proof of the transaction on the sending chain/receipt on the receiving chain) to the clients on each side of the channel. 

Though relaying is permissionless and completely decentralized and accessible, it does come with operational costs. Running full nodes to query transaction proofs and paying for transaction fees associated with IBC packets are two of the primary cost burdens which have driven the overall discussion on a general, in-protocol incentivization mechanism for relayers.

Initially, a [simple proposal](https://github.com/cosmos/ibc/pull/577/files) was created to incentivize relaying on ICS20 token transfers on the destination chain. However, the proposal was specific to ICS20 token transfers and would have to be reimplemented in this format on every other IBC application module. 

After much discussion, the proposal was expanded to a [general incentivisation design](https://github.com/cosmos/ibc/tree/master/spec/app/ics-029-fee-payment) that can be adopted by any ICS application protocol as [middleware](/docs/ibc/middleware/develop.md). THe first version of fee payments middleware will only support incentivisation of new channels, however, channel upgradeability will enable incentivisation of all existing channels.

## Concepts 

ICS29 fee payments in this middleware design are built on the assumption that sender chains are the source of incentives — the chain that sends the packets is the same chain that pays out fees to operators. Therefore, the middleware enables the registering of addresses associated with each party involved in relaying the packet on the source chain, and the escrowing of fees by any party which will be paid out to each party on completion of the packet lifecycle. This registration process can be automated on start up of relayer infrastructure.

`forward relayer`: The relayer that submits the recvPacket message for a given packet (on the destination chain)

`reverse relayer`: The relayer that submits the acknowledgePacket message for a given packet (on the source chain)

`timeout relayer`: The relayer that submits the timeoutPacket or timeoutOnClose messages for a given packet (on the source chain)

`payee`: The account address on the source chain to be paid on completion of the packet lifecycle

`counterparty payee`: The account address to be paid on completion of the packet lifecycle on the destination chain

`refund address`: The address of the account paying for the fees