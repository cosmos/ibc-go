package multihop

import (
	"fmt"

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
	ClientID() string
	GetLatestHeight() exported.Height
	GetClientState() exported.ClientState
	GetConsensusState(height exported.Height) (exported.ConsensusState, bool)
	ConnectionID() string
	GetConnection() (*connectiontypes.ConnectionEnd, error)
	// Returns the value of the `key`` at `height` within the ibc module store and optionally the proof
	QueryStateAtHeight(key []byte, height int64, doProof bool) ([]byte, []byte, error)

	// QueryMinimumConsensusHeight returns the minimum height within the provided range at which the consensusState exists (processedHeight)
	// and the corresponding consensus state height (consensusHeight).
	QueryMinimumConsensusHeight(minConsensusHeight exported.Height, maxConsensusHeight exported.Height) (exported.Height, exported.Height, error)
	GetMerklePath(path string) (commitmenttypes.MerklePath, error)
	Counterparty() Endpoint
	UpdateClient() error
}

// Path contains two endpoints of chains that have a direct IBC connection, ie. a single-hop IBC path.
type Path struct {
	EndpointA Endpoint
	EndpointB Endpoint
}

// ProofHeights contains multi-hop proof data.
type ProofHeights struct {
	proofHeight     exported.Height // query the proof at this height
	consensusHeight exported.Height // the proof is for the consensusState at this height
}

// ChanPath represents a multihop channel path that spans 2 or more single-hop `Path`s.
type ChanPath struct {
	Paths []*Path
}

// NewChanPath creates a new multi-hop ChanPath from a list of single-hop Paths.
func NewChanPath(paths []*Path) ChanPath {
	if len(paths) < 1 {
		panic(fmt.Sprintf("multihop channel path expects at least 1 single-hop paths, but got %d", len(paths)))
	}
	return ChanPath{
		Paths: paths,
	}
}

// Counterparty returns the ChanPath as seen in the reverse direction.
func (p ChanPath) Counterparty() ChanPath {
	paths := make([]*Path, len(p.Paths))
	for i, hop := range p.Paths {
		reversedSinglePath := Path{EndpointA: hop.EndpointB, EndpointB: hop.EndpointA}
		paths[len(p.Paths)-i-1] = &reversedSinglePath
	}
	return NewChanPath(paths)
}

// GetConnectionHops returns the connection hops for the multihop channel.
func (p ChanPath) GetConnectionHops() []string {
	hops := make([]string, len(p.Paths))
	for i, path := range p.Paths {
		hops[i] = path.EndpointA.ConnectionID()
	}
	return hops
}

// Source returns the Source chain for the chan path
func (p ChanPath) Source() Endpoint {
	return p.Paths[0].EndpointA
}

// QueryMultihopProof returns a multi-hop proof for the given key on the the source
// chain along with the proofHeight, which is to be verified on the dest chain.
// The proofHeight is the consensus state height for the immediate counterparty on
// the receiving chain. This is the known/trusted consensus state which starts the
// multi-hop proof verification.
func (p ChanPath) QueryMultihopProof(
	key []byte,
	keyProofHeight exported.Height,
) (
	multihopProof channeltypes.MsgMultihopProofs,
	multihopProofHeight exported.Height,
	err error,
) {

	if len(p.Paths) < 1 {
		err = fmt.Errorf("multihop proof query requires channel path length >= 1")
		return
	}

	proofHeights, err := p.calcProofPath(0, keyProofHeight)
	if err != nil {
		return
	}

	// the consensus state height of the proving chain's counterparty
	// this is where multi-hop proof verification begins
	multihopProofHeight, _ = proofHeights[0].consensusHeight.Decrement()

	// the key/value proof height is the height of the consensusState on the first chain
	keyProofHeight, ok := proofHeights[len(proofHeights)-1].consensusHeight.Decrement()
	if !ok {
		err = fmt.Errorf("failed to decrement consensusHeight while setting key proof height")
		return
	}

	// query the proof of the key/value on the source chain
	if multihopProof.KeyProof, err = queryProof(p.Source(), key, keyProofHeight, false); err != nil {
		return
	}

	multihopProof.ConsensusProofs = make([]*channeltypes.MultihopProof, len(p.Paths)-1)
	multihopProof.ConnectionProofs = make([]*channeltypes.MultihopProof, len(p.Paths)-1)

	// query proofs of consensus/connection states on intermediate chains
	if err = p.Counterparty().queryIntermediateProofs(
		0, proofHeights[1:],
		multihopProof.ConsensusProofs,
		multihopProof.ConnectionProofs); err != nil {
		return
	}

	return
}

