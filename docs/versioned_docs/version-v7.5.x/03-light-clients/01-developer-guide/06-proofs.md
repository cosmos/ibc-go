---
title: Existence/Non-Existence Proofs
sidebar_label: Existence/Non-Existence Proofs
sidebar_position: 6
slug: /ibc/light-clients/proofs
---


# Existence and non-existence proofs 

IBC uses merkle proofs in order to verify the state of a remote counterparty state machine given a trusted root, and [ICS-23](https://github.com/cosmos/ics23/tree/master/go) is a general approach for verifying merkle trees which is used in ibc-go.

Currently, all Cosmos SDK modules contain their own stores, which maintain the state of the application module in an IAVL (immutable AVL) binary merkle tree format. Specifically with regard to IBC, core IBC maintains its own IAVL store, and IBC apps (e.g. transfer) maintain their own dedicated stores. The Cosmos SDK multistore therefore creates a simple merkle tree of all of these IAVL trees, and from each of these individual IAVL tree root hashes it derives a root hash for the application state tree as a whole (the `AppHash`).

For the purposes of ibc-go, there are two types of proofs which are important: existence and non-existence proofs, terms which have been used interchangeably with membership and non-membership proofs. For the purposes of this guide, we will stick with "existence" and "non-existence".

## Existence proofs

Existence proofs are used in IBC transactions which involve verification of counterparty state for transactions which will result in the writing of provable state. For example, this includes verification of IBC store state for handshakes and packets.

Put simply, existence proofs prove that a particular key and value exists in the tree. Under the hood, an IBC existence proof comprises of two  proofs: an IAVL proof that the key exists in IBC store/IBC root hash, and a proof that the IBC root hash exists in the multistore root hash.

## Non-existence proofs

Non-existence proofs verify the absence of data stored within counterparty state and are used to prove that a key does NOT exist in state. As stated above, these types of proofs can be used to timeout packets by proving that the counterparty has not written a packet receipt into the store, meaning that a token transfer has NOT successfully occurred.

Some trees (e.g. SMT) may have a sentinel empty child for non-existent keys. In this case, the ICS-23 proof spec should include this `EmptyChild` so that ICS-23 handles the non-existence proof correctly.

In some cases, there is a necessity to "mock" non-existence proofs if the counterparty does not have ability to prove absence. Since the verification method is designed to give complete control to client implementations, clients can support chains that do not provide absence proofs by verifying the existence of a non-empty sentinel `ABSENCE` value. In these special cases, the proof provided will be an ICS-23 `Existence` proof, and the client will verify that the `ABSENCE` value is stored under the given path for the given height.

## State verification methods: `VerifyMembership` and `VerifyNonMembership`

The state verification functions for all IBC data types have been consolidated into two generic methods, `VerifyMembership` and `VerifyNonMembership`.

From the [`ClientState` interface definition](https://github.com/cosmos/ibc-go/blob/e650be91614ced7be687c30eb42714787a3bbc59/modules/core/exported/client.go#L68-L91), we find:

```go
VerifyMembership(
    ctx sdk.Context,
    clientStore sdk.KVStore,
    cdc codec.BinaryCodec,
    height Height,
    delayTimePeriod uint64,
    delayBlockPeriod uint64,
    proof []byte,
    path Path,
    value []byte,
) error

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
VerifyNonMembership(
    ctx sdk.Context,
    clientStore sdk.KVStore,
    cdc codec.BinaryCodec,
    height Height,
    delayTimePeriod uint64,
    delayBlockPeriod uint64,
    proof []byte,
    path Path,
) error
```

Both are expected to be provided with a standardised key path, `exported.Path`, as defined in [ICS-24 host requirements](https://github.com/cosmos/ibc/tree/main/spec/core/ics-024-host-requirements). Membership verification requires callers to provide the value marshalled as `[]byte`. Delay period values should be zero for non-packet processing verification. A zero proof height is now allowed by core IBC and may be passed into `VerifyMembership` and `VerifyNonMembership`. Light clients are responsible for returning an error if a zero proof height is invalid behaviour.

Please refer to the [ICS-23 implementation](https://github.com/cosmos/ibc-go/blob/e093d85b533ab3572b32a7de60b88a0816bed4af/modules/core/23-commitment/types/merkle.go#L131-L205) for a concrete example.
