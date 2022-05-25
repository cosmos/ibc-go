package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/apps/icq/host/types"
	icqtypes "github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// OnRecvPacket handles a given interchain accounts packet on a destination host chain.
// If the transaction is successfully executed, the transaction response bytes will be returned.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) ([]byte, error) {
	var data icqtypes.InterchainQueryPacketData

	if err := icqtypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		// UnmarshalJSON errors are indeterminate and therefore are not wrapped and included in failed acks
		return nil, sdkerrors.Wrapf(icqtypes.ErrUnknownDataType, "cannot unmarshal ICQ packet data")
	}

	switch data.Type {
	case icqtypes.QUERY:
		response, err := k.executeQuery(ctx, data.Request)
		if err != nil {
			return nil, err
		}
		return response, err
	default:
		return nil, icqtypes.ErrUnknownDataType
	}
}

func (k Keeper) executeQuery(ctx sdk.Context, q abci.RequestQuery) ([]byte, error) {
	if err := k.authenticateQuery(ctx, q); err != nil {
		return nil, err
	}

	response := k.querier.Query(q)
	// Remove non-deterministic fields from response
	response = abci.ResponseQuery{
		Code:     response.Code,
		Index:    response.Index,
		Key:      response.Key,
		Value:    response.Value,
		ProofOps: response.ProofOps,
		Height:   response.Height,
	}

	ack := icqtypes.InterchainQueryPacketAck{
		Response: response,
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
	if !k.GetAllowHeight(ctx) && q.Height != 0 {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "query height not allowed: %d", q.Height)
	}
	if !k.GetAllowProof(ctx) && q.Prove {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "query proof not allowed")
	}

	return nil
}