// calcProofPath calculates the optimal proof heights to generate a multi-hop proof along the channel path
// and performs client updates as needed.
func (p ChanPath) calcProofPath(pathIdx int, consensusHeight exported.Height) (heights []*ProofHeights, err error) {
	var height ProofHeights
	chain := p.Paths[pathIdx].EndpointB

	// find minimum consensus height provable on the next chain
	if height.proofHeight, height.consensusHeight, err = chain.QueryMinimumConsensusHeight(consensusHeight, nil); err != nil {
		return nil, err
	}

	// if no suitable consensusHeight then update client and use latest chain height/client height
	//
	// TODO: It would be even more efficient to update the client with the missing block height
	// rather than the latest block height since it would be less likely to need client updates
	// on subsequent chains.
	if height.proofHeight == nil {
		if err := chain.UpdateClient(); err != nil {
			return nil, err
		}

		height.proofHeight = chain.GetLatestHeight()
		height.consensusHeight = chain.GetClientState().GetLatestHeight()
	}

	// stop on the next to last path segment
	if pathIdx == len(p.Paths)-1 {
		return []*ProofHeights{&height}, nil
	}

	// use the proofHeight as the next consensus height
	if heights, err = p.calcProofPath(pathIdx+1, height.proofHeight); err != nil {
		return nil, err
	}

	heights = append(heights, &height)
	return
}

// queryIntermediateProofs recursively queries intermediate chains in a multi-hop channel path for consensus state
// and connection proofs. It stops at the second to last path since the consensus and connection state on the
// final hop is already known on the destination.
func (p ChanPath) queryIntermediateProofs(
	proofIdx int,
	proofHeights []*ProofHeights,
	consensusProofs []*channeltypes.MultihopProof,
	connectionProofs []*channeltypes.MultihopProof,
) (err error) {

	// no need to query proofs on final chain since the clientState is already known
	if proofIdx == len(p.Paths)-1 {
		return
	}

	chain := p.Paths[proofIdx+1].EndpointA
	mhp := proofHeights[proofIdx]

	// query proof of the consensusState
	if consensusProofs[proofIdx], err = queryConsensusStateProof(chain, mhp.proofHeight, mhp.consensusHeight); err != nil {
		return
	}

	// query proof of the connectionEnd
	if connectionProofs[proofIdx], err = queryConnectionProof(chain, mhp.proofHeight); err != nil {
		return
	}

	// continue querying proofs on the next chain in the path
	return p.queryIntermediateProofs(proofIdx+1, proofHeights, consensusProofs, connectionProofs)
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
	includeKey bool,
) (*channeltypes.MultihopProof, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("key must be non-empty")
	}

	if height == nil {
		return nil, fmt.Errorf("height must be non-nil")
	}

	var keyMerklePath *commitmenttypes.MerklePath = nil
	if includeKey {
		merklePath, err := chain.GetMerklePath(string(key))
		if err != nil {
			return nil, fmt.Errorf("fail to create merkle path on chain '%s' with path '%s' due to: %v",
				chain.ChainID(), key, err)
		}
		keyMerklePath = &merklePath
	}

	valueBytes, proof, err := chain.QueryStateAtHeight(key, int64(height.GetRevisionHeight()), true)
	if err != nil {
		return nil, fmt.Errorf("fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
			chain.ChainID(), key, height, err,
		)
	}

	return &channeltypes.MultihopProof{
		Proof:       proof,
		Value:       valueBytes,
		PrefixedKey: keyMerklePath,
	}, nil
}
