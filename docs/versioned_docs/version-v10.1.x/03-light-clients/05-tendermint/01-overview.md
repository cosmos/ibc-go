---
title: Overview
sidebar_label: Overview
sidebar_position: 1
slug: /ibc/light-clients/tendermint/overview
---

# `07-tendermint`

## Overview

:::note Synopsis
Learn about the 07-tendermint light client module.
:::

The Tendermint client is the first and most deployed light client in IBC. It implements the IBC [light client module interface](https://github.com/cosmos/ibc-go/blob/v9.0.0-beta.1/modules/core/exported/client.go#L41-L123) to track a counterparty running [CometBFT](https://github.com/cometbft/cometbft) consensus. 

:::note
Tendermint is the old name of CometBFT which has been retained in IBC to avoid expensive migration costs.
:::

The Tendermint client consists of two important structs that keep track of the state of the counterparty chain and allow for future updates. The `ClientState` struct contains all the parameters necessary for CometBFT header verification. The `ConsensusState`, on the other hand, is a compressed view of a particular header of the counterparty chain. Unlike off chain light clients, IBC does not store full header. Instead it stores only the information it needs to prove verification of key/value pairs in the counterparty state (i.e. the header `AppHash`), and the information necessary to use the consensus state as the next root of trust to add a new consensus state to the client (i.e. the header `NextValidatorsHash` and `Timestamp`). The relayer provides the full trusted header on `UpdateClient`, which will get checked against the compressed root-of-trust consensus state. If the trusted header matches a previous consensus state, and the trusted header and new header pass the CometBFT light client update algorithm, then the new header is compressed into a consensus state and added to the IBC client.

Each Tendermint Client is composed of a single `ClientState` keyed on the client ID, and multiple consensus states which are keyed on both the clientID and header height. Relayers can use the consensus states to verify merkle proofs of packet commitments, acknowledgements, and receipts against the `AppHash` of the counterparty chain in order to enable verified packet flow.

If a counterparty chain violates the CometBFT protocol in a way that is detectable to off-chain light clients, this misbehaviour can also be submitted to an IBC client by any off-chain actor. Upon verification of this misbehaviour, the Tendermint IBC Client will freeze, preventing any further packet flow from this malicious chain from occurring. Governance or some other out-of-band protocol may then be used to unwind any damage that has already occurred.

## Initialization

The Tendermint light client is initialized with a `ClientState` that contains parameters necessary for CometBFT header verification along with a latest height and `ConsensusState` that encapsulates the application state root of a trusted header that will serve to verify future incoming headers from the counterparty.

```proto
message ClientState {
  // human readable chain-id that will be included in header
  // and signed over by the validator set
  string   chain_id    = 1;
  // trust level is the fraction of the trusted validator set
  // that must sign over a new untrusted header before it is accepted
  // it can be a minimum of 1/3 and a maximum of 2/3
  // Note these are the bounds of liveness. 1/3 is the minimum 
  // honest stake needed to maintain liveness on a chain,
  // requiring more than 2/3 to sign over the new header would
  // break the BFT threshold of allowing 1/3 malicious validators
  Fraction trust_level = 2;
  // duration of the period since the LatestTimestamp during which the
  // submitted headers are valid for update
  google.protobuf.Duration trusting_period = 3;
  // duration of the staking unbonding period
  google.protobuf.Duration unbonding_period = 4;
  // defines how much new (untrusted) header's Time can drift 
  // into the future relative to our local clock.
  google.protobuf.Duration max_clock_drift = 5;

  // Block height when the client was frozen due to a misbehaviour
  ibc.core.client.v1.Height frozen_height = 6;
  // Latest height the client was updated to
  ibc.core.client.v1.Height latest_height = 7;

  // Proof specifications used in verifying counterparty state
  repeated cosmos.ics23.v1.ProofSpec proof_specs = 8;

  // Path at which next upgraded client will be committed.
  // Each element corresponds to the key for a single CommitmentProof in the
  // chained proof. NOTE: ClientState must stored under
  // `{upgradePath}/{upgradeHeight}/clientState` ConsensusState must be stored
  // under `{upgradepath}/{upgradeHeight}/consensusState` For SDK chains using
  // the default upgrade module, upgrade_path should be []string{"upgrade",
  // "upgradedIBCState"}`
  repeated string upgrade_path = 9;
}
```

```proto
message ConsensusState {
  // timestamp that corresponds to the block height in which the ConsensusState
  // was stored.
  google.protobuf.Timestamp timestamp = 1;
  // commitment root (i.e app hash) that will be used
  // to verify proofs of packet flow messages
  ibc.core.commitment.v1.MerkleRoot root = 2;
  // hash of the next validator set that will be used as
  // a new updated source of trust to verify future updates
  bytes next_validators_hash = 3;
}
```

## Updates

Once the initial client state and consensus state are submitted, future consensus states can be added to the client by submitting IBC [headers](https://github.com/cosmos/ibc-go/blob/v9.0.0-beta.1/proto/ibc/lightclients/tendermint/v1/tendermint.proto#L76-L94). These headers contain all necessary information to run the CometBFT light client protocol.

```proto
message Header {
  // this is the new signed header that we want to add
  // as a new consensus state to the ibc client. 
  // the signed header contains the commit signatures of the `validator_set` below
  .tendermint.types.SignedHeader signed_header = 1;

  // the validator set which signed the new header
  .tendermint.types.ValidatorSet validator_set      = 2;
  // the trusted height of the consensus state which we are updating from
  ibc.core.client.v1.Height      trusted_height     = 3;
  // the trusted validator set, the hash of the trusted validators must be equal to 
  // `next_validators_hash` of the current consensus state
  .tendermint.types.ValidatorSet trusted_validators = 4;
}
```

For detailed information on the CometBFT light client protocol and its safety properties please refer to the [original Tendermint whitepaper](https://arxiv.org/abs/1807.04938).

## Proofs

As consensus states are added to the client, they can be used for proof verification by relayers wishing to prove packet flow messages against a particular height on the counterparty. This uses the `VerifyMembership` and `VerifyNonMembership` methods on the Tendermint client.

```go
// VerifyMembership is a generic proof verification method
//which verifies a proof of the existence of a value at a 
// given CommitmentPath at the specified height. The caller
// is expected to construct the full CommitmentPath from a 
// CommitmentPrefix and a standardized path (as defined in ICS 24).
VerifyMembership(
    ctx sdk.Context,
    clientID string,
    height Height,
    delayTimePeriod uint64,
    delayBlockPeriod uint64,
    proof []byte,
    path Path,
    value []byte,
) error

