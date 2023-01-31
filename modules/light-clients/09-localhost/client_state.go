package localhost

import (
	"bytes"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new 09-localhost ClientState instance.
func NewClientState(chainID string, height clienttypes.Height) exported.ClientState {
	return &ClientState{
		ChainId:      chainID,
		LatestHeight: height,
	}
}

// GetChainID returns the client state chain ID.
func (cs ClientState) GetChainID() string {
	return cs.ChainId
}

// ClientType returns the 09-localhost client type.
func (cs ClientState) ClientType() string {
	return exported.Localhost
}

// GetLatestHeight returns the 09-localhost client state latest height.
func (cs ClientState) GetLatestHeight() exported.Height {
	return cs.LatestHeight
}

// Status always returns Active. The 09-localhost status cannot be changed.
func (cs ClientState) Status(_ sdk.Context, _ sdk.KVStore, _ codec.BinaryCodec) exported.Status {
	return exported.Active
}

// Validate performs a basic validation of the client state fields.
func (cs ClientState) Validate() error {
	if strings.TrimSpace(cs.ChainId) == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidChainID, "chain id cannot be blank")
	}

	if cs.LatestHeight.RevisionHeight == 0 {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidHeight, "local revision height cannot be zero")
	}

	return nil
}

// ZeroCustomFields returns the same client state since there are no custom fields in the 09-localhost client state.
func (cs ClientState) ZeroCustomFields() exported.ClientState {
	return &cs
}

// Initialize ensures that initial consensus state for localhost is nil.
func (cs ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, consState exported.ConsensusState) error {
	if consState != nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidConsensus, "initial consensus state for localhost must be nil.")
	}

	clientState := ClientState{
		ChainId:      ctx.ChainID(),
		LatestHeight: clienttypes.GetSelfHeight(ctx),
	}

	clientStore.Set([]byte(host.KeyClientState), clienttypes.MustMarshalClientState(cdc, &clientState))

	return nil
}

// GetTimestampAtHeight must return the timestamp for the consensus state associated with the provided height.
func (cs ClientState) GetTimestampAtHeight(ctx sdk.Context, clientStore sdk.KVStore, cdc codec.BinaryCodec, height exported.Height) (uint64, error) {
	return uint64(ctx.BlockTime().UnixNano()), nil
}

// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
func (cs ClientState) VerifyMembership(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	bz := store.Get([]byte(path.String()))
	if bz == nil {
		return sdkerrors.Wrapf(clienttypes.ErrFailedChannelStateVerification, "todo: update error -- not found for path %s", path)
	}

	if !bytes.Equal(bz, value) {
		return sdkerrors.Wrapf(
			clienttypes.ErrFailedChannelStateVerification,
			"todo: update error",
		)
	}

	return nil
}

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
func (cs ClientState) VerifyNonMembership(
	ctx sdk.Context,
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	bz := store.Get([]byte(path.String()))
	if bz != nil {
		return sdkerrors.Wrapf(clienttypes.ErrFailedChannelStateVerification, "todo: update error -- found for path %s", path)
	}

	return nil
}

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (cs ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	return nil
}

// Checks for evidence of a misbehaviour in Header or Misbehaviour type. It assumes the ClientMessage
// has already been verified.
func (cs ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) bool {
	return false
}

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified.
func (cs ClientState) UpdateStateOnMisbehaviour(_ sdk.Context, _ codec.BinaryCodec, _ sdk.KVStore, _ exported.ClientMessage) {
}

// UpdateState updates and stores as necessary any associated information for an IBC client, such as the ClientState and corresponding ConsensusState.
// Upon successful update, a list of consensus heights is returned. It assumes the ClientMessage has already been verified.
func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	height := clienttypes.GetSelfHeight(ctx)

	clientState := NewClientState(ctx.ChainID(), height)
	clientStore.Set([]byte(host.KeyClientState), clienttypes.MustMarshalClientState(cdc, clientState))

	return []exported.Height{height}
}

// ExportMetadata is a no-op for the 09-localhost client.
func (cs ClientState) ExportMetadata(_ sdk.KVStore) []exported.GenesisMetadata {
	return nil
}

// CheckSubstituteAndUpdateState returns an error. The localhost cannot be modified by
// proposals.
func (cs ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore, substituteClientStore sdk.KVStore, substituteClient exported.ClientState) error {
	return sdkerrors.Wrap(clienttypes.ErrUpdateClientFailed, "cannot update localhost client with a proposal")
}

// VerifyUpgradeAndUpdateState returns an error since localhost cannot be upgraded
func (cs ClientState) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	store sdk.KVStore,
	newClient exported.ClientState,
	newConsState exported.ConsensusState,
	proofUpgradeClient,
	proofUpgradeConsState []byte,
) error {
	return sdkerrors.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade localhost client")
}
