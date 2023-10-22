package avalanche

import (
	"reflect"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

const (
	// MaxChainIDLen is a maximum length of the chain ID.
	MaxChainIDLen = 50
)

// NewClientState creates a new ClientState instance
func NewClientState(
	chainID string, trustLevel Fraction,
	trustingPeriod time.Duration,
	maxClockDrift time.Duration,
	latestHeight clienttypes.Height,
	upgradePath string,
	proof [][]byte,
) *ClientState {
	return &ClientState{
		ChainId:        chainID,
		TrustLevel:     trustLevel,
		TrustingPeriod: trustingPeriod,
		MaxClockDrift:  maxClockDrift,
		LatestHeight:   latestHeight,
		FrozenHeight:   clienttypes.ZeroHeight(),
		UpgradePath:    upgradePath,
		Proof:          proof,
	}
}

func (cs *ClientState) ClientType() string {
	return exported.Avalanche
}

func (cs *ClientState) GetChainID() string {
	return cs.ChainId
}

func (cs *ClientState) GetLatestHeight() exported.Height {
	return cs.LatestHeight
}

func (cs *ClientState) Validate() error {
	if strings.TrimSpace(cs.ChainId) == "" {
		return errorsmod.Wrap(ErrInvalidChainID, "chain id cannot be empty string")
	}

	if len(cs.ChainId) > MaxChainIDLen {
		return errorsmod.Wrapf(ErrInvalidChainID, "chainID is too long; got: %d, max: %d", len(cs.ChainId), MaxChainIDLen)
	}

	if cs.TrustingPeriod <= 0 {
		return errorsmod.Wrap(ErrInvalidTrustingPeriod, "trusting period must be greater than zero")
	}

	// the latest height revision number must match the chain id revision number
	if cs.LatestHeight.RevisionNumber != clienttypes.ParseChainID(cs.ChainId) {
		return errorsmod.Wrapf(ErrInvalidHeaderHeight,
			"latest height revision number must match chain id revision number (%d != %d)", cs.LatestHeight.RevisionNumber, clienttypes.ParseChainID(cs.ChainId))
	}
	if cs.LatestHeight.RevisionHeight == 0 {
		return errorsmod.Wrapf(ErrInvalidHeaderHeight, "tendermint client's latest height revision height cannot be zero")
	}
	// UpgradePath may be empty, but if it isn't, each key must be non-empty
	for cs.UpgradePath == "" {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "upgrade path cannot be empty")
	}

	return nil
}

func (cs *ClientState) Status(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec) exported.Status {
	if !cs.FrozenHeight.IsZero() {
		return exported.Frozen
	}

	// get latest consensus state from clientStore to check for expiry
	consState, found := GetConsensusState(clientStore, cdc, cs.GetLatestHeight())
	if !found {
		// if the client state does not have an associated consensus state for its latest height
		// then it must be expired
		return exported.Expired
	}

	if cs.IsExpired(consState.Timestamp, ctx.BlockTime()) {
		return exported.Expired
	}

	return exported.Active
}

// IsExpired returns whether or not the client has passed the trusting period since the last
// update (in which case no headers are considered valid).
func (cs *ClientState) IsExpired(latestTimestamp, now time.Time) bool {
	expirationTime := latestTimestamp.Add(cs.TrustingPeriod)
	return !expirationTime.After(now)
}

func (cs *ClientState) ZeroCustomFields() exported.ClientState {
	// copy over all chain-specified fields
	// and leave custom fields empty
	return &ClientState{
		ChainId:      cs.ChainId,
		LatestHeight: cs.LatestHeight,
		UpgradePath:  cs.UpgradePath,
	}
}

func (cs *ClientState) GetTimestampAtHeight(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height) (uint64, error) {
	// get consensus state at height from clientStore to check for expiry
	consState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return 0, errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "height (%s)", height)
	}
	return consState.GetTimestamp(), nil
}

func (cs *ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, consState exported.ConsensusState) error {
	consensusState, ok := consState.(*ConsensusState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type: %T, got: %T",
			&ConsensusState{}, consState)
	}

	setClientState(clientStore, cdc, cs)
	SetConsensusState(clientStore, cdc, consensusState, cs.GetLatestHeight())

	return nil
}

