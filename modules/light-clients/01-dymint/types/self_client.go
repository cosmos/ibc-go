package types

import (
	"reflect"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

var _ exported.SelfClient = (*SelfClient)(nil)

type SelfClient struct{}

// NewClientState creates a new ClientState instance
func NewSelfClient() exported.SelfClient {
	return &SelfClient{}
}

// ValidateSelfClientState validates the client parameters for a client of the running chain
// This function is only used to validate the client state the counterparty stores for this chain
// Client must be in same revision as the executing chain
// dymint doesn't care about the unbonding period, so ignore it
func (sc SelfClient) ValidateSelfClientState(
	ctx sdk.Context,
	expectedUbdPeriod time.Duration,
	clientState exported.ClientState,
) error {
	tmClient, ok := clientState.(*ClientState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "client must be a Dymint client, expected: %T, got: %T",
			&ClientState{}, tmClient)
	}

	if !tmClient.FrozenHeight.IsZero() {
		return clienttypes.ErrClientFrozen
	}

	if ctx.ChainID() != tmClient.ChainId {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "invalid chain-id. expected: %s, got: %s",
			ctx.ChainID(), tmClient.ChainId)
	}

	revision := clienttypes.ParseChainID(ctx.ChainID())

	// client must be in the same revision as executing chain
	if tmClient.LatestHeight.RevisionNumber != revision {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "client is not in the same revision as the chain. expected revision: %d, got: %d",
			tmClient.LatestHeight.RevisionNumber, revision)
	}

	selfHeight := clienttypes.NewHeight(revision, uint64(ctx.BlockHeight()))
	if tmClient.LatestHeight.GTE(selfHeight) {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "client has LatestHeight %d greater than or equal to chain height %d",
			tmClient.LatestHeight, selfHeight)
	}

	expectedProofSpecs := commitmenttypes.GetSDKSpecs()
	if !reflect.DeepEqual(expectedProofSpecs, tmClient.ProofSpecs) {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "client has invalid proof specs. expected: %v got: %v",
			expectedProofSpecs, tmClient.ProofSpecs)
	}

	if len(tmClient.UpgradePath) != 0 {
		// For now, SDK IBC implementation assumes that upgrade path (if defined) is defined by SDK upgrade module
		expectedUpgradePath := []string{upgradetypes.StoreKey, upgradetypes.KeyUpgradedIBCState}
		if !reflect.DeepEqual(expectedUpgradePath, tmClient.UpgradePath) {
			return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "upgrade path must be the upgrade path defined by upgrade module. expected %v, got %v",
				expectedUpgradePath, tmClient.UpgradePath)
		}
	}
	return nil
}

func (sc SelfClient) GetSelfConsensusStateFromBlocHeader(
	cdc codec.BinaryCodec,
	blockHeader []byte,
) (exported.ConsensusState, error) {
	// unmarshal block header
	tmBlockHeader := &tmproto.Header{}
	if err := cdc.Unmarshal(blockHeader, tmBlockHeader); err != nil {
		return nil, sdkerrors.Wrapf(clienttypes.ErrInvalidHeader, "could not unmarshal block header: %v", err)
	}
	return NewConsensusState(tmBlockHeader.Time,
		commitmenttypes.NewMerkleRoot(tmBlockHeader.GetAppHash()),
		tmBlockHeader.NextValidatorsHash), nil
}

func (sc SelfClient) ClientType() string {
	return exported.Dymint
}
