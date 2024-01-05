package keeper

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	gogotypes "github.com/cosmos/gogoproto/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/merkle"
	cmbytes "github.com/cometbft/cometbft/libs/bytes"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

// emitCreateClientEvent emits a create client event
func emitCreateClientEvent(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientState.ClientType()),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, clientState.GetLatestHeight().String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// cdcEncode returns nil if the input is nil, otherwise returns
// proto.Marshal(<type>Value{Value: item}).
func cdcEncode(item interface{}) []byte {
	if item != nil {
		switch item := item.(type) {
		case string:
			i := gogotypes.StringValue{
				Value: item,
			}
			bz, err := i.Marshal()
			if err != nil {
				return nil
			}
			return bz
		case int64:
			i := gogotypes.Int64Value{
				Value: item,
			}
			bz, err := i.Marshal()
			if err != nil {
				return nil
			}
			return bz
		case cmbytes.HexBytes:
			i := gogotypes.BytesValue{
				Value: item,
			}
			bz, err := i.Marshal()
			if err != nil {
				return nil
			}
			return bz
		default:
			return nil
		}
	}

	return nil
}

func headerClientMsgToHashBz(clientMsg exported.ClientMessage) []byte {
	h := clientMsg.(*ibctmtypes.Header).Header
	fmt.Println("valhash", h.ValidatorsHash)
	if len(h.ValidatorsHash) == 0 {
		return nil
	}
	hbz, err := h.Version.Marshal()
	if err != nil {
		return nil
	}

	pbt, err := gogotypes.StdTimeMarshal(h.Time)
	if err != nil {
		return nil
	}

	pbbi := h.LastBlockId
	bzbi, err := pbbi.Marshal()
	if err != nil {
		return nil
	}

	return merkle.HashFromByteSlices([][]byte{
		hbz,
		cdcEncode(h.ChainID),
		cdcEncode(h.Height),
		pbt,
		bzbi,
		cdcEncode(h.LastCommitHash),
		cdcEncode(h.DataHash),
		cdcEncode(h.ValidatorsHash),
		cdcEncode(h.NextValidatorsHash),
		cdcEncode(h.ConsensusHash),
		cdcEncode(h.AppHash),
		cdcEncode(h.LastResultsHash),
		cdcEncode(h.EvidenceHash),
		cdcEncode(h.ProposerAddress),
	})
}

// emitUpdateClientEvent emits an update client event
func emitUpdateClientEvent(ctx sdk.Context, clientID string, clientType string, consensusHeights []exported.Height, _ codec.BinaryCodec, clientMsg exported.ClientMessage) {
	// calculating header hash, reference https://github.com/cometbft/cometbft/blob/c88d5512350f6671c92051cf5ebbbe9841f644cb/types/block.go#L439C28
	headerHashBz := headerClientMsgToHashBz(clientMsg)
	headerHashAttr := hex.EncodeToString(headerHashBz)

	var consensusHeightAttr string
	if len(consensusHeights) != 0 {
		consensusHeightAttr = consensusHeights[0].String()
	}

	consensusHeightsAttr := make([]string, len(consensusHeights))
	for i, height := range consensusHeights {
		consensusHeightsAttr[i] = height.String()
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientType),
			// Deprecated: AttributeKeyConsensusHeight is deprecated and will be removed in a future release.
			// Please use AttributeKeyConsensusHeights instead.
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, consensusHeightAttr),
			sdk.NewAttribute(types.AttributeKeyConsensusHeights, strings.Join(consensusHeightsAttr, ",")),
			sdk.NewAttribute(types.AttributeKeyHeaderHash, headerHashAttr),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitUpgradeClientEvent emits an upgrade client event
func emitUpgradeClientEvent(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpgradeClient,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientState.ClientType()),
			sdk.NewAttribute(types.AttributeKeyConsensusHeight, clientState.GetLatestHeight().String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitSubmitMisbehaviourEvent emits a client misbehaviour event
func emitSubmitMisbehaviourEvent(ctx sdk.Context, clientID string, clientState exported.ClientState) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSubmitMisbehaviour,
			sdk.NewAttribute(types.AttributeKeyClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientState.ClientType()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitRecoverClientEvent emits a recover client event
func emitRecoverClientEvent(ctx sdk.Context, clientID, clientType string) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecoverClient,
			sdk.NewAttribute(types.AttributeKeySubjectClientID, clientID),
			sdk.NewAttribute(types.AttributeKeyClientType, clientType),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// emitScheduleIBCSoftwareUpgradeEvent emits a schedule IBC software upgrade event
func emitScheduleIBCSoftwareUpgradeEvent(ctx sdk.Context, title string, height int64) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeScheduleIBCSoftwareUpgrade,
			sdk.NewAttribute(types.AttributeKeyUpgradePlanTitle, title),
			sdk.NewAttribute(types.AttributeKeyUpgradePlanHeight, fmt.Sprintf("%d", height)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}

// EmitUpgradeChainEvent emits an upgrade chain event.
func EmitUpgradeChainEvent(ctx sdk.Context, height int64) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpgradeChain,
			sdk.NewAttribute(types.AttributeKeyUpgradePlanHeight, strconv.FormatInt(height, 10)),
			sdk.NewAttribute(types.AttributeKeyUpgradeStore, upgradetypes.StoreKey), // which store to query proof of consensus state from
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}