// verifyDelayPeriodPassed will ensure that at least delayTimePeriod amount of time and delayBlockPeriod number of blocks have passed
// since consensus state was submitted before allowing verification to continue.
func verifyDelayPeriodPassed(ctx sdk.Context, clientStore storetypes.KVStore, proofHeight exported.Height, delayTimePeriod, delayBlockPeriod uint64) error {
	if delayTimePeriod != 0 {
		// check that executing chain's timestamp has passed consensusState's processed time + delay time period
		processedTime, ok := GetProcessedTime(clientStore, proofHeight)
		if !ok {
			return errorsmod.Wrapf(ErrProcessedTimeNotFound, "processed time not found for height: %s", proofHeight)
		}

		currentTimestamp := uint64(ctx.BlockTime().UnixNano())
		validTime := processedTime + delayTimePeriod

		// NOTE: delay time period is inclusive, so if currentTimestamp is validTime, then we return no error
		if currentTimestamp < validTime {
			return errorsmod.Wrapf(ErrDelayPeriodNotPassed, "cannot verify packet until time: %d, current time: %d",
				validTime, currentTimestamp)
		}

	}

	if delayBlockPeriod != 0 {
		// check that executing chain's height has passed consensusState's processed height + delay block period
		processedHeight, ok := GetProcessedHeight(clientStore, proofHeight)
		if !ok {
			return errorsmod.Wrapf(ErrProcessedHeightNotFound, "processed height not found for height: %s", proofHeight)
		}

		currentHeight := clienttypes.GetSelfHeight(ctx)
		validHeight := clienttypes.NewHeight(processedHeight.GetRevisionNumber(), processedHeight.GetRevisionHeight()+delayBlockPeriod)

		// NOTE: delay block period is inclusive, so if currentHeight is validHeight, then we return no error
		if currentHeight.LT(validHeight) {
			return errorsmod.Wrapf(ErrDelayPeriodNotPassed, "cannot verify packet until height: %s, current height: %s",
				validHeight, currentHeight)
		}
	}

	return nil
}

// pruneOldestConsensusState will retrieve the earliest consensus state for this clientID and check if it is expired. If it is,
// that consensus state will be pruned from store along with all associated metadata. This will prevent the client store from
// becoming bloated with expired consensus states that can no longer be used for updates and packet verification.
func (cs ClientState) pruneOldestConsensusState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore) {
	// Check the earliest consensus state to see if it is expired, if so then set the prune height
	// so that we can delete consensus state and all associated metadata.
	var (
		pruneHeight exported.Height
	)

	pruneCb := func(height exported.Height) bool {
		consState, found := GetConsensusState(clientStore, cdc, height)
		// this error should never occur
		if !found {
			panic(errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "failed to retrieve consensus state at height: %s", height))
		}

		if cs.IsExpired(consState.Timestamp, ctx.BlockTime()) {
			pruneHeight = height
		}

		return true
	}

	IterateConsensusStateAscending(clientStore, pruneCb)

	// if pruneHeight is set, delete consensus state and metadata
	if pruneHeight != nil {
		deleteConsensusState(clientStore, pruneHeight)
		deleteConsensusMetadata(clientStore, pruneHeight)
	}
}

func (cs *ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore, substituteClientStore storetypes.KVStore, substituteClient exported.ClientState) error {
	substituteClientState, ok := substituteClient.(*ClientState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "expected type %T, got %T", &ClientState{}, substituteClient)
	}

	if !IsMatchingClientState(*cs, *substituteClientState) {
		return errorsmod.Wrap(clienttypes.ErrInvalidSubstitute, "subject client state does not match substitute client state")
	}

	if cs.Status(ctx, subjectClientStore, cdc) == exported.Frozen {
		// unfreeze the client
		cs.FrozenHeight = clienttypes.ZeroHeight()
	}

	// copy consensus states and processed time from substitute to subject
	// starting from initial height and ending on the latest height (inclusive)
	height := substituteClientState.GetLatestHeight()

	consensusState, found := GetConsensusState(substituteClientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "unable to retrieve latest consensus state for substitute client")
	}

	SetConsensusState(subjectClientStore, cdc, consensusState, height)

	// set metadata stored for the substitute consensus state
	processedHeight, found := GetProcessedHeight(substituteClientStore, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrUpdateClientFailed, "unable to retrieve processed height for substitute client latest height")
	}

	processedTime, found := GetProcessedTime(substituteClientStore, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrUpdateClientFailed, "unable to retrieve processed time for substitute client latest height")
	}

	setConsensusMetadataWithValues(subjectClientStore, height, processedHeight, processedTime)

	cs.LatestHeight = substituteClientState.LatestHeight
	cs.ChainId = substituteClientState.ChainId

	// set new trusting period based on the substitute client state
	cs.TrustingPeriod = substituteClientState.TrustingPeriod

	// no validation is necessary since the substitute is verified to be Active
	// in 02-client.
	setClientState(subjectClientStore, cdc, cs)

	return nil
}

