package localhost

import (
	"bytes"

	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
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
	cdc          codec.BinaryCodec
	storeService corestore.KVStoreService
}

// NewLightClientModule creates and returns a new 09-localhost LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec, storeService corestore.KVStoreService) *LightClientModule {
	return &LightClientModule{
		cdc:          cdc,
		storeService: storeService,
	}
}

// Initialize returns an error because it is stateless.
func (LightClientModule) Initialize(_ sdk.Context, _ string, _, _ []byte) error {
	return errorsmod.Wrap(clienttypes.ErrClientExists, "localhost is stateless and cannot be initialized")
}

// VerifyClientMessage is unsupported by the 09-localhost client type and returns an error.
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
func (LightClientModule) UpdateState(ctx sdk.Context, _ string, _ exported.ClientMessage) []exported.Height {
	return []exported.Height{clienttypes.GetSelfHeight(ctx)}
}

// VerifyMembership is a generic proof verification method which verifies the existence of a given key and value within the IBC store.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// The caller must provide the full IBC store.
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
	ibcStore := l.storeService.OpenKVStore(ctx)

	// ensure the proof provided is the expected sentinel localhost client proof
	if !bytes.Equal(proof, SentinelProof) {
		return errorsmod.Wrapf(commitmenttypes.ErrInvalidProof, "expected %s, got %s", string(SentinelProof), string(proof))
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	if len(merklePath.GetKeyPath()) != 2 {
		return errorsmod.Wrapf(host.ErrInvalidPath, "path must be of length 2: %s", merklePath.GetKeyPath())
	}

	// The commitment prefix (eg: "ibc") is omitted when operating on the core IBC store
	bz, err := ibcStore.Get(merklePath.KeyPath[1])
	if err != nil {
		panic(err)
	}
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
func (l LightClientModule) VerifyNonMembership(
	ctx sdk.Context,
	clientID string,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	ibcStore := l.storeService.OpenKVStore(ctx)

	// ensure the proof provided is the expected sentinel localhost client proof
	if !bytes.Equal(proof, SentinelProof) {
		return errorsmod.Wrapf(commitmenttypes.ErrInvalidProof, "expected %s, got %s", string(SentinelProof), string(proof))
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	if len(merklePath.GetKeyPath()) != 2 {
		return errorsmod.Wrapf(host.ErrInvalidPath, "path must be of length 2: %s", merklePath.GetKeyPath())
	}

	// The commitment prefix (eg: "ibc") is omitted when operating on the core IBC store
	has, err := ibcStore.Has(merklePath.KeyPath[1])
	if err != nil {
		return errorsmod.Wrapf(err, "error checking for value for path %s", path)
	}
	if has {
		return errorsmod.Wrapf(clienttypes.ErrFailedNonMembershipVerification, "value found for path %s", path)
	}

	return nil
}

// Status always returns Active. The 09-localhost status cannot be changed.
func (LightClientModule) Status(_ sdk.Context, _ string) exported.Status {
	return exported.Active
}

// LatestHeight returns the context height.
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
