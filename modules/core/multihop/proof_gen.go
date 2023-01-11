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
func (p ChanPath) GenerateMembershipProof(key []byte, expectedVal []byte) (*channeltypes.MsgMultihopProofs, error) {
	if len(key) == 0 {
		return nil, sdkerrors.Wrap(channeltypes.ErrMultihopProofGeneration, "key cannot be empty")
	}
	if len(expectedVal) == 0 {
		return nil, sdkerrors.Wrap(channeltypes.ErrMultihopProofGeneration, "expected value cannot be empty")
	}

	result := &channeltypes.MsgMultihopProofs{}
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
		result.KeyProof = &channeltypes.MultihopProof{Proof: keyProof}
	}

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
		path, nextPath := p[i], p[i+1]
		self, next := path.EndpointB, nextPath.EndpointB
		heightAB := self.GetClientState().GetLatestHeight()
		heightBC := next.GetClientState().GetLatestHeight()
		consStateBC, err := next.GetConsensusState(heightBC)
		if err != nil {
			return nil, sdkerrors.Wrapf(
				channeltypes.ErrMultihopProofGeneration,
				"failed to get consensus state root of chain '%s' at height %s on chain '%s': %v",
				self.ChainID(), heightBC, next.ChainID(), err,
			)
		}
		consStateBCRoot := consStateBC.GetRoot()

		for j, proofGenFunc := range proofGenFuncs {
			proof := proofGenFunc(self, next, heightAB, heightBC, consStateBCRoot)
			result[j] = append(result[j], proof)
		}
	}
	return result, nil
}

type proofGenFunc func(Endpoint, Endpoint, exported.Height, exported.Height, exported.Root) *channeltypes.MultihopProof

func genConsensusStateProof(
	chainB, chainC Endpoint,
	heightAB, heightBC exported.Height,
	consStateBCRoot exported.Root,
) *channeltypes.MultihopProof {
	chainA := chainB.Counterparty()
	consStateAB, err := chainB.GetConsensusState(heightAB)
	tryPanicOnProofGenError(err, "consensus state of chain '%s' at height %s not found on chain '%s' due to: %v",
		chainB.ChainID(), heightAB, chainC.ChainID(), err,
	)
	bzConsStateAB, err := chainB.Codec().MarshalInterface(consStateAB)
	tryPanicOnProofGenError(err, "fail to marshal consensus state of chain '%s' on chain '%s' at height %s due to: %v",
		chainA.ChainID(), chainB.ChainID(), heightAB, err,
	)
	keyConsAB, err := chainB.GetMerklePath(host.FullConsensusStatePath(chainB.ClientID(), heightAB))
	tryPanicOnProofGenError(err, "fail to create merkle path on chain '%s' with path '%s' due to: %v",
		chainB.ChainID(), host.FullConsensusStatePath(chainB.ClientID(), heightAB), err,
	)
	bzConsStateABProof, _, err := chainB.QueryProofAtHeight(
		host.FullConsensusStateKey(chainB.ClientID(), heightAB),
		int64(heightBC.GetRevisionHeight()),
	)
	tryPanicOnProofGenError(err, "fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
		chainB.ChainID(), host.FullConsensusStateKey(chainB.ClientID(), heightAB), heightBC.GetRevisionHeight(), err,
	)

	var consStateABProof commitmenttypes.MerkleProof
	err = chainB.Codec().Unmarshal(bzConsStateABProof, &consStateABProof)
	tryPanicOnProofGenError(err, "fail to unmarshal chain [%s]'s proof on chain '%s' due to: %v",
		chainA.ChainID(), chainB.ChainID(), err,
	)

	// ensure consStateAB can be verified by consStateBC
	err = consStateABProof.VerifyMembership(
		commitmenttypes.GetSDKSpecs(), consStateBCRoot,
		keyConsAB, bzConsStateAB,
	)
	tryPanicOnProofGenError(
		err,
		"fail to verify proof of chain [%s]'s consensus state on chain '%s' at path '%s' using [%s]'s state root on chain '%s' due to: %v",
		chainA.ChainID(),
		chainB.ChainID(),
		keyConsAB,
		chainB.ChainID(),
		chainC.ChainID(),
		err,
	)
	return &channeltypes.MultihopProof{
		Proof:       bzConsStateABProof,
		Value:       bzConsStateAB,
		PrefixedKey: &keyConsAB,
	}
}

func genConnProof(
	chainB, chainC Endpoint,
	heightAB, heightBC exported.Height,
	consStateBCRoot exported.Root,
) *channeltypes.MultihopProof {
	chainA := chainB.Counterparty()
	keyConnAB, err := chainB.GetMerklePath(host.ConnectionPath(chainB.ConnectionID()))
	tryPanicOnProofGenError(err, "fail to create merkle path on chain '%s' with path '%s' due to: %v",
		chainB.ChainID(), host.ConnectionPath(chainB.ConnectionID()), err,
	)
	bzConnABProof, _, err := chainB.QueryProofAtHeight(
		host.ConnectionKey(chainB.ConnectionID()),
		int64(heightBC.GetRevisionHeight()),
	)
	tryPanicOnProofGenError(err, "fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
		chainB.ChainID(), host.ConnectionKey(chainB.ConnectionID()), heightBC.GetRevisionHeight(), err,
	)
	var connProof commitmenttypes.MerkleProof
	err = chainB.Codec().Unmarshal(bzConnABProof, &connProof)
	tryPanicOnProofGenError(err, "fail to unmarshal chain [%s]'s proof on chain '%s' due to: %v",
		chainA.ChainID(), chainB.ChainID(), err,
	)
	connAB, err := chainB.GetConnection()
	tryPanicOnProofGenError(err, "fail to get connection '%s' on chain '%s' due to: %v",
		chainB.ConnectionID(), chainB.ChainID(), err,
	)
	bzConnAB, err := chainB.Codec().Marshal(connAB)
	tryPanicOnProofGenError(err, "fail to marshal connection '%s' on chain '%s' due to: %v",
		chainB.ConnectionID(), chainB.ChainID(), err,
	)
	// ensure connecitonAB can be verified by consStateBC
	err = connProof.VerifyMembership(
		commitmenttypes.GetSDKSpecs(), consStateBCRoot,
		keyConnAB, bzConnAB,
	)
	tryPanicOnProofGenError(
		err,
		"fail to verify proof of chain [%s]'s connection on chain '%s' at path '%s' using [%s]'s state root on chain '%s' due to: %v",
		chainA.ChainID(),
		chainB.ChainID(),
		keyConnAB,
		chainB.ChainID(),
		chainC.ChainID(),
		err,
	)
	return &channeltypes.MultihopProof{
		Proof:       bzConnABProof,
		Value:       bzConnAB,
		PrefixedKey: &keyConnAB,
	}
}

func tryPanicOnProofGenError(err error, format string, args ...interface{}) {
	if err != nil {
		panic(fmt.Sprintf(format, args...))
	}
}
