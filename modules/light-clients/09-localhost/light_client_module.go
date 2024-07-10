package localhost

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

const (
	// ModuleName defines the 09-localhost light client module name
	ModuleName = "09-localhost"
)

// SentinelProof defines the 09-localhost sentinel proof.
// Submission of nil or empty proofs is disallowed in core IBC messaging.
// This serves as a placeholder value for relayers to leverage as the proof field in various message types.
// Localhost client state verification will fail if the sentintel proof value is not provided.
var SentinelProof = []byte{0x01}

var _ exported.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface.
type LightClientModule struct {
	cdc           codec.BinaryCodec
	key           storetypes.StoreKey
	storeProvider exported.ClientStoreProvider
}

// NewLightClientModule creates and returns a new 09-localhost LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec, key storetypes.StoreKey) *LightClientModule {
	return &LightClientModule{
		cdc: cdc,
		key: key,
	}
}

// RegisterStoreProvider is called by core IBC when a LightClientModule is added to the router.
// It allows the LightClientModule to set a ClientStoreProvider which supplies isolated prefix client stores
// to IBC light client instances.
func (l *LightClientModule) RegisterStoreProvider(storeProvider exported.ClientStoreProvider) {
	l.storeProvider = storeProvider
}

// Initialize returns an error because it is stateless.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (LightClientModule) Initialize(_ sdk.Context, _ string, _, _ []byte) error {
	return errorsmod.Wrap(clienttypes.ErrClientExists, "localhost is stateless and cannot be initialized")
}

// VerifyClientMessage is unsupported by the 09-localhost client type and returns an error.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (LightClientModule) VerifyClientMessage(_ sdk.Context, _ string, _ exported.ClientMessage) error {
	return errorsmod.Wrap(clienttypes.ErrUpdateClientFailed, "client message verification is unsupported by the localhost client")
}

// CheckForMisbehaviour is unsupported by the 09-localhost client type and performs a no-op, returning false.
func (LightClientModule) CheckForMisbehaviour(_ sdk.Context, _ string, _ exported.ClientMessage) bool {
	return false
}

// UpdateStateOnMisbehaviour is unsupported by the 09-localhost client type and performs a no-op.
func (LightClientModule) UpdateStateOnMisbehaviour(_ sdk.Context, _ string, _ exported.ClientMessage) {
	// no-op
}

// UpdateState performs a no-op and returns the context height in the updated heights return value.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (LightClientModule) UpdateState(ctx sdk.Context, _ string, _ exported.ClientMessage) []exported.Height {
	return []exported.Height{clienttypes.GetSelfHeight(ctx)}
}

// VerifyMembership is a generic proof verification method which verifies the existence of a given key and value within the IBC store.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// The caller must provide the full IBC store.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (l LightClientModule) VerifyMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	ibcStore := ctx.KVStore(l.key)

	// ensure the proof provided is the expected sentinel localhost client proof
	if !bytes.Equal(proof, SentinelProof) {
		return errorsmod.Wrapf(commitmenttypes.ErrInvalidProof, "expected %s, got %s", string(SentinelProof), string(proof))
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypes.MerklePath{}, path)
	}

	if len(merklePath.GetKeyPath()) != 2 {
		return errorsmod.Wrapf(host.ErrInvalidPath, "path must be of length 2: %s", merklePath.GetKeyPath())
	}

	// The commitment prefix (eg: "ibc") is omitted when operating on the core IBC store
	bz := ibcStore.Get(merklePath.KeyPath[1])
	if bz == nil {
		return errorsmod.Wrapf(clienttypes.ErrFailedMembershipVerification, "value not found for path %s", path)
	}

	if !bytes.Equal(bz, value) {
		return errorsmod.Wrapf(clienttypes.ErrFailedMembershipVerification, "value provided does not equal value stored at path: %s", path)
	}

	return nil
}

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath within the IBC store.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// The caller must provide the full IBC store.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (l LightClientModule) VerifyNonMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	ibcStore := ctx.KVStore(l.key)

	// ensure the proof provided is the expected sentinel localhost client proof
	if !bytes.Equal(proof, SentinelProof) {
		return errorsmod.Wrapf(commitmenttypes.ErrInvalidProof, "expected %s, got %s", string(SentinelProof), string(proof))
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypes.MerklePath{}, path)
	}

	if len(merklePath.GetKeyPath()) != 2 {
		return errorsmod.Wrapf(host.ErrInvalidPath, "path must be of length 2: %s", merklePath.GetKeyPath())
	}

	// The commitment prefix (eg: "ibc") is omitted when operating on the core IBC store
	if ibcStore.Has(merklePath.KeyPath[1]) {
		return errorsmod.Wrapf(clienttypes.ErrFailedNonMembershipVerification, "value found for path %s", path)
	}

	return nil
}

// Status always returns Active. The 09-localhost status cannot be changed.
func (LightClientModule) Status(_ sdk.Context, _ string) exported.Status {
	return exported.Active
}

// LatestHeight returns the context height.
//
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.
func (LightClientModule) LatestHeight(ctx sdk.Context, _ string) exported.Height {
	return clienttypes.GetSelfHeight(ctx)
}

// TimestampAtHeight returns the current block time retrieved from the application context. The localhost client does not store consensus states and thus
// cannot provide a timestamp for the provided height.
func (LightClientModule) TimestampAtHeight(ctx sdk.Context, _ string, _ exported.Height) (uint64, error) {
	return uint64(ctx.BlockTime().UnixNano()), nil
}

// RecoverClient returns an error. The localhost cannot be modified by proposals.
func (LightClientModule) RecoverClient(_ sdk.Context, _, _ string) error {
	return errorsmod.Wrap(clienttypes.ErrUpdateClientFailed, "cannot update localhost client with a proposal")
}

// VerifyUpgradeAndUpdateState returns an error since localhost cannot be upgraded.
func (LightClientModule) VerifyUpgradeAndUpdateState(_ sdk.Context, _ string, _, _, _, _ []byte) error {
	return errorsmod.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade localhost client")
}
