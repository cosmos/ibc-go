package multihop

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
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
	GetMerklePath(path string) (commitmenttypes.MerklePath, error)
	// UpdateClient updates the clientState of counterparty chain's header
	UpdateClient() error
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

// UpdateClient updates the clientState{AB, BC, .. YZ} so chainA's consensusState is propogated to chainZ.
func (p ChanPath) UpdateClient() error {
	for _, path := range p {
		if err := path.EndpointB.UpdateClient(); err != nil {
			return err
		}
	}
	return nil
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
func (p ChanPath) GenerateProof(key []byte, val []byte, doVerify bool) (result *channeltypes.MsgMultihopProofs, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = sdkerrors.Wrapf(channeltypes.ErrMultihopProofGeneration, "%v", r)
		}
	}()

	result = &channeltypes.MsgMultihopProofs{}
	// generate proof for key on source chain
	result.KeyProof = queryProof(p.source(), key, val, nil, nil, doVerify)

	linkedPathProofs, err := p.GenerateConsensusAndConnectionProofs()
	if err != nil {
		return nil, err
	}
	if len(linkedPathProofs) != 2 {
		return nil, sdkerrors.Wrapf(
			channeltypes.ErrMultihopProofGeneration,
			"expected 2 linked path proofs for both consensus states and connections, but got %d",
			len(linkedPathProofs),
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

// GenerateConsensusAndConnectionProofs generates lists of membership proofs from the source to dest chains.
func (p ChanPath) GenerateConsensusAndConnectionProofs() (result [][]*channeltypes.MultihopProof, err error) {
	return p.GenerateProofsOnLinkedPaths([]proofGenFunc{
		genConsensusStateProof,
		genConnProof,
	})
}

// GenerateProofsOnLinkedPaths generates lists of membership proofs from the source to dest chains.
func (p ChanPath) GenerateProofsOnLinkedPaths(
	proofGenFuncs []proofGenFunc,
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
		heightAB := chainB.GetClientState().GetLatestHeight()
		heightBC := chainC.GetClientState().GetLatestHeight()
		consStateBC, err := chainC.GetConsensusState(heightBC)
		panicIfErr(err,
			"failed to get consensus state root of chain '%s' at height %s on chain '%s': %v",
			chainB.ChainID(), heightBC, chainC.ChainID(), err,
		)

		consStateBCRoot := consStateBC.GetRoot()

		for j, proofGenFunc := range proofGenFuncs {
			proof := proofGenFunc(chainB, heightAB, heightBC, consStateBCRoot)
			result[j] = append(result[j], proof)
		}
	}
	return result, nil
}

type proofGenFunc func(Endpoint, exported.Height, exported.Height, exported.Root) *channeltypes.MultihopProof

// Generate a proof for A's consensusState stored on B using B's consensusState root stored on C.
func genConsensusStateProof(
	chainB Endpoint,
	heightAB, heightBC exported.Height,
	consStateBCRoot exported.Root,
) *channeltypes.MultihopProof {
	chainAID := chainB.Counterparty().ChainID()
	consStateAB, err := chainB.GetConsensusState(heightAB)
	panicIfErr(err, "chain [%s]'s consensus state on chain '%s' at height %s not found due to: %v",
		chainAID, chainB.ChainID(), heightAB, err,
	)
	bzConsStateAB, err := chainB.Codec().MarshalInterface(consStateAB)
	panicIfErr(err, "fail to marshal consensus state of chain '%s' on chain '%s' at height %s due to: %v",
		chainAID, chainB.ChainID(), heightAB, err,
	)
	return queryProof(
		chainB,
		host.FullConsensusStateKey(chainB.ClientID(), heightAB),
		bzConsStateAB,
		heightBC,
		consStateBCRoot,
		true,
	)
}

// Generate a proof for the connEnd denoting A stored on B using B's consensusState root stored on C.
func genConnProof(
	chainB Endpoint,
	heightAB, heightBC exported.Height,
	consStateBCRoot exported.Root,
) *channeltypes.MultihopProof {
	connAB, err := chainB.GetConnection()
	panicIfErr(err, "fail to get connection '%s' on chain '%s' due to: %v",
		chainB.ConnectionID(), chainB.ChainID(), err,
	)
	bzConnAB, err := chainB.Codec().Marshal(connAB)
	panicIfErr(err, "fail to marshal connection '%s' on chain '%s' due to: %v",
		chainB.ConnectionID(), chainB.ChainID(), err,
	)
	return queryProof(chainB, host.ConnectionKey(chainB.ConnectionID()), bzConnAB, heightBC, consStateBCRoot, true)
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
		panic("key and value must be non-empty")
	}

	chainB := chainA.Counterparty()
	// set optional params if not passed in
	if heightAB == nil {
		heightAB = chainB.GetClientState().GetLatestHeight()
	}
	if consStateABRoot == nil {
		consState, err := chainB.GetConsensusState(heightAB)
		panicIfErr(err, "fail to get chain [%s]'s consensus state at height %s on chain '%s' due to: %v",
			chainA.ChainID(), heightAB, chainB.ChainID(), err,
		)
		consStateABRoot = consState.GetRoot()
	}

	keyMerklePath, err := chainB.GetMerklePath(string(key))
	panicIfErr(err, "fail to create merkle path on chain '%s' with path '%s' due to: %v",
		chainB.ChainID(), key, err,
	)

	bzProof, _, err := chainA.QueryProofAtHeight(key, int64(heightAB.GetRevisionHeight()))
	panicIfErr(err, "fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
		chainA.ChainID(), key, heightAB, err,
	)

	// only verify ke/value if value is not nil
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
