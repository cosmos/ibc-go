package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v9/internal/validate"
	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// IncentivizedPackets implements the Query/IncentivizedPackets gRPC method
func (k Keeper) IncentivizedPackets(ctx context.Context, req *types.QueryIncentivizedPacketsRequest) (*types.QueryIncentivizedPacketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	var identifiedPackets []types.IdentifiedPacketFees

	store := prefix.NewStore(runtime.KVStoreAdapter(k.KVStoreService.OpenKVStore(ctx)), []byte(types.FeesInEscrowPrefix))
	pagination, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		packetID, err := types.ParseKeyFeesInEscrow(types.FeesInEscrowPrefix + string(key))
		if err != nil {
			return err
		}

		packetFees := k.MustUnmarshalFees(value)
		identifiedPackets = append(identifiedPackets, types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees))
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryIncentivizedPacketsResponse{
		IncentivizedPackets: identifiedPackets,
		Pagination:          pagination,
	}, nil
}

// IncentivizedPacket implements the Query/IncentivizedPacket gRPC method
func (k Keeper) IncentivizedPacket(ctx context.Context, req *types.QueryIncentivizedPacketRequest) (*types.QueryIncentivizedPacketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	feesInEscrow, exists := k.GetFeesInEscrow(ctx, req.PacketId)
	if !exists {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrFeeNotFound, "channel: %s, port: %s, sequence: %d", req.PacketId.ChannelId, req.PacketId.PortId, req.PacketId.Sequence).Error())
	}

	return &types.QueryIncentivizedPacketResponse{
		IncentivizedPacket: types.NewIdentifiedPacketFees(req.PacketId, feesInEscrow.PacketFees),
	}, nil
}

// IncentivizedPacketsForChannel implements the Query/IncentivizedPacketsForChannel gRPC method
func (k Keeper) IncentivizedPacketsForChannel(ctx context.Context, req *types.QueryIncentivizedPacketsForChannelRequest) (*types.QueryIncentivizedPacketsForChannelResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := validate.GRPCRequest(req.PortId, req.ChannelId); err != nil {
		return nil, err
	}

	if !k.channelKeeper.HasChannel(ctx, req.PortId, req.ChannelId) {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port-id: %s, channel-id %s", req.PortId, req.ChannelId).Error(),
		)
	}

	var packets []*types.IdentifiedPacketFees
	keyPrefix := types.KeyFeesInEscrowChannelPrefix(req.PortId, req.ChannelId)
	store := prefix.NewStore(runtime.KVStoreAdapter(k.KVStoreService.OpenKVStore(ctx)), keyPrefix)
	pagination, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		packetID, err := types.ParseKeyFeesInEscrow(string(keyPrefix) + string(key))
		if err != nil {
			return err
		}

		packetFees := k.MustUnmarshalFees(value)

		identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)
		packets = append(packets, &identifiedPacketFees)

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryIncentivizedPacketsForChannelResponse{
		IncentivizedPackets: packets,
		Pagination:          pagination,
	}, nil
}

// TotalRecvFees implements the Query/TotalRecvFees gRPC method
func (k Keeper) TotalRecvFees(ctx context.Context, req *types.QueryTotalRecvFeesRequest) (*types.QueryTotalRecvFeesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	feesInEscrow, found := k.GetFeesInEscrow(ctx, req.PacketId)
	if !found {
		return nil, status.Errorf(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrFeeNotFound, "channel: %s, port: %s, sequence: %d", req.PacketId.ChannelId, req.PacketId.PortId, req.PacketId.Sequence).Error(),
		)
	}

	var recvFees sdk.Coins
	for _, packetFee := range feesInEscrow.PacketFees {
		recvFees = recvFees.Add(packetFee.Fee.RecvFee...)
	}

	return &types.QueryTotalRecvFeesResponse{
		RecvFees: recvFees,
	}, nil
}

// TotalAckFees implements the Query/TotalAckFees gRPC method
func (k Keeper) TotalAckFees(ctx context.Context, req *types.QueryTotalAckFeesRequest) (*types.QueryTotalAckFeesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	feesInEscrow, found := k.GetFeesInEscrow(ctx, req.PacketId)
	if !found {
		return nil, status.Errorf(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrFeeNotFound, "channel: %s, port: %s, sequence: %d", req.PacketId.ChannelId, req.PacketId.PortId, req.PacketId.Sequence).Error(),
		)
	}

	var ackFees sdk.Coins
	for _, packetFee := range feesInEscrow.PacketFees {
		ackFees = ackFees.Add(packetFee.Fee.AckFee...)
	}

	return &types.QueryTotalAckFeesResponse{
		AckFees: ackFees,
	}, nil
}

