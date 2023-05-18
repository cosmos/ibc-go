package multihop

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
)

// Endpoint represents a Cosmos chain endpoint for queries.
// Endpoint is stateless from caller's perspective.
type Endpoint interface {
	ChainID() string
	Codec() codec.BinaryCodec
	ClientID() string
	GetClientState() exported.ClientState
	GetConsensusState(height exported.Height) (exported.ConsensusState, error)
	ConnectionID() string
	GetConnection() (*connectiontypes.ConnectionEnd, error)
	// Returns the proof of the `key`` at `height` within the ibc module store.
	QueryProofAtHeight(key []byte, height int64) ([]byte, clienttypes.Height, error)
	QueryStateAtHeight(key []byte, height int64) []byte

	// QueryMinimumConsensusHeight returns the minimum height within the provided range at which the consensusState exists (processedHeight)
	// and the height of the corresponding consensus state (consensusHeight).
	QueryMinimumConsensusHeight(minHeight exported.Height, maxHeight exported.Height) (exported.Height, exported.Height, error)
	// QueryMaximumProofHeight returns the maxmimum height which can be used to prove a key/val pair by search consecutive heights
	// to find the first point at which the value changes for the given key.
	QueryMaximumProofHeight(key []byte, minKeyHeight exported.Height, maxKeyHeightLimit exported.Height) exported.Height
	GetMerklePath(path string) (commitmenttypes.MerklePath, error)
	Counterparty() Endpoint
}

// Path contains two endpoints of chains that have a direct IBC connection, ie. a single-hop IBC path.
type Path struct {
	EndpointA Endpoint
	EndpointB Endpoint
}

// ChanPath represents a multihop channel path that spans 2 or more single-hop `Path`s.
type ChanPath []*Path

// NewChanPath creates a new multi-hop ChanPath from a list of single-hop Paths.
func NewChanPath(paths []*Path) ChanPath {
	if len(paths) < 2 {
		panic(fmt.Sprintf("multihop channel path expects at least 2 single-hop paths, but got %d", len(paths)))
	}
	return ChanPath(paths)
}

// GetConnectionHops returns the connection hops for the multihop channel.
func (p ChanPath) GetConnectionHops() []string {
	hops := make([]string, len(p))
	for i, path := range p {
		hops[i] = path.EndpointA.ConnectionID()
	}
	return hops
}

// GenerateProof generates a proof for the given key on the the source chain, which is to be verified on the dest
// chain.
func (p ChanPath) GenerateProof(
	key []byte,
	val []byte,
	proofHeight exported.Height,
	doVerify bool,
) (result *channeltypes.MsgMultihopProofs, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = sdkerrors.Wrapf(channeltypes.ErrMultihopProofGeneration, "%v", r)
		}
	}()

	result = &channeltypes.MsgMultihopProofs{}

	// generate proof for key on source chain at the minimum consensus height known on the counterparty chain
	maxProofHeight := p.source().Counterparty().QueryMaximumProofHeight(key, proofHeight, nil)
	_, consensusHeightAB, err := p.source().Counterparty().QueryMinimumConsensusHeight(proofHeight, maxProofHeight)
	if err != nil {
		return nil, err
	}

	result.KeyProof = queryProof(p.source(), key, val, consensusHeightAB, nil, doVerify)

	proofGenFuncs := []proofGenFunc{
		genConsensusStateProof,
		genConnProof,
	}

	// create a maximum height for the proof to be verified against
	linkedPathProofs, err := p.GenerateIntermediateStateProofs(proofGenFuncs, proofHeight)
	if err != nil {
		return nil, err
	}
	if len(linkedPathProofs) != len(proofGenFuncs) {
		return nil, sdkerrors.Wrapf(
			channeltypes.ErrMultihopProofGeneration,
			"expected %d linked path proofs for consensus, connections, and client states but got %d",
			len(proofGenFuncs), len(linkedPathProofs),
		)
	}
	result.ConsensusProofs = linkedPathProofs[0]
	result.ConnectionProofs = linkedPathProofs[1]

	return result, nil
}

// The source chain
func (p ChanPath) source() Endpoint {
	return p[0].EndpointA
}