// IsMatchingClientState returns true if all the client state parameters match
// except for frozen height, latest height, trusting period, chain-id.
func IsMatchingClientState(subject, substitute ClientState) bool {
	// zero out parameters which do not need to match
	subject.LatestHeight = clienttypes.ZeroHeight()
	subject.FrozenHeight = clienttypes.ZeroHeight()
	subject.TrustingPeriod = time.Duration(0)
	substitute.LatestHeight = clienttypes.ZeroHeight()
	substitute.FrozenHeight = clienttypes.ZeroHeight()
	substitute.TrustingPeriod = time.Duration(0)
	subject.ChainId = ""
	substitute.ChainId = ""
	// sets both sets of flags to true as these flags have been DEPRECATED, see ADR-026 for more information
	subject.AllowUpdateAfterExpiry = true
	substitute.AllowUpdateAfterExpiry = true
	subject.AllowUpdateAfterMisbehaviour = true
	substitute.AllowUpdateAfterMisbehaviour = true

	return reflect.DeepEqual(subject, substitute)
}

func (cs *ClientState) VerifyUpgradeAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, upgradedClient exported.ClientState, upgradedConsState exported.ConsensusState, proofUpgradeClient, proofUpgradeConsState []byte) error {
	if len(cs.UpgradePath) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidUpgradeClient, "cannot upgrade client, no upgrade path set")
	}

	// last height of current counterparty chain must be client's latest height
	lastHeight := cs.GetLatestHeight()

	if !upgradedClient.GetLatestHeight().GT(lastHeight) {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidHeight, "upgraded client height %s must be at greater than current client height %s",
			upgradedClient.GetLatestHeight(), lastHeight)
	}

	// upgraded client state and consensus state must be IBC tendermint client state and consensus state
	// this may be modified in the future to upgrade to a new IBC tendermint type
	// counterparty must also commit to the upgraded consensus state at a sub-path under the upgrade path specified
	avaUpgradeClient, ok := upgradedClient.(*ClientState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "upgraded client must be Tendermint client. expected: %T got: %T",
			&ClientState{}, upgradedClient)
	}
	avaUpgradeConsState, ok := upgradedConsState.(*ConsensusState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidConsensus, "upgraded consensus state must be Tendermint consensus state. expected %T, got: %T",
			&ConsensusState{}, upgradedConsState)
	}

	// Must prove against latest consensus state to ensure we are verifying against latest upgrade plan
	// This verifies that upgrade is intended for the provided revision, since committed client must exist
	// at this consensus state
	consState, found := GetConsensusState(clientStore, cdc, lastHeight)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "could not retrieve consensus state for lastHeight")
	}

	// Verify client proof
	bz, err := cdc.MarshalInterface(upgradedClient.ZeroCustomFields())
	if err != nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "could not marshal client state: %v", err)
	}
	keyClientMerkle := MerkleKey{Key: cs.UpgradePath}

	err = VerifyMembership(cs.Proof, consState.StorageRoot, bz, &keyClientMerkle)
	if err != nil {
		return err
	}

	// Verify consensus state proof
	bz, err = cdc.MarshalInterface(upgradedConsState)
	if err != nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidConsensus, "could not marshal consensus state: %v", err)
	}
	keyConsStateMerkle := MerkleKey{Key: cs.UpgradePath}

	err = VerifyMembership(cs.Proof, consState.StorageRoot, bz, &keyConsStateMerkle)
	if err != nil {
		return err
	}

	// Construct new client state and consensus state
	// Relayer chosen client parameters are ignored.
	// All chain-chosen parameters come from committed client, all client-chosen parameters
	// come from current client.
	newClientState := NewClientState(
		avaUpgradeClient.ChainId,
		cs.TrustLevel,
		cs.TrustingPeriod,
		cs.MaxClockDrift,
		avaUpgradeClient.LatestHeight,
		avaUpgradeClient.UpgradePath,
		avaUpgradeClient.Proof,
	)

	if err := newClientState.Validate(); err != nil {
		return errorsmod.Wrap(err, "updated client state failed basic validation")
	}

	// The new consensus state is merely used as a trusted kernel against which headers on the new
	// chain can be verified. The root is just a stand-in sentinel value as it cannot be known in advance, thus no proof verification will pass.
	// The timestamp and the NextValidatorsHash of the consensus state is the blocktime and NextValidatorsHash
	// of the last block committed by the old chain. This will allow the first block of the new chain to be verified against
	// the last validators of the old chain so long as it is submitted within the TrustingPeriod of this client.
	// NOTE: We do not set processed time for this consensus state since this consensus state should not be used for packet verification
	// as the root is empty. The next consensus state submitted using update will be usable for packet-verification.
	newConsState := NewConsensusState(
		avaUpgradeConsState.Timestamp,
		avaUpgradeConsState.Vdrs,
		avaUpgradeConsState.StorageRoot,
		avaUpgradeConsState.SignedStorageRoot,
		avaUpgradeConsState.ValidatorSet,
		avaUpgradeConsState.SignedValidatorSet,
		avaUpgradeConsState.SignersInput,
	)

	setClientState(clientStore, cdc, newClientState)
	SetConsensusState(clientStore, cdc, newConsState, newClientState.LatestHeight)
	setConsensusMetadata(ctx, clientStore, avaUpgradeClient.LatestHeight)

	return nil
}

// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// If a zero proof height is passed in, it will fail to retrieve the associated consensus state.
func (cs ClientState) VerifyMembership(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	// TODO
	networkID := uint32(1)

	if cs.GetLatestHeight().LT(height) {
		return errorsmod.Wrapf(
			ibcerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.GetLatestHeight(), height,
		)
	}

	if err := verifyDelayPeriodPassed(ctx, clientStore, height, delayTimePeriod, delayBlockPeriod); err != nil {
		return err
	}

	consensusState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "please ensure the proof was constructed against a height that exists on the client")
	}

	vdrs, totalWeigth, err := ValidateValidatorSet(ctx, consensusState.Vdrs)
	if err != nil {
		return err
	}

	chainID, _ := ids.ToID([]byte(cs.ChainId))
	unsignedMsg, _ := warp.NewUnsignedMessage(
		networkID,
		chainID,
		consensusState.ValidatorSet,
	)

	// check ValidatorSet by SignedValidatorSet signature, and check signers and vdrs ratio by cs.TrustLevel ratio
	err = VerifyBls(consensusState.SignersInput, SetSignature(consensusState.SignedValidatorSet), unsignedMsg.Bytes(), vdrs, totalWeigth, cs.TrustLevel.Numerator, cs.TrustLevel.Denominator)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify ValidatorSet signature")
	}

	unsignedMsg, _ = warp.NewUnsignedMessage(
		networkID,
		chainID,
		consensusState.StorageRoot,
	)

	// check StorageRoot by SignedStorageRoot signature, and check signers and vdrs ratio by cs.TrustLevel ratio
	err = VerifyBls(consensusState.SignersInput, SetSignature(consensusState.SignedStorageRoot), unsignedMsg.Bytes(), vdrs, totalWeigth, cs.TrustLevel.Numerator, cs.TrustLevel.Denominator)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify StorageRoot signature")
	}

	key := path.(*MerkleKey)

	// check merkleProof verifycation, by go-ethereum lib
	return VerifyMembership(cs.Proof, consensusState.StorageRoot, value, key)
}

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
// If a zero proof height is passed in, it will fail to retrieve the associated consensus state.
func (cs ClientState) VerifyNonMembership(
	ctx sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	// TODO
	networkID := uint32(1)

	if cs.GetLatestHeight().LT(height) {
		return errorsmod.Wrapf(
			ibcerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.GetLatestHeight(), height,
		)
	}

	if err := verifyDelayPeriodPassed(ctx, clientStore, height, delayTimePeriod, delayBlockPeriod); err != nil {
		return err
	}

	consensusState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "please ensure the proof was constructed against a height that exists on the client")
	}

	vdrs, totalWeigth, err := ValidateValidatorSet(ctx, consensusState.Vdrs)
	if err != nil {
		return err
	}

	chainID, _ := ids.ToID([]byte(cs.ChainId))
	unsignedMsg, _ := warp.NewUnsignedMessage(
		networkID,
		chainID,
		consensusState.ValidatorSet,
	)

	// check ValidatorSet by SignedValidatorSet signature, and check signers and vdrs ratio by cs.TrustLevel ratio
	err = VerifyBls(consensusState.SignersInput, SetSignature(consensusState.SignedValidatorSet), unsignedMsg.Bytes(), vdrs, totalWeigth, cs.TrustLevel.Numerator, cs.TrustLevel.Denominator)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify ValidatorSet signature")
	}

	unsignedMsg, _ = warp.NewUnsignedMessage(
		networkID,
		chainID,
		consensusState.StorageRoot,
	)

	// check StorageRoot by SignedStorageRoot signature, and check signers and vdrs ratio by cs.TrustLevel ratio
	err = VerifyBls(consensusState.SignersInput, SetSignature(consensusState.SignedStorageRoot), unsignedMsg.Bytes(), vdrs, totalWeigth, cs.TrustLevel.Numerator, cs.TrustLevel.Denominator)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify StorageRoot signature")
	}

	key := path.(*MerkleKey)

	return VerifyNonMembership(cs.Proof, consensusState.StorageRoot, key)
}
