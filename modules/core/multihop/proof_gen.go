package multihop

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
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
	QueryMinimumConsensusHeight(minConsensusHeight exported.Height, maxConsensusHeight exported.Height) (exported.Height, exported.Height, error)
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

// GenerateProof generates a proof for the given key on the the source chain, which is to be verified on the dest chain.
func (p ChanPath) GenerateProof(key []byte, proofHeight exported.Height) (multihopProof channeltypes.MsgMultihopProofs, err error) {

	// generate proof for key on source chain at the minimum consensus height known on the counterparty chain
	maxProofHeight := p.source().QueryMaximumProofHeight(key, proofHeight, nil)
	processedHeight, consensusHeight, err := p.source().Counterparty().QueryMinimumConsensusHeight(proofHeight, maxProofHeight)
	if err != nil {
		return
	}

	// query the proof of the key/value on the source chain at a height provable on the next chain.
	multihopProof.KeyProof = queryProof(p.source(), key, nil, consensusHeight)

	proofGenFuncs := []proofGenFunc{
		genConsensusStateProof,
		genConnProof,
	}

	// create a maximum height for the proof to be verified against
	linkedPathProofs, err := p.GenerateIntermediateStateProofs(proofGenFuncs, processedHeight, consensusHeight)
	if err != nil {
		return
	}

	multihopProof.ConsensusProofs = linkedPathProofs[0]
	multihopProof.ConnectionProofs = linkedPathProofs[1]

	return
}

// The source chain
func (p ChanPath) source() Endpoint {
	return p[0].EndpointA
}

// GenerateIntermediateStateProofs generates lists of connection, consensus, and client state proofs from the source to dest chains.
func (p ChanPath) GenerateIntermediateStateProofs(
	proofGenFuncs []proofGenFunc,
	processedHeight exported.Height,
	consensusHeight exported.Height,
) (proofs [][]*channeltypes.MultihopProof, err error) {

	// initialize a 2-d slice of proofs, where 1st dim is the proof gen funcs, and 2nd dim is the path iter count
	proofs = make([][]*channeltypes.MultihopProof, len(proofGenFuncs))

	// iterate over all but last single-hop path
	iterCount := len(p) - 1
	for i := 0; i < iterCount; i++ {
		// 1. Query the prior chain consensus/connection state proofs on the i'th chain
		// 2. Prepare the next proof round. the processed height indicates the minimum height
		//    which can prove the desired consensusState at a specific height.
		// 3. The processed height is used to query the minimum consensus height on the nextChain

		chain, nextChain := p[i].EndpointB, p[i+1].EndpointB

		// query proof of consensus/connection state on next chain
		for j, proofGenFunc := range proofGenFuncs {
			proof := proofGenFunc(chain, processedHeight, consensusHeight)
			proofs[j] = append([]*channeltypes.MultihopProof{proof}, proofs[j]...)
		}

		// no need to query min consensus height on final chain
		if i == len(p)-2 {
			break
		}

		// find minimum height on nextChain that can prove the key/value at a specific height on the current chain
		processedHeight, consensusHeight, err = nextChain.QueryMinimumConsensusHeight(processedHeight, nil)
		if err != nil {
			return nil, err
		}
	}

	return proofs, nil
}

type proofGenFunc func(Endpoint, exported.Height, exported.Height) *channeltypes.MultihopProof

// Generate a proof for A's consensusState stored on B
func genConsensusStateProof(
	chainB Endpoint,
	processedHeight exported.Height,
	consensusHeight exported.Height,
) *channeltypes.MultihopProof {

	key := host.FullConsensusStateKey(chainB.ClientID(), consensusHeight)
	bzConsStateAB := chainB.QueryStateAtHeight(key, int64(processedHeight.GetRevisionHeight()))

	consensusProof := queryProof(
		chainB,
		host.FullConsensusStateKey(chainB.ClientID(), consensusHeight),
		bzConsStateAB,
		processedHeight.Increment(), // need to match height used in QueryStateAtHeight
	)

	// debug code
	// var proof commitmenttypes.MerkleProof
	// if err := chainB.Codec().Unmarshal(consensusProof.Proof, &proof); err != nil {
	// 	panicIfErr(err, "failed to unmarshal")
	// }
	// if proof.GetProofs()[0].GetExist() == nil {
	// 	panic("queried non-existence proof!")
	// }

	return consensusProof
}

// Generate a proof for the connEnd denoting A stored on B using B's consensusState root stored on C.
func genConnProof(
	chainB Endpoint,
	processedHeight exported.Height,
	_ exported.Height,
) *channeltypes.MultihopProof {
	key := host.ConnectionKey(chainB.ConnectionID())
	bzConnAB := chainB.QueryStateAtHeight(key, int64(processedHeight.GetRevisionHeight()))
	return queryProof(
		chainB,
		host.ConnectionKey(chainB.ConnectionID()),
		bzConnAB,
		processedHeight.Increment()) // need to match height used in QueryStateAtHeight
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
	chain Endpoint,
	key, value []byte,
	height exported.Height,
) *channeltypes.MultihopProof {
	if len(key) == 0 {
		panic("key must be non-empty")
	}

	if height == nil {
		panic("height must be non-nil")
	}

	counterpartyChain := chain.Counterparty()

	keyMerklePath, err := counterpartyChain.GetMerklePath(string(key))
	panicIfErr(err, "fail to create merkle path on chain '%s' with path '%s' due to: %v",
		counterpartyChain.ChainID(), key, err,
	)

	bzProof, _, err := chain.QueryProofAtHeight(key, int64(height.GetRevisionHeight()))
	panicIfErr(err, "fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
		chain.ChainID(), key, height, err,
	)

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