// TotalTimeoutFees implements the Query/TotalTimeoutFees gRPC method
func (k Keeper) TotalTimeoutFees(ctx context.Context, req *types.QueryTotalTimeoutFeesRequest) (*types.QueryTotalTimeoutFeesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	feesInEscrow, found := k.GetFeesInEscrow(ctx, req.PacketId)
	if !found {
		return nil, status.Errorf(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrFeeNotFound, "channel: %s, port: %s, sequence: %d", req.PacketId.ChannelId, req.PacketId.PortId, req.PacketId.Sequence).Error(),
		)
	}

	var timeoutFees sdk.Coins
	for _, packetFee := range feesInEscrow.PacketFees {
		timeoutFees = timeoutFees.Add(packetFee.Fee.TimeoutFee...)
	}

	return &types.QueryTotalTimeoutFeesResponse{
		TimeoutFees: timeoutFees,
	}, nil
}

// Payee implements the Query/Payee gRPC method and returns the registered payee address to which packet fees are paid out
func (k Keeper) Payee(ctx context.Context, req *types.QueryPayeeRequest) (*types.QueryPayeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	payeeAddr, found := k.GetPayeeAddress(ctx, req.Relayer, req.ChannelId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "payee address not found for address: %s on channel: %s", req.Relayer, req.ChannelId)
	}

	return &types.QueryPayeeResponse{
		PayeeAddress: payeeAddr,
	}, nil
}

// CounterpartyPayee implements the Query/CounterpartyPayee gRPC method and returns the registered counterparty payee address for forward relaying
func (k Keeper) CounterpartyPayee(ctx context.Context, req *types.QueryCounterpartyPayeeRequest) (*types.QueryCounterpartyPayeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	counterpartyPayeeAddr, found := k.GetCounterpartyPayeeAddress(ctx, req.Relayer, req.ChannelId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "counterparty payee address not found for address: %s on channel: %s", req.Relayer, req.ChannelId)
	}

	return &types.QueryCounterpartyPayeeResponse{
		CounterpartyPayee: counterpartyPayeeAddr,
	}, nil
}

// FeeEnabledChannels implements the Query/FeeEnabledChannels gRPC method and returns a list of fee enabled channels
func (k Keeper) FeeEnabledChannels(ctx context.Context, req *types.QueryFeeEnabledChannelsRequest) (*types.QueryFeeEnabledChannelsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	var feeEnabledChannels []types.FeeEnabledChannel
	store := prefix.NewStore(runtime.KVStoreAdapter(k.KVStoreService.OpenKVStore(ctx)), []byte(types.FeeEnabledKeyPrefix))
	pagination, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		portID, channelID, err := types.ParseKeyFeeEnabled(types.FeeEnabledKeyPrefix + string(key))
		if err != nil {
			return err
		}

		feeEnabledChannel := types.FeeEnabledChannel{
			PortId:    portID,
			ChannelId: channelID,
		}

		feeEnabledChannels = append(feeEnabledChannels, feeEnabledChannel)

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryFeeEnabledChannelsResponse{
		FeeEnabledChannels: feeEnabledChannels,
		Pagination:         pagination,
	}, nil
}

// FeeEnabledChannel implements the Query/FeeEnabledChannel gRPC method and returns true if the provided
// port and channel identifiers belong to a fee enabled channel
func (k Keeper) FeeEnabledChannel(ctx context.Context, req *types.QueryFeeEnabledChannelRequest) (*types.QueryFeeEnabledChannelResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := validate.GRPCRequest(req.PortId, req.ChannelId); err != nil {
		return nil, err
	}

	if !k.HasChannel(ctx, req.PortId, req.ChannelId) {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", req.PortId, req.ChannelId).Error(),
		)
	}

	isFeeEnabled := k.IsFeeEnabled(ctx, req.PortId, req.ChannelId)

	return &types.QueryFeeEnabledChannelResponse{
		FeeEnabled: isFeeEnabled,
	}, nil
}
