package avalanche

import (
	fmt "fmt"
	"reflect"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// VerifyClientMessage checks if the clientMessage is of type Header or Misbehaviour and verifies the message
func (cs *ClientState) VerifyClientMessage(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore,
	clientMsg exported.ClientMessage,
) error {
	switch msg := clientMsg.(type) {
	case *Header:
		return cs.verifyHeader(ctx, clientStore, cdc, msg)
	case *Misbehaviour:
		return cs.verifyMisbehaviour(ctx, clientStore, cdc, msg)
	default:
		return clienttypes.ErrInvalidClientType
	}
}

// verifyHeader returns an error if:
// - the client or header provided are not parseable to tendermint types
// - the header is invalid
// - header height is less than or equal to the trusted header height
// - header revision is not equal to trusted header revision
// - header valset commit verification fails
// - header timestamp is past the trusting period in relation to the consensus state
// - header timestamp is less than or equal to the consensus state timestamp
func (cs *ClientState) verifyHeader(
	ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec,
	header *Header,
) error {
	// Retrieve trusted consensus states for each Header in misbehaviour
	consState, found := GetConsensusState(clientStore, cdc, header.SubnetHeader.Height)
	if !found {
		return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header at TrustedHeight: %s", header.SubnetHeader.Height)
	}

	// UpdateClient only accepts updates with a header at the same revision
	// as the trusted consensus state
	if header.SubnetHeader.Height.RevisionNumber != header.SubnetHeader.Height.RevisionNumber {
		return errorsmod.Wrapf(
			ErrInvalidHeaderHeight,
			"header height revision %d does not match trusted header revision %d",
			header.SubnetHeader.Height.RevisionNumber, header.SubnetHeader.Height.RevisionNumber,
		)
	}

	if header.PchainHeader.Height.RevisionNumber != header.SubnetHeader.Height.RevisionNumber {
		return errorsmod.Wrapf(
			ErrInvalidHeaderHeight,
			"header height revision %d does not match trusted header revision %d",
			header.PchainHeader.Height.RevisionNumber, header.SubnetHeader.Height.RevisionNumber,
		)
	}

	// assert header height is newer than consensus state
	if header.SubnetHeader.Height.LTE(*header.SubnetHeader.Height) {
		return errorsmod.Wrapf(
			clienttypes.ErrInvalidHeader,
			"SubnetHeader height ≤ consensus state height (%s < %s)", header.SubnetHeader.Height, header.SubnetHeader.Height,
		)
	}

	// assert header height is newer than consensus state
	if header.SubnetHeader.Height.LTE(*header.PchainHeader.Height) {
		return errorsmod.Wrapf(
			clienttypes.ErrInvalidHeader,
			"PchainHeader height ≤ consensus state height (%s < %s)", header.SubnetHeader.Height, header.PchainHeader.Height,
		)
	}

	uniqVdrs, uniqWeight, err := ValidateValidatorSet(ctx, header.SubnetHeader.PchainVdrs)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify header")
	}

	headerUniqVdrs, headerTotalWeight, err := ValidateValidatorSet(ctx, header.Vdrs)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify header")
	}
	consensusUniqVdrs, consensusTotalWeight, err := ValidateValidatorSet(ctx, consState.Vdrs)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify header")
	}

	if headerTotalWeight != consensusTotalWeight || headerTotalWeight != uniqWeight {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "failed to verify header")
	}

	if len(headerUniqVdrs) != len(consensusUniqVdrs) || len(headerUniqVdrs) != len(uniqVdrs) {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "failed to verify header")
	}
	for i := range headerUniqVdrs {
		if !reflect.DeepEqual(headerUniqVdrs[i].PublicKeyBytes, consensusUniqVdrs[i].PublicKeyBytes) || !reflect.DeepEqual(headerUniqVdrs[i].PublicKeyBytes, uniqVdrs[i].PublicKeyBytes) {
			return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "failed to verify header")
		}
	}
	return nil
}

func (cs *ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, _ exported.ClientMessage) {
	cs.FrozenHeight = FrozenHeight

	clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(cdc, cs))
}

func (cs *ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	header, ok := clientMsg.(*Header)
	if !ok {
		panic(fmt.Errorf("expected type %T, got %T", &Header{}, clientMsg))
	}

	cs.pruneOldestConsensusState(ctx, cdc, clientStore)

	// check for duplicate update
	if consensusState, _ := GetConsensusState(clientStore, cdc, header.SubnetHeader.Height); consensusState != nil {
		// perform no-op
		return []exported.Height{header.SubnetHeader.Height}
	}

	height := header.SubnetHeader.Height
	if height.GT(cs.LatestHeight) {
		cs.LatestHeight = *height
	}

	consensusState := &ConsensusState{
		Timestamp:          header.SubnetHeader.Timestamp,
		StorageRoot:        header.StorageRoot,
		SignedStorageRoot:  header.SignedStorageRoot,
		ValidatorSet:       header.ValidatorSet,
		SignedValidatorSet: header.SignedValidatorSet,
		Vdrs:               header.Vdrs,
		SignersInput:       header.SignersInput,
	}

	// set client state, consensus state and asssociated metadata
	setClientState(clientStore, cdc, cs)
	SetConsensusState(clientStore, cdc, consensusState, header.SubnetHeader.Height)
	setConsensusMetadata(ctx, clientStore, header.SubnetHeader.Height)

	return []exported.Height{height}
}