// VerifyNonMembership is a generic proof verification method 
// which verifies the absence of a given CommitmentPath at a 
// specified height. The caller is expected to construct the 
// full CommitmentPath from a CommitmentPrefix and a standardized
// path (as defined in ICS 24).
VerifyNonMembership(
    ctx sdk.Context,
    clientID string,
    height Height,
    delayTimePeriod uint64,
    delayBlockPeriod uint64,
    proof []byte,
    path Path,
) error
```

The Tendermint client is initialized with an ICS23 proof spec. This allows the Tendermint implementation to support many different merkle tree structures so long as they can be represented in an [`ics23.ProofSpec`](https://github.com/cosmos/ics23/blob/go/v0.10.0/proto/cosmos/ics23/v1/proofs.proto#L145-L170).

## Misbehaviour

The Tendermint light client directly tracks consensus of a CometBFT counterparty chain. So long as the counterparty is Byzantine Fault Tolerant, that is to say, the malicious subset of the bonded validators does not exceed the trust level of the client, then the client is secure.

In case the malicious subset of the validators exceeds the trust level of the client, then the client can be deceived into accepting invalid blocks and the connection is no longer secure.

The Tendermint client has some mitigations in place to prevent this. If there are two valid blocks signed by the counterparty validator set at the same height [e.g. a valid block signed by an honest subset and an invalid block signed by a malicious one], then these conflicting headers can be submitted to the client as [misbehaviour](https://github.com/cosmos/ibc-go/blob/v9.0.0-beta.1/proto/ibc/lightclients/tendermint/v1/tendermint.proto#L65-L74). The client will verify the headers and freeze the client; preventing any future updates and proof verification from succeeding. This effectively halts communication with the compromised counterparty while out-of-band social consensus can unwind any damage done.

Similarly, if the timestamps of the headers are not monotonically increasing, this can also be evidence of malicious behaviour and cause the client to freeze.

Thus, any consensus faults that are detectable by a light client are part of the misbehaviour protocol and can be used to minimize the damage caused by a compromised counterparty chain.

### Security model

It is important to note that IBC is not a completely trustless protocol; it is **trust-minimized**. This means that the safety property of bilateral IBC communication between two chains is dependent on the safety properties of the two chains in question. If one of the chains is compromised completely, then the IBC connection to the other chain is liable to receive invalid packets from the malicious chain. For example, if a malicious validator set has taken over more than 2/3 of the validator power on a chain; that malicious validator set can create a single chain of blocks with arbitrary commitment roots and arbitrary commitments to the next validator set. This would seize complete control of the chain and prevent the honest subset from even being able to create a competing honest block.

In this case, there is no ability for the IBC Tendermint client solely tracking CometBFT consensus to detect the misbehaviour and freeze the client. The IBC protocol would require out-of-band mechanisms to detect and fix such an egregious safety fault on the counterparty chain. Since the Tendermint light client is only tracking consensus and not also verifying the validity of state transitions, malicious behaviour from a validator set that is beyond the BFT fault threshold is an accepted risk of this light client implementation.

The IBC protocol has principles of fault isolation (e.g. all tokens are prefixed by their channel, so tokens from different chains are not mutually fungible) and fault mitigation (e.g. ability to freeze the client if misbehaviour can be detected before complete malicious takeover) that make this risk as minimal as possible.
