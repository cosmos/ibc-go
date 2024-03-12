package query

import (
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/math"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// QueryModuleAccountAddress returns the address of the given module on the given chain.
// Added because interchaintest's method doesn't work.
func QueryModuleAccountAddress(ctx context.Context, moduleName string, chain ibc.Chain) (sdk.AccAddress, error) {
	modAccResp, err := GRPCQuery[authtypes.QueryModuleAccountByNameResponse](
		ctx, chain, &authtypes.QueryModuleAccountByNameRequest{Name: moduleName},
	)
	if err != nil {
		return nil, err
	}

	cfg := chain.Config().EncodingConfig
	var account sdk.AccountI
	err = cfg.InterfaceRegistry.UnpackAny(modAccResp.Account, &account)
	if err != nil {
		return nil, err
	}

	govAccount, ok := account.(sdk.ModuleAccountI)
	if !ok {
		return nil, fmt.Errorf("account is not a module account")
	}
	if govAccount.GetAddress().String() == "" {
		return nil, fmt.Errorf("module account address is empty")
	}

	return govAccount.GetAddress(), nil
}

// QueryClientState queries the client state on the given chain for the provided clientID.
func QueryClientState(ctx context.Context, chain ibc.Chain, clientID string) (ibcexported.ClientState, error) {
	clientStateResp, err := GRPCQuery[clienttypes.QueryClientStateResponse](ctx, chain, &clienttypes.QueryClientStateRequest{
		ClientId: ibctesting.FirstClientID,
	})
	if err != nil {
		return nil, err
	}

	clientStateAny := clientStateResp.ClientState

	clientState, err := clienttypes.UnpackClientState(clientStateAny)
	if err != nil {
		return nil, err
	}

	return clientState, nil
}

// QueryClientStatus queries the status of the client by clientID
func QueryClientStatus(ctx context.Context, chain ibc.Chain, clientID string) (string, error) {
	clientStatusResp, err := GRPCQuery[clienttypes.QueryClientStatusResponse](ctx, chain, &clienttypes.QueryClientStatusRequest{
		ClientId: clientID,
	})
	if err != nil {
		return "", err
	}

	return clientStatusResp.Status, nil
}

// GetValidatorSetByHeight returns the validators of the given chain at the specified height. The returned validators
// are sorted by address.
func GetValidatorSetByHeight(ctx context.Context, chain ibc.Chain, height uint64) ([]*cmtservice.Validator, error) {
	res, err := GRPCQuery[cmtservice.GetValidatorSetByHeightResponse](ctx, chain, &cmtservice.GetValidatorSetByHeightRequest{
		Height: int64(height),
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(res.Validators, func(i, j int) bool {
		return res.Validators[i].Address < res.Validators[j].Address
	})

	return res.Validators, nil
}

// QueryBalance returns the balance of a specific denomination for a given account by address.
func QueryBalance(ctx context.Context, chain ibc.Chain, address string, denom string) (math.Int, error) {
	res, err := GRPCQuery[banktypes.QueryBalanceResponse](ctx, chain, &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	})
	if err != nil {
		return math.Int{}, err
	}

	return res.Balance.Amount, nil
}

// QueryChannel queries the channel on a given chain for the provided portID and channelID
func QueryChannel(ctx context.Context, chain ibc.Chain, portID, channelID string) (channeltypes.Channel, error) {
	res, err := GRPCQuery[channeltypes.QueryChannelResponse](ctx, chain, &channeltypes.QueryChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	if err != nil {
		return channeltypes.Channel{}, err
	}

	return *res.Channel, nil
}

// QueryCounterPartyPayee queries the counterparty payee of the given chain and relayer address on the specified channel.
func QueryCounterPartyPayee(ctx context.Context, chain ibc.Chain, relayerAddress, channelID string) (string, error) {
	res, err := GRPCQuery[feetypes.QueryCounterpartyPayeeResponse](ctx, chain, &feetypes.QueryCounterpartyPayeeRequest{
		ChannelId: channelID,
		Relayer:  relayerAddress,
	})
	if err != nil {
		return "", err
	}

	return res.CounterpartyPayee, nil
}

// QueryIncentivizedPacketsForChannel queries the incentivized packets on the specified channel.
func QueryIncentivizedPacketsForChannel(
	ctx context.Context,
	chain ibc.Chain,
	portID,
	channelID string,
) ([]*feetypes.IdentifiedPacketFees, error) {
	res, err := GRPCQuery[feetypes.QueryIncentivizedPacketsForChannelResponse](ctx, chain, &feetypes.QueryIncentivizedPacketsForChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	})

	if err != nil {
		return nil, err
	}
	return res.IncentivizedPackets, err
}

// QueryFeeEnabledChannel queries the fee-enabled status of a channel.
func QueryFeeEnabledChannel(ctx context.Context, chain ibc.Chain, portID, channelID string) (bool, error) {
	res, err := GRPCQuery[feetypes.QueryFeeEnabledChannelResponse](ctx, chain, &feetypes.QueryFeeEnabledChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	})

	if err != nil {
		return false, err
	}
	return res.FeeEnabled, nil
}

// QueryTotalEscrowForDenom queries the total amount of tokens in escrow for a denom
func QueryTotalEscrowForDenom(ctx context.Context, chain ibc.Chain, denom string) (sdk.Coin, error) {
	res, err := GRPCQuery[transfertypes.QueryTotalEscrowForDenomResponse](ctx, chain, &transfertypes.QueryTotalEscrowForDenomRequest{
		Denom: denom,
	})
	if err != nil {
		return sdk.Coin{}, err
	}

	return res.Amount, nil
}

// QueryPacketAcknowledgements queries the packet acknowledgements on the given chain for the provided channel (optional) list of packet commitment sequences.
func QueryPacketAcknowledgements(ctx context.Context, chain ibc.Chain, portID, channelID string, packetCommitmentSequences []uint64) ([]*channeltypes.PacketState, error) {
	res, err := GRPCQuery[channeltypes.QueryPacketAcknowledgementsResponse](ctx, chain, &channeltypes.QueryPacketAcknowledgementsRequest{
		PortId:                    portID,
		ChannelId:                 channelID,
		PacketCommitmentSequences: packetCommitmentSequences,
	})
	if err != nil {
		return nil, err
	}

	return res.Acknowledgements, nil
}

// QueryUpgradeError queries the upgrade error on the given chain for the provided channel.
func QueryUpgradeError(ctx context.Context, chain ibc.Chain, portID, channelID string) (channeltypes.ErrorReceipt, error) {
	res, err := GRPCQuery[channeltypes.QueryUpgradeErrorResponse](ctx, chain, &channeltypes.QueryUpgradeErrorRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	if err != nil {
		return channeltypes.ErrorReceipt{}, err
	}
	return res.ErrorReceipt, nil
}
