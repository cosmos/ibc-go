package multihop

import (
	"fmt"
	"log"

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

// GenerateProof generates a proof for the given path with expected value on the the source chain, which is to be verified on the dest
// chain.
func (p ChanPath) GenerateMembershipProof(
	key []byte,
	expectedVal []byte,
) (result *channeltypes.MsgMultihopProofs, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = sdkerrors.Wrapf(channeltypes.ErrMultihopProofGeneration, "%v", r)
		}
	}()

	log.Printf("Generating proof on ChanPath\n")
	result = &channeltypes.MsgMultihopProofs{}
	// generate proof for key on source chain, where key/expectedValue are checked
	result.KeyProof = ensureKeyValueProof(key, expectedVal, p[0].EndpointB, nil, nil)

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
			proof := proofGenFunc(chainB, chainC, heightAB, heightBC, consStateBCRoot)
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
	chainAID := chainB.Counterparty().ChainID()
	consStateAB, err := chainB.GetConsensusState(heightAB)
	panicIfErr(err, "consensus state of chain '%s' at height %s not found on chain '%s' due to: %v",
		chainB.ChainID(), heightAB, chainC.ChainID(), err,
	)
	bzConsStateAB, err := chainB.Codec().MarshalInterface(consStateAB)
	panicIfErr(err, "fail to marshal consensus state of chain '%s' on chain '%s' at height %s due to: %v",
		chainAID, chainB.ChainID(), heightAB, err,
	)
	keyConsAB, err := chainB.GetMerklePath(host.FullConsensusStatePath(chainB.ClientID(), heightAB))
	panicIfErr(err, "fail to create merkle path on chain '%s' with path '%s' due to: %v",
		chainB.ChainID(), host.FullConsensusStatePath(chainB.ClientID(), heightAB), err,
	)
	bzConsStateABProof, _, err := chainB.QueryProofAtHeight(
		host.FullConsensusStateKey(chainB.ClientID(), heightAB),
		int64(heightBC.GetRevisionHeight()),
	)
	panicIfErr(err, "fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
		chainB.ChainID(), host.FullConsensusStateKey(chainB.ClientID(), heightAB), heightBC.GetRevisionHeight(), err,
	)

	var consStateABProof commitmenttypes.MerkleProof
	err = chainB.Codec().Unmarshal(bzConsStateABProof, &consStateABProof)
	panicIfErr(err, "fail to unmarshal chain [%s]'s proof on chain '%s' due to: %v",
		chainAID, chainB.ChainID(), err,
	)

	// ensure consStateAB can be verified by consStateBC
	err = consStateABProof.VerifyMembership(
		commitmenttypes.GetSDKSpecs(), consStateBCRoot,
		keyConsAB, bzConsStateAB,
	)
	panicIfErr(
		err,
		"fail to verify proof of chain [%s]'s consensus state on chain '%s' at path '%s' using [%s]'s state root on chain '%s' due to: %v",
		chainAID,
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
	panicIfErr(err, "fail to create merkle path on chain '%s' with path '%s' due to: %v",
		chainB.ChainID(), host.ConnectionPath(chainB.ConnectionID()), err,
	)
	bzConnABProof, _, err := chainB.QueryProofAtHeight(
		host.ConnectionKey(chainB.ConnectionID()),
		int64(heightBC.GetRevisionHeight()),
	)
	panicIfErr(err, "fail to generate proof on chain '%s' for key '%s' at height %d due to: %v",
		chainB.ChainID(), host.ConnectionKey(chainB.ConnectionID()), heightBC.GetRevisionHeight(), err,
	)
	var connProof commitmenttypes.MerkleProof
	err = chainB.Codec().Unmarshal(bzConnABProof, &connProof)
	panicIfErr(err, "fail to unmarshal chain [%s]'s proof on chain '%s' due to: %v",
		chainA.ChainID(), chainB.ChainID(), err,
	)
	connAB, err := chainB.GetConnection()
	panicIfErr(err, "fail to get connection '%s' on chain '%s' due to: %v",
		chainB.ConnectionID(), chainB.ChainID(), err,
	)
	bzConnAB, err := chainB.Codec().Marshal(connAB)
	panicIfErr(err, "fail to marshal connection '%s' on chain '%s' due to: %v",
		chainB.ConnectionID(), chainB.ChainID(), err,
	)
	// ensure connecitonAB can be verified by consStateBC
	err = connProof.VerifyMembership(
		commitmenttypes.GetSDKSpecs(), consStateBCRoot,
		keyConnAB, bzConnAB,
	)
	panicIfErr(
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

// ensureKeyValueProof ensures that the proof of the key-value pair stored on A can be verified by A's consensus state
// root stored on B at heightAB. where A--B is connected by a single ibc connection.
// Panic if proof generation or verification fails.
func ensureKeyValueProof(
	key, value []byte,
	chainB Endpoint,
	heightAB exported.Height,
	consStateABRoot exported.Root,
) *channeltypes.MultihopProof {
	if len(key) == 0 || len(value) == 0 {
		panic("key and value must be non-empty")
	}

	chainA := chainB.Counterparty()
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
	var proof commitmenttypes.MerkleProof
	err = chainB.Codec().Unmarshal(bzProof, &proof)
	panicIfErr(err, "fail to unmarshal chain [%s]'s proof on chain '%s' due to: %v",
		chainB.Counterparty().ChainID(), chainB.ChainID(), err,
	)

	// ensure key-value pair can be verified by consStateBC
	err = proof.VerifyMembership(
		commitmenttypes.GetSDKSpecs(), consStateABRoot,
		keyMerklePath, value,
	)
	panicIfErr(
		err,
		"fail to verify proof of chain [%s]'s key-value pair at path '%s' at height %s due to: %v",
		chainA.ChainID(), key, heightAB, err,
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
