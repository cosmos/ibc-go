package multihop

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// Endpoint represents a Cosmos chain endpoint for queries.
// Endpoint is stateless from caller's perspective.
type Endpoint interface {
	ChainID() string
	Codec() codec.BinaryCodec
	GetClientState() exported.ClientState
	GetConsensusState(height exported.Height) (exported.ConsensusState, error)
	GetConnection() (*connectiontypes.ConnectionEnd, error)
	// Returns the proof of the `key`` at `height` within the ibc module store.
	QueryProofAtHeight(key []byte, height int64) ([]byte, clienttypes.Height, error)
	GetMerklePath(path string) (commitmenttypes.MerklePath, error)
	// UpdateClient updates the clientState of counterparty chain's header
	UpdateClient() error
	Counterparty() Endpoint
}

// // PairedEnd represents one end of an IBC connection.
// type PairedEnd struct {
// 	Endpoint
// 	ClientID     string
// 	ConnectionID string
// 	Counterparty *PairedEnd
// }

// Path contains two endpoints of chains that have a direct IBC connection, ie. a single-hop IBC path.
type Path struct {
	EndpointA Endpoint
	EndpointB Endpoint
}

// ChanPath represents a multihop channel path that spans 2 or more single-hop `Path`s.
type ChanPath []Path

// GenerateProof generates a proof for the given path with expected value on the the source chain, which is to be verified on the dest
// chain.
func (p ChanPath) GenerateMembershipProof(
	key []byte,
	expectedVal []byte,
	doVerify bool,
) (*channeltypes.MsgMultihopProofs, error) {
	if len(key) == 0 {
		return nil, sdkerrors.Wrap(channeltypes.ErrMultihopProofGeneration, "key cannot be empty")
	}
	if len(expectedVal) == 0 {
		return nil, sdkerrors.Wrap(channeltypes.ErrMultihopProofGeneration, "expected value cannot be empty")
	}

	proofs := &channeltypes.MsgMultihopProofs{}
	// generate proof for key on source chain
	{
		endpointB := p.source().Counterparty()
		heightBC := endpointB.GetClientState().GetLatestHeight()
		keyProof, _, err := endpointB.QueryProofAtHeight(key, int64(heightBC.GetRevisionHeight()))
		if err != nil {
			return nil, sdkerrors.Wrapf(
				channeltypes.ErrMultihopProofGeneration,
				"failed to generate proof for key %s on source chain %s at height %d: %v",
				string(key),
				endpointB.ChainID(),
				heightBC.GetRevisionHeight(),
				err,
			)
		}
		// TODO: always verify membership

		// save proof of key/value on source chain
		proofs.KeyProof = &channeltypes.MultihopProof{Proof: keyProof}
	}

	return nil, nil
}

// GenerateNonMembershipProof generates a proof for the given path with NO value stored on the the source chain, which
// is to be verified on the dest chain.
func (p ChanPath) GenerateNonMembershipProof(key []byte) (*channeltypes.MsgMultihopProofs, error) {
	return nil, nil
}

// The source chain
func (p ChanPath) source() Endpoint {
	return p[0].EndpointA
}

// The destination chain
func (p ChanPath) dest() Endpoint {
	return p[len(p)-1].EndpointB
}
