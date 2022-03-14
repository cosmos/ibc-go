---
ics: 11
title: Beefy Client
stage: draft
category: IBC/TAO
kind: instantiation
implements: 2
author: Seun Lanlege <seun@composable.finance>
created: 2022-03-08
---

## Synopsis

This specification document describes a client (verification algorithm) for a parachain using Beefy finality gadget

### Motivation

Parachains which get their finality from the relay chain (either polkadot/kusama in this case) might like to interface with other replicated state machines or solo machines over IBC.

### Definitions

Functions & terms are as defined in [ICS 2](../../core/ics-002-client-semantics).

`currentTimestamp` is as defined in [ICS 24](../../core/ics-024-host-requirements).

The Beefy light client uses a custom merkle proof format as described by [paritytech/trie](https://github.com/paritytech/trie)

`hash` is a generic collision-resistant hash function, and can easily be configured.

### Desired Properties

This specification must satisfy the client interface defined in ICS 2.


## Technical Specification


### Client state

The Beefy client state tracks the mmr root hash & height of the latest block, current validator set, next validator set, and a possible frozen height.

```golang
type ClientState struct {
	// Latest mmr root hash
	MmrRootHash []byte 
	// block number for the latest mmr_root_hash
	LatestBeefyHeight uint32
	// Block height when the client was frozen due to a misbehaviour
	FrozenHeight uint64 
	// block number that the beefy protocol was activated on the relay chain.
	// This shoould be the first block in the merkle-mountain-range tree.
	BeefyActivationBlock uint32 
	// authorities for the current round
	Authority *BeefyAuthoritySet 
	// authorities for the next round
	NextAuthoritySet *BeefyAuthoritySet
}
```

### Consensus state

The Beefy client tracks the timestamp (block time), actual parachain header & commitment root for all Ibc packets committed at this height

```golang
// ConsensusEngineID is a 4-byte identifier 
type ConsensusEngineID [4]byte

type Consensus struct {
    ConsensusEngineID ConsensusEngineID
    Bytes             []byte
}

type PreRuntime struct {
    ConsensusEngineID ConsensusEngineID
    Bytes             []byte
}
// DigestItem specifies the item in the logs of a digest
type DigestItem struct {
    IsChangesTrieRoot   bool // 2
    AsChangesTrieRoot   Hash
    IsPreRuntime        bool // 6
    AsPreRuntime        PreRuntime
    IsConsensus         bool // 4
    AsConsensus         Consensus
    IsSeal              bool // 5
    AsSeal              Seal
    IsChangesTrieSignal bool // 7
    AsChangesTrieSignal ChangesTrieSignal
    IsOther             bool // 0
    AsOther             []byte
}

type Digest []DigestItem

type ParachainHeader struct {
	// hash of the parent block
    ParentHash     [32]byte    
	// current block number/height
    Number         uint32 
	// merkle root hash of state trie
    StateRoot      [32]byte   
	// merkle root hash of all extrisincs in the block
    ExtrinsicsRoot [32]byte 
	// consensus related metadata (aka Consensus Proofs)
    Digest         Digest
}
// ConsensusState defines the consensus state from Tendermint.
type ConsensusState struct {
    // timestamp that corresponds to the block height in which the ConsensusState
    // was stored.
    Timestamp time.Time
    // parachain header
    ParachainHeader ParachainHeader
}
```

### Headers

The Beefy client headers include the height, the timestamp, the commitment root, the complete validator set, and the signatures by the validators who committed the block.

```golang
// Beefy Authority Info
type BeefyAuthoritySet struct {
    // Id of the authority set, it should be strictly increasing
    Id uint64 
    // size of the authority set
    Len uint32 
    // merkle root of the sorted authority public keys.
    AuthorityRoot *[32]byte 
}

// Partial data for MmrLeaf
type BeefyMmrLeafPartial struct {
    // leaf version
    Version uint8 
    // parent block for this leaf
    ParentNumber uint32 
    // parent hash for this leaf
    ParentHash *[32]byte 
    // next authority set.
    BeefyNextAuthoritySet BeefyAuthoritySet 
}


// data needed to prove parachain header inclusion in mmr.
type ParachainHeader struct {
    // scale-encoded parachain header bytes
    ParachainHeader []byte 
    // reconstructed MmrLeaf, see beefy-go spec
    MmrLeafPartial *BeefyMmrLeafPartial 
    // para_id of the header.
    ParaId uint32 
    // proofs for our header in the parachain heads root
    ParachainHeadsProof [][]byte 
    // leaf index for parachain heads proof
    HeadsLeafIndex uint32 
    // total number of para heads in parachain_heads_root
    HeadsTotalCount uint32
    // trie merkle proof of pallet_timestamp::Call::set() inclusion in header.extrinsic_root
    // this already encodes the actual extrinsic
    ExtrinsicProof [][]byte
}

type BeefyMmrLeaf struct {
    // leaf version
    Version uint8 
    // parent block for this leaf
    ParentNumber uint32 
    // parent hash for this leaf
    ParentHash *[32]byte 
    // beefy next authority set.
    BeefyNextAuthoritySet BeefyAuthoritySet 
    // merkle root hash of parachain heads included in the leaf.
    ParachainHeads *[32]byte 
}

// Actual payload items
type PayloadItem struct {
    // 2-byte payload id
    PayloadId *[2]byte
    // arbitrary length payload data., eg mmr_root_hash
    PayloadData []byte 
}

// Commitment message signed by beefy validators
type Commitment struct {
    // array of payload items signed by Beefy validators
    Payload []*PayloadItem
    // block number for this commitment
    BlockNumer uint32
    // validator set that signed this commitment
    ValidatorSetId uint64
}

// Signature belonging to a single validator
type CommitmentSignature struct {
    // actual signature bytes
    Signature [65]byte 
    // authority leaf index in the merkle tree.
    AuthorityIndex uint32 
}

// signed commitment data
type SignedCommitment struct {
    // commitment data being signed
    Commitment *Commitment
    // gotten from rpc subscription
    Signatures []*CommitmentSignature 
}

type MmrUpdateProof struct {
    // the new mmr leaf SCALE encoded.
    MmrLeaf *BeefyMmrLeaf 
    // leaf index for the mmr_leaf
    MmrLeafIndex uint64 
    // proof that this mmr_leaf index is valid.
    MmrProof [][]byte 
    // signed commitment data
    SignedCommitment *SignedCommitment 
    // generated using full authority list from runtime
    AuthoritiesProof [][]byte 
}

// Header contains the neccessary data to proove finality about IBC commitments
type Header struct {
    // parachain headers needed for proofs and ConsensusState
    ParachainHeaders []*ParachainHeader
    // mmr proofs for the headers
    MmrProofs [][]byte
    // size of the mmr for the given proof
    MmrSize uint64 
    // payload to update the mmr root hash.
    MmrUpdateProof MmrUpdateProof 
}
```

### Client initialisation

Beefy client initialisation requires a (subjectively chosen) latest beefy height, latest mmr root hash, cureent validator set & the next validator set.

```golang
func Initialise(MmrRootHash []byte, LatestBeefyHeight uint32,FrozenHeight uint64, BeefyActivationBlock uint32, Authority *BeefyAuthoritySet,NextAuthoritySet *BeefyAuthoritySet) ClientState {
    return ClientState {
        MmrRootHash: MmrRootHash,
        LatestBeefyHeight: LatestBeefyHeight,
        FrozenHeight: FrozenHeight,
        BeefyActivationBlock: BeefyActivationBlock,
        Authority: Authority,
        NextAuthoritySet: NextAuthoritySet,
    }
}
```

The Beefy client `latestClientHeight` function returns the latest stored height, which is updated every time a new (more recent) header is validated.

```golang
func (cs *ClientState) latestClientHeight() Height {
  return cs.LatestBeefyHeight
}
```

### Validity predicate

Beefy client validity checking happens in two stages, first we check the signatures of the `Commitment`, and use the recovered public keys to reconstruct an authority merkle root,
If this merkle root matches the light client's authority merkle root, we update the `LatestBeefyHeight` and `MmrRootHash` on the client. Optionally rotating our view of the next authority set if the authority set id is higher.

Next in order to verify if some given parachain headers have been finalized by the Beefy protocol, we attempt to reconstruct each `MmrLeaf` for every parachain header.
This is by reconstructing the `ParachainHeads` field - which contains the merkle root of all parachain headers that have been finalized by the relay chain at this leaf height.
This value, along with the `MmrLeafPartial` will be used to reconstruct the `MmrLeaf`. Then using the [ComposableFi/go-merkle-trees/mmr](https://github.com/composableFi/go-merkle-trees) we can reconstruct the mmr root hash
and compare this with what the light client percieves to the latest root hash. If there's a match, the verified headers are persisted as consensus states to the store.

```typescript
function checkValidityAndUpdateState(
  clientState: ClientState,
  revision: uint64,
  header: Header
) {
    unimplemented()
}
```

### Upgrades

The chain which this light client is tracking can elect to write a special pre-determined key in state to allow the light client to update its client state (e.g. with a new chain ID or revision) in preparation for an upgrade.

As the client state change will be performed immediately, once the new client state information is written to the predetermined key, the client will no longer be able to follow blocks on the old chain, so it must upgrade promptly.

```typescript
function upgradeClientState(
  clientState: ClientState,
  newClientState: ClientState,
  height: Height,
  proof: CommitmentPrefix) {
    // check proof of updated client state in state at predetermined commitment prefix and key
    path = applyPrefix(clientState.upgradeCommitmentPrefix, clientState.upgradeKey)
    // check that the client is at a sufficient height
    assert(clientState.latestHeight >= height)
    // check that the client is unfrozen or frozen at a higher height
    assert(clientState.frozenHeight === null || clientState.frozenHeight > height)
    // fetch the previously verified commitment root & verify membership
    root = get("clients/{identifier}/consensusStates/{height}")
    // verify that the provided consensus state has been stored
    assert(root.verifyMembership(path, newClientState, proof))
    // update client state
    clientState = newClientState
    set("clients/{identifier}", clientState)
}
```

### State verification functions

The Beefy client state verification functions check a Merkle proof against a previously validated commitment root.

```typescript
function verifyClientConsensusState(
  clientState: ClientState,
  height: Height,
  prefix: CommitmentPrefix,
  proof: CommitmentProof,
  clientIdentifier: Identifier,
  consensusStateHeight: Height,
  consensusState: ConsensusState) {
    path = applyPrefix(prefix, "clients/{clientIdentifier}/consensusState/{consensusStateHeight}")
    // check that the client is at a sufficient height
    assert(clientState.latestHeight >= height)
    // check that the client is unfrozen or frozen at a higher height
    assert(clientState.frozenHeight === null || clientState.frozenHeight > height)
    // fetch the previously verified commitment root & verify membership
    root = get("clients/{identifier}/consensusStates/{height}")
    // load proof items into paritytech/trie
    trie = trie.NewEmptyTrie().LoadFromProof(proof, root)
    // verify that the provided consensus state has been stored
    assert(trie.Get(path) === consensusState)
}

function verifyConnectionState(
  clientState: ClientState,
  height: Height,
  prefix: CommitmentPrefix,
  proof: CommitmentProof,
  connectionIdentifier: Identifier,
  connectionEnd: ConnectionEnd) {
    path = applyPrefix(prefix, "connections/{connectionIdentifier}")
    // check that the client is at a sufficient height
    assert(clientState.latestHeight >= height)
    // check that the client is unfrozen or frozen at a higher height
    assert(clientState.frozenHeight === null || clientState.frozenHeight > height)
    // fetch the previously verified commitment root & verify membership
    root = get("clients/{identifier}/consensusStates/{height}")
    // load proof items into paritytech/trie
    trie = trie.NewEmptyTrie().LoadFromProof(proof, root)
    // verify that the provided connection end has been stored
    assert(trie.Get(path) === connectionEnd)
}

function verifyChannelState(
  clientState: ClientState,
  height: Height,
  prefix: CommitmentPrefix,
  proof: CommitmentProof,
  portIdentifier: Identifier,
  channelIdentifier: Identifier,
  channelEnd: ChannelEnd) {
    path = applyPrefix(prefix, "ports/{portIdentifier}/channels/{channelIdentifier}")
    // check that the client is at a sufficient height
    assert(clientState.latestHeight >= height)
    // check that the client is unfrozen or frozen at a higher height
    assert(clientState.frozenHeight === null || clientState.frozenHeight > height)
    // fetch the previously verified commitment root & verify membership
    root = get("clients/{identifier}/consensusStates/{height}")
    // load proof items into paritytech/trie
    trie = trie.NewEmptyTrie().LoadFromProof(proof, root)
    // verify that the provided channel end has been stored
    assert(trie.Get(path) === channelEnd)
}

function verifyPacketData(
  clientState: ClientState,
  height: Height,
  delayPeriodTime: uint64,
  delayPeriodBlocks: uint64,
  prefix: CommitmentPrefix,
  proof: CommitmentProof,
  portIdentifier: Identifier,
  channelIdentifier: Identifier,
  sequence: uint64,
  data: bytes) {
    path = applyPrefix(prefix, "ports/{portIdentifier}/channels/{channelIdentifier}/packets/{sequence}")
    // check that the client is at a sufficient height
    assert(clientState.latestHeight >= height)
    // check that the client is unfrozen or frozen at a higher height
    assert(clientState.frozenHeight === null || clientState.frozenHeight > height)
    // fetch the processed time
    processedTime = get("clients/{identifier}/processedTimes/{height}")
    // fetch the processed height
    processedHeight = get("clients/{identifier}/processedHeights/{height}")
    // assert that enough time has elapsed
    assert(currentTimestamp() >= processedTime + delayPeriodTime)
    // assert that enough blocks have elapsed
    assert(currentHeight() >= processedHeight + delayPeriodBlocks)
    // fetch the previously verified commitment root & verify membership
    root = get("clients/{identifier}/consensusStates/{height}")
    // load proof items into paritytech/trie
    trie = trie.NewEmptyTrie().LoadFromProof(proof, root)
    // verify that the provided commitment has been stored
    assert(trie.Get(path) === data)
}

function verifyPacketAcknowledgement(
  clientState: ClientState,
  height: Height,
  delayPeriodTime: uint64,
  delayPeriodBlocks: uint64,
  prefix: CommitmentPrefix,
  proof: CommitmentProof,
  portIdentifier: Identifier,
  channelIdentifier: Identifier,
  sequence: uint64,
  acknowledgement: bytes) {
    path = applyPrefix(prefix, "ports/{portIdentifier}/channels/{channelIdentifier}/acknowledgements/{sequence}")
    // check that the client is at a sufficient height
    assert(clientState.latestHeight >= height)
    // check that the client is unfrozen or frozen at a higher height
    assert(clientState.frozenHeight === null || clientState.frozenHeight > height)
    // fetch the processed time
    processedTime = get("clients/{identifier}/processedTimes/{height}")
    // fetch the processed height
    processedHeight = get("clients/{identifier}/processedHeights/{height}")
    // assert that enough time has elapsed
    assert(currentTimestamp() >= processedTime + delayPeriodTime)
    // assert that enough blocks have elapsed
    assert(currentHeight() >= processedHeight + delayPeriodBlocks)
    // fetch the previously verified commitment root & verify membership
    root = get("clients/{identifier}/consensusStates/{height}")
    // load proof items into paritytech/trie
    trie = trie.NewEmptyTrie().LoadFromProof(proof, root)
    // verify that the provided acknowledgement has been stored
    assert(trie.Get(path) === hash(acknowledgement))
}

function verifyPacketReceiptAbsence(
  clientState: ClientState,
  height: Height,
  delayPeriodTime: uint64,
  delayPeriodBlocks: uint64,
  prefix: CommitmentPrefix,
  proof: CommitmentProof,
  portIdentifier: Identifier,
  channelIdentifier: Identifier,
  sequence: uint64) {
    path = applyPrefix(prefix, "ports/{portIdentifier}/channels/{channelIdentifier}/receipts/{sequence}")
    // check that the client is at a sufficient height
    assert(clientState.latestHeight >= height)
    // check that the client is unfrozen or frozen at a higher height
    assert(clientState.frozenHeight === null || clientState.frozenHeight > height)
    // fetch the processed time
    processedTime = get("clients/{identifier}/processedTimes/{height}")
    // fetch the processed height
    processedHeight = get("clients/{identifier}/processedHeights/{height}")
    // assert that enough time has elapsed
    assert(currentTimestamp() >= processedTime + delayPeriodTime)
    // assert that enough blocks have elapsed
    assert(currentHeight() >= processedHeight + delayPeriodBlocks)
    // fetch the previously verified commitment root & verify membership
    root = get("clients/{identifier}/consensusStates/{height}")
    // load proof items into paritytech/trie
    trie = trie.NewEmptyTrie().LoadFromProof(proof, root)
    // verify that no acknowledgement has been stored
    assert(trie.Get(path) === nill)
}

function verifyNextSequenceRecv(
  clientState: ClientState,
  height: Height,
  delayPeriodTime: uint64,
  delayPeriodBlocks: uint64,
  prefix: CommitmentPrefix,
  proof: CommitmentProof,
  portIdentifier: Identifier,
  channelIdentifier: Identifier,
  nextSequenceRecv: uint64) {
    path = applyPrefix(prefix, "ports/{portIdentifier}/channels/{channelIdentifier}/nextSequenceRecv")
    // check that the client is at a sufficient height
    assert(clientState.latestHeight >= height)
    // check that the client is unfrozen or frozen at a higher height
    assert(clientState.frozenHeight === null || clientState.frozenHeight > height)
    // fetch the processed time
    processedTime = get("clients/{identifier}/processedTimes/{height}")
    // fetch the processed height
    processedHeight = get("clients/{identifier}/processedHeights/{height}")
    // assert that enough time has elapsed
    assert(currentTimestamp() >= processedTime + delayPeriodTime)
    // assert that enough blocks have elapsed
    assert(currentHeight() >= processedHeight + delayPeriodBlocks)
    // fetch the previously verified commitment root & verify membership
    root = get("clients/{identifier}/consensusStates/{height}")
    // load proof items into paritytech/trie
    trie = trie.NewEmptyTrie().LoadFromProof(proof, root)
    // verify that the nextSequenceRecv is as claimed
    assert(trie.Get(path) === nextSequenceRecv)
}
```

### Properties & Invariants

Correctness guarantees as provided by the Beefy light client protocol.

## Backwards Compatibility

Not applicable.

## Forwards Compatibility

Not applicable. Alterations to the client verification algorithm will require a new client standard.

## Example Implementation

None yet.

## Other Implementations

None at present.

## History

March 14th, 2022 - Initial version

## Copyright

All content herein is licensed under [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0).
