package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	icqtypes "github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (k Keeper) SendQuery(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	chanCap *capabilitytypes.Capability,
	reqs []abci.RequestQuery,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) (uint64, error) {
	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	bz, err := types.SerializeCosmosQuery(reqs)
	if err != nil {
		return 0, err
	}
	icqPacketData := types.InterchainQueryPacketData{
		Data: bz,
	}

	return k.createOutgoingPacket(ctx, sourcePort, sourceChannel, destinationPort, destinationChannel, chanCap, icqPacketData, timeoutTimestamp)
}

func (k Keeper) createOutgoingPacket(
	ctx sdk.Context,
	sourcePort,
	sourceChannel,
	destinationPort,
	destinationChannel string,
	chanCap *capabilitytypes.Capability,
	icqPacketData types.InterchainQueryPacketData,
	timeoutTimestamp uint64,
) (uint64, error) {
	if err := icqPacketData.ValidateBasic(); err != nil {
		return 0, sdkerrors.Wrap(err, "invalid interchain query packet data")
	}

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, sdkerrors.Wrapf(channeltypes.ErrSequenceSendNotFound, "failed to retrieve next sequence send for channel %s on port %s", sourceChannel, sourcePort)
	}

	packet := channeltypes.NewPacket(
		icqPacketData.GetBytes(),
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		clienttypes.ZeroHeight(),
		timeoutTimestamp,
	)

	if err := k.ics4Wrapper.SendPacket(ctx, chanCap, packet); err != nil {
		return 0, err
	}

	return packet.Sequence, nil
}

// OnRecvPacket handles a given interchain queries packet on a destination host chain.
// If the transaction is successfully executed, the transaction response bytes will be returned.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) ([]byte, error) {
	var data icqtypes.InterchainQueryPacketData

	if err := icqtypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		// UnmarshalJSON errors are indeterminate and therefore are not wrapped and included in failed acks
		return nil, sdkerrors.Wrapf(icqtypes.ErrUnknownDataType, "cannot unmarshal ICQ packet data")
	}

	reqs, err := types.DeserializeCosmosQuery(data.GetData())
	if err != nil {
		return nil, err
	}

	response, err := k.executeQuery(ctx, reqs)
	if err != nil {
		return nil, err
	}
	return response, err
}

func (k Keeper) executeQuery(ctx sdk.Context, reqs []abci.RequestQuery) ([]byte, error) {
	resps := make([]abci.ResponseQuery, len(reqs))
	for i, req := range reqs {
		if err := k.authenticateQuery(ctx, req); err != nil {
			return nil, err
		}

		resp := k.querier.Query(req)
		// Remove non-deterministic fields from response
		resps[i] = abci.ResponseQuery{
			Code:   resp.Code,
			Index:  resp.Index,
			Key:    resp.Key,
			Value:  resp.Value,
			Height: resp.Height,
		}
	}

	bz, err := types.SerializeCosmosResponse(resps)
	if err != nil {
		return nil, err
	}
	ack := icqtypes.InterchainQueryPacketAck{
		Data: bz,
	}
	data, err := icqtypes.ModuleCdc.MarshalJSON(&ack)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to marshal tx data")
	}

	return data, nil
}

// authenticateQuery ensures the provided query request is in the whitelist.
func (k Keeper) authenticateQuery(ctx sdk.Context, q abci.RequestQuery) error {
	allowQueries := k.GetAllowQueries(ctx)
	if !types.ContainsQueryPath(allowQueries, q.Path) {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "query path not allowed: %s", q.Path)
	}
	if !(q.Height == 0 || q.Height == ctx.BlockHeight()) {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "query height not allowed: %d", q.Height)
	}
	if q.Prove {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "query proof not allowed")
	}

	return nil
}
