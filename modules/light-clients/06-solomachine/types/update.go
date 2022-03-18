package types

import (
	"encoding/hex"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// CheckHeaderAndUpdateState checks if the provided header is valid and updates
// the consensus state if appropriate. It returns an error if:
// - the header provided is not parseable to a solo machine header
// - the header sequence does not match the current sequence
// - the header timestamp is less than the consensus state timestamp
// - the currently registered public key did not provide the update signature
func (cs ClientState) CheckHeaderAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore,
	msg exported.ClientMessage,
) (exported.ClientState, exported.ConsensusState, error) {
	if err := cs.VerifyClientMessage(ctx, cdc, clientStore, msg); err != nil {
		return nil, nil, err
	}

	foundMisbehaviour := cs.CheckForMisbehaviour(ctx, cdc, clientStore, msg)
	if foundMisbehaviour {
		return cs.UpdateStateOnMisbehaviour(ctx, cdc, clientStore)
	}

	return cs.UpdateState(ctx, cdc, clientStore, msg)
}

// VerifyClientMessage introspects the provided ClientMessage and checks its validity
// A Solomachine Header is considered valid if the currently registered public key has signed over the new public key with the correct sequence
// A Solomachine Misbehaviour is considered valid if duplicate signatures of the current public key are found on two different messages at a given sequence
func (cs ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	switch msg := clientMsg.(type) {
	case *Header:
		return cs.verifyHeader(ctx, cdc, clientStore, msg)
	case *Misbehaviour:
		return cs.verifyMisbehaviour(ctx, cdc, clientStore, msg)
	default:
		return sdkerrors.Wrapf(types.ErrInvalidClientType, "expected type of %T or %T, got type %T", Header{}, Misbehaviour{}, msg)
	}
}

func (cs ClientState) verifyHeader(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, header *Header) error {
	// assert update sequence is current sequence
	if header.Sequence != cs.Sequence {
		return sdkerrors.Wrapf(
			types.ErrInvalidHeader,
			"header sequence does not match the client state sequence (%d != %d)", header.Sequence, cs.Sequence,
		)
	}

	// assert update timestamp is not less than current consensus state timestamp
	if header.Timestamp < cs.ConsensusState.Timestamp {
		return sdkerrors.Wrapf(
			types.ErrInvalidHeader,
			"header timestamp is less than to the consensus state timestamp (%d < %d)", header.Timestamp, cs.ConsensusState.Timestamp,
		)
	}

	// assert currently registered public key signed over the new public key with correct sequence
	data, err := HeaderSignBytes(cdc, header)
	if err != nil {
		return err
	}

	sigData, err := UnmarshalSignatureData(cdc, header.Signature)
	if err != nil {
		return err
	}

	publicKey, err := cs.ConsensusState.GetPubKey()
	if err != nil {
		return err
	}

	if err := VerifySignature(publicKey, data, sigData); err != nil {
		return sdkerrors.Wrap(ErrInvalidHeader, err.Error())
	}

	return nil
}

func (cs ClientState) verifyMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, misbehaviour *Misbehaviour) error {
	// NOTE: a check that the misbehaviour message data are not equal is done by
	// misbehaviour.ValidateBasic which is called by the 02-client keeper.
	// verify first signature
	if err := cs.verifySignatureAndData(cdc, misbehaviour, misbehaviour.SignatureOne); err != nil {
		return sdkerrors.Wrap(err, "failed to verify signature one")
	}

	// verify second signature
	if err := cs.verifySignatureAndData(cdc, misbehaviour, misbehaviour.SignatureTwo); err != nil {
		return sdkerrors.Wrap(err, "failed to verify signature two")
	}

	return nil
}

// UpdateState updates the consensus state to the new public key and an incremented sequence.
func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) (exported.ClientState, exported.ConsensusState, error) {
	smHeader, ok := clientMsg.(*Header)
	if !ok {
		return nil, nil, sdkerrors.Wrapf(types.ErrInvalidClientType, "expected %T got %T", Header{}, clientMsg)
	}

	// create new solomachine ConsensusState
	consensusState := &ConsensusState{
		PublicKey:   smHeader.NewPublicKey,
		Diversifier: smHeader.NewDiversifier,
		Timestamp:   smHeader.Timestamp,
	}

	cs.Sequence++
	cs.ConsensusState = consensusState

	clientStore.Set(host.ClientStateKey(), types.MustMarshalClientState(cdc, &cs))

	// set default consensus height with header height
	consensusHeight := smHeader.GetHeight()
	if cs.ClientType() != exported.Localhost {
		clientStore.Set(host.ConsensusStateKey(consensusHeight), types.MustMarshalConsensusState(cdc, consensusState))
	} else {
		consensusHeight = types.GetSelfHeight(ctx)
	}

	// TODO: Should be telemetry and events be emitted from 02-client?
	// The clientID would need to be passed an additional arg here
	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"ibc", "client", "update"},
			1,
			[]metrics.Label{
				telemetry.NewLabel(types.LabelClientType, cs.ClientType()),
				telemetry.NewLabel(types.LabelClientID, "clientID"), // TODO: Should clientID be passed as an arg
				telemetry.NewLabel(types.LabelUpdateType, "msg"),
			},
		)
	}()

	// Marshal the Header as an Any and encode the resulting bytes to hex.
	// This prevents the event value from containing invalid UTF-8 characters
	// which may cause data to be lost when JSON encoding/decoding.
	headerStr := hex.EncodeToString(types.MustMarshalClientMessage(cdc, clientMsg))
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateClient,
			sdk.NewAttribute(types.AttributeKeyClientID, "clientID"),
			sdk.NewAttribute(types.AttributeKeyClientType, cs.ClientType()),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, consensusHeight.String()),
			sdk.NewAttribute(types.AttributeKeyHeader, headerStr),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})

	return &cs, consensusState, nil
}

// CheckForMisbehaviour returns true for type Misbehaviour (passed VerifyClientMessage check), otherwise returns false
func (cs ClientState) CheckForMisbehaviour(_ sdk.Context, _ codec.BinaryCodec, _ sdk.KVStore, clientMsg exported.ClientMessage) bool {
	if _, ok := clientMsg.(*Misbehaviour); ok {
		return true
	}

	return false
}

// UpdateStateOnMisbehaviour updates state upon misbehaviour. This method should only be called on misbehaviour
// as it does not perform any misbehaviour checks.
func (cs ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore) (*ClientState, exported.ConsensusState, error) {
	cs.IsFrozen = true

	clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(cdc, &cs))

	// TODO: Telemetry and events

	return &cs, cs.ConsensusState, nil
}
