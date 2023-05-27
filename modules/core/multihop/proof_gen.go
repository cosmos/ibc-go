package multihop

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
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
	// Returns the value of the `key`` at `height` within the ibc module store and optionally the proof
	QueryStateAtHeight(key []byte, height int64, doProof bool) ([]byte, []byte, error)

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

// The source chain
func (p ChanPath) source() Endpoint {
	return p[0].EndpointA
}

// QueryMultihopProof returns a multi-hop proof for the given key on the the source chain, which is to be verified on the dest chain.
func (p ChanPath) QueryMultihopProof(key []byte, proofHeight exported.Height) (multihopProof channeltypes.MsgMultihopProofs, err error) {

	if len(p) < 2 {
		err = fmt.Errorf("multihop proof query requires channel path length >= 2")
		return
	}

	// query the maximum valid height for the key which is the first height at which its value changes
	maxProofHeight := p.source().QueryMaximumProofHeight(key, proofHeight, nil)

	// query the minimum height to prove the key on the source chain
	proofHeight, consensusHeight, err := p.source().Counterparty().QueryMinimumConsensusHeight(proofHeight, maxProofHeight)
	if err != nil {
		return
	}

	// TODO: why does this need to decrement?
	keyProofHeight, ok := consensusHeight.Decrement()
	if !ok {
		err = fmt.Errorf("failed to decrement height: %v\n", consensusHeight)
		return
	}

	// query the proof of the key/value on the source chain at a height provable on the next chain.
	if multihopProof.KeyProof, err = queryProof(p.source(), key, keyProofHeight, false); err != nil {
		return
	}

	// query the consensus state proof on the counterparty chain
	multihopProof.ConsensusProofs = make([]*channeltypes.MultihopProof, len(p)-1)
	if multihopProof.ConsensusProofs[len(p)-2], err = queryConsensusStateProof(p.source().Counterparty(), proofHeight, consensusHeight); err != nil {
		return
	}

	// query the connection proof on the counterparty chain
	multihopProof.ConnectionProofs = make([]*channeltypes.MultihopProof, len(p)-1)
	if multihopProof.ConnectionProofs[len(p)-2], err = queryConnectionProof(p.source().Counterparty(), proofHeight); err != nil {
		return
	}

	// query proofs of consensus/connection states on intermediate chains
	if err = p.queryIntermediateProofs(1, proofHeight, multihopProof.ConsensusProofs, multihopProof.ConnectionProofs); err != nil {
		return
	}

	return
}

// queryIntermediateProofs recursively queries intermediate chains in a multi-hop channel path for consensus state
// and connection proofs. It stops at the second to last path since the consensus and connection state on the
// final hop is already known on the destination.
func (p ChanPath) queryIntermediateProofs(
	pathIdx int,
	proofHeight exported.Height,
	consensusProofs []*channeltypes.MultihopProof,
	connectionProofs []*channeltypes.MultihopProof,
) (err error) {

	chain := p[pathIdx].EndpointB

	var consensusHeight exported.Height

	// determine the minimum consensusState height on the next chain within the provided height range
	// also returns the height at which this consensusState was processed on the nextChain which will
	// be used to query a proof of the consensus state at the minimum possible height
	if proofHeight, consensusHeight, err = chain.QueryMinimumConsensusHeight(proofHeight, nil); err != nil {
		return
	}

	if consensusProofs[len(p)-pathIdx-2], err = queryConsensusStateProof(chain, proofHeight, consensusHeight); err != nil {
		return
	}

	if connectionProofs[len(p)-pathIdx-2], err = queryConnectionProof(chain, proofHeight); err != nil {
		return
	}

	// no need to query min consensus height on final chain
	if pathIdx == len(p)-2 {
		return
	}

	return p.queryIntermediateProofs(pathIdx+1, proofHeight, consensusProofs, connectionProofs)
}

// queryConsensusStateProof queries a chain for a proof at `proofHeight` for a consensus state at `consensusHeight`
func queryConsensusStateProof(
	chain Endpoint,
	proofHeight exported.Height,
	consensusHeight exported.Height,
) (*channeltypes.MultihopProof, error) {
	key := host.FullConsensusStateKey(chain.ClientID(), consensusHeight)
	return queryProof(chain, key, proofHeight, true)
}

// queryConnectionProof queries a chain for a proof at `proofHeight` for a connection
func queryConnectionProof(
	chain Endpoint,
	proofHeight exported.Height,
) (*channeltypes.MultihopProof, error) {
	key := host.ConnectionKey(chain.ConnectionID())
	return queryProof(chain, key, proofHeight, true)
}

// queryProof queries a (non-)membership proof for the key on the specified chain and
// returns the proof
//
// if doValue, the queried value is added to the proof this is required for
// intermediate consensus/connection multihop proofs
func queryProof(
	chain Endpoint,
	key []byte,
	height exported.Height,
	doValue bool,
) (*channeltypes.MultihopProof, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("key must be non-empty")
	}

	if height == nil {
		return nil, fmt.Errorf("height must be non-nil")
	}

	keyMerklePath, err := chain.GetMerklePath(string(key))
	if err != nil {
		return nil, fmt.Errorf("fail to create merkle path on chain '%s' with path '%s' due to: %v",
			chain.ChainID(), key, err)
	}

	valueBytes, proof, err := chain.QueryStateAtHeight(key, int64(height.GetRevisionHeight()), true)
	if err != nil {
		return nil, fmt.Errorf("fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
			chain.ChainID(), key, height, err,
		)
	}

	if !doValue {
		valueBytes = nil
	}

	return &channeltypes.MultihopProof{
		Proof:       proof,
		Value:       valueBytes,
		PrefixedKey: &keyMerklePath,
	}, nil
}