// GenerateIntermediateStateProofs generates lists of connection, consensus, and client state proofs from the source to dest chains.
func (p ChanPath) GenerateIntermediateStateProofs(
	proofGenFuncs []proofGenFunc,
	proofHeight exported.Height,
) (result [][]*channeltypes.MultihopProof, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = sdkerrors.Wrapf(channeltypes.ErrMultihopProofGeneration, "%v", r)
		}
	}()
	// initialize a 2-d slice of proofs, where 1st dim is the proof gen funcs, and 2nd dim is the path iter count
	result = make([][]*channeltypes.MultihopProof, len(proofGenFuncs))

	// iterate over all but last single-hop path
	iterCount := len(p) - 1
	for i := 0; i < iterCount; i++ {
		// Given 3 chains connected by 2 paths:
		// A -(path)-> B -(nextPath)-> C
		// , We need to generate proofs for chain A's key paths. The proof is verified with B's consensus state on C.
		// ie. proof to verify A's state on C.
		// The loop starts with the source chain as A, and ends with the dest chain as chain C.
		// NOTE: chain {A,B,C} are relatively referenced to the current iteration, not to be confused with the chainID
		// or endpointA/B.

		chainB, chainC := p[i].EndpointB, p[i+1].EndpointB

		// find minimum height on chainB that can prove the key/value at a specific height on chainA
		proofHeightAB, consensusHeightAB, err := chainB.QueryMinimumConsensusHeight(proofHeight, nil)
		panicIfErr(err, "failed to query minimum proof height")

		// find minimum height on chainC that can prove the consensusState at the proof height on chainB
		// this is done only for verifying proofs inline
		proofHeightBC, consensusHeightBC, err := chainC.QueryMinimumConsensusHeight(proofHeightAB, nil)
		panicIfErr(err, "failed to query minimum proof height for verification")

		// query the consensusState on chainC to use for proof checking
		key := host.FullConsensusStateKey(chainC.ClientID(), consensusHeightBC)
		bzConsStateBC := chainC.QueryStateAtHeight(key, int64(proofHeightBC.GetRevisionHeight()))
		var consState exported.ConsensusState
		err = chainC.Codec().UnmarshalInterface(bzConsStateBC, &consState)
		panicIfErr(err, "fail to unmarshal consensus state of chain '%s' on chain '%s' at height %s due to: %v", chainC.Counterparty().ChainID(), chainC.ChainID(), proofHeightBC, err)
		consStateBC, ok := consState.(*tmclient.ConsensusState)
		if !ok {
			panic(fmt.Sprintf("expected consensus state to be tendermint consensus state, got: %T", consStateBC))
		}
		rootBC := consStateBC.GetRoot()

		for j, proofGenFunc := range proofGenFuncs {
			proof := proofGenFunc(chainB, proofHeightAB.Increment(), consensusHeightAB, rootBC)
			result[j] = append([]*channeltypes.MultihopProof{proof}, result[j]...)
		}

		// prepare for next iteration
		proofHeight = proofHeightAB
	}

	return result, nil
}

type proofGenFunc func(Endpoint, exported.Height, exported.Height, exported.Root) *channeltypes.MultihopProof

// Generate a proof for A's consensusState stored on B using B's consensusState root stored on C.
func genConsensusStateProof(
	chainB Endpoint,
	processedHeight exported.Height,
	consensusHeight exported.Height,
	consStateBCRoot exported.Root,
) *channeltypes.MultihopProof {

	key := host.FullConsensusStateKey(chainB.ClientID(), consensusHeight)
	bzConsStateAB := chainB.QueryStateAtHeight(key, int64(processedHeight.GetRevisionHeight()))

	return queryProof(
		chainB,
		host.FullConsensusStateKey(chainB.ClientID(), consensusHeight),
		bzConsStateAB,
		processedHeight,
		consStateBCRoot,
		true,
	)
}

// Generate a proof for the connEnd denoting A stored on B using B's consensusState root stored on C.
func genConnProof(
	chainB Endpoint,
	processedHeight exported.Height,
	_ exported.Height,
	consStateBCRoot exported.Root,
) *channeltypes.MultihopProof {
	key := host.ConnectionKey(chainB.ConnectionID())
	bzConnAB := chainB.QueryStateAtHeight(key, int64(processedHeight.GetRevisionHeight()))
	return queryProof(chainB, host.ConnectionKey(chainB.ConnectionID()), bzConnAB, processedHeight, consStateBCRoot, true)
}

// queryProof queries the key-value pair or absence proof stored on A and optionally ensures the proof
// can be verified by A's consensus state root stored on B at heightAB. where A--B is connected by a
// single ibc connection.
//
// if doVerify is false, skip verification.
// If value is nil, do non-membership verification.
// If heightAB is nil, use the latest height of B's client state.
// If consStateABRoot is nil, use the root of the consensus state of clientAB at heightAB.
//
// Panic if proof generation or verification fails.
func queryProof(
	chainA Endpoint,
	key, value []byte,
	heightAB exported.Height,
	consStateABRoot exported.Root,
	doVerify bool,
) *channeltypes.MultihopProof {
	if len(key) == 0 {
		panic("key must be non-empty")
	}

	if heightAB == nil {
		panic("height must be non-nil")
	}

	chainB := chainA.Counterparty()

	keyMerklePath, err := chainB.GetMerklePath(string(key))
	panicIfErr(err, "fail to create merkle path on chain '%s' with path '%s' due to: %v",
		chainB.ChainID(), key, err,
	)

	bzProof, _, err := chainA.QueryProofAtHeight(key, int64(heightAB.GetRevisionHeight()))
	panicIfErr(err, "fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
		chainA.ChainID(), key, heightAB, err,
	)

	if doVerify {

		var proof commitmenttypes.MerkleProof
		err = chainA.Codec().Unmarshal(bzProof, &proof)
		panicIfErr(err, "fail to unmarshal chain [%s]'s proof on chain [%s] due to: %v",
			chainA.ChainID(), chainB.ChainID(), err,
		)
		if len(value) > 0 {
			// ensure key-value pair can be verified by consStateBC
			err = proof.VerifyMembership(
				commitmenttypes.GetSDKSpecs(), consStateABRoot,
				keyMerklePath, value,
			)
		} else {
			err = proof.VerifyNonMembership(
				commitmenttypes.GetSDKSpecs(), consStateABRoot,
				keyMerklePath,
			)
		}

		panicIfErr(
			err,
			"fail to verify proof chain [%s]'s key path '%s' at height %s due to: %v",
			chainA.ChainID(), key, heightAB, err,
		)
	}

	return &channeltypes.MultihopProof{
		Proof:       bzProof,
		Value:       value,
		PrefixedKey: &keyMerklePath,
	}
}

func panicIfErr(err error, format string, args ...interface{}) {
	if err != nil {
		panic(fmt.Sprintf(format, args...))
	}
}
