<!--
order: 7
-->

# Existence and Non-Existence Proofs 

IBC uses merkle proofs in order to verify the state of a remote counterparty state machine given a trusted root, and [ICS23](https://github.com/cosmos/ics23/tree/master/go) is a general approach for verifying merkle trees which is used in `ibc-go`.

Currently, all Cosmos SDK modules contain their own stores, which maintain the state of the application module in an IAVL (immutable AVL) binary merkle tree format. Specifically with regard to IBC, core IBC maintains its own IAVL store, and IBC apps (e.g. transfer) maintain their own dedicated stores. The Cosmos SDK multistore therefore creates a simple merkle tree of all of these IAVL trees, and from each of these individual IAVL tree root hashes derives a root hash for the application state tree as a whole (the apphash).

For the purposes of `ibc-go`, there are two types of proofs which are important: existence and non-existence proofs, terms which have been used interchangeably with membership and non-membership proofs. For the purposes of this guide, we will stick withe 'existence' and 'non-existence'. Existence proofs are used for transactions which will result in the writing of a packet receipt into the IBC store on the receiving end of the transaction (ie: token transfers, channel/connection handshakes), whereas non-existence proofs are used to timeout IBC packets.

## Existence Proofs

Put simply, existence proofs prove that a particular key and value exists in the tree -- that the counterparty has written a packet receipt into the store. Under the hood, an IBC existence proof comprises of two  proofs: an IAVL proof that the key exists in IBC store/IBC root hash, and a proof that the IBC root hash exists in the multistore root hash.

## Non-Existence Proofs

Non-existence proofs are used to prove that a key does NOT exist in the store. As stated above, these types of proofs are used to timeout packets, and prove that counterparty has not written a packet receipt into the store ie: that a token transfer has NOT successfully occurred.

There are cases where there is a necessity to "mock" non-existence proofs if the counterparty does not have ability to prove absence.

Since the verification method is designed to give complete control to client implementations, clients can support chains that do not provide absence proofs by verifying the existence of a non-empty sentinel `ABSENCE` value. In these special cases, the proof provided will be an ICS-23 `Existence` proof, and the client will verify that the `ABSENCE` value is stored under the given path for the given height.

