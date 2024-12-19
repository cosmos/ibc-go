package query

import (
	"context"
	"errors"
	"sort"

	"github.com/strangelove-ventures/interchaintest/v9/ibc"

	"cosmossdk.io/math"
	banktypes "cosmossdk.io/x/bank/types"

	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	controllertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
)

const queryPathTransferDenoms = "/ibc.applications.transfer.v2.QueryV2/Denoms"

// ModuleAccountAddress returns the address of the given module on the given chain.
// Added because interchaintest's method doesn't work.
func ModuleAccountAddress(ctx context.Context, moduleName string, chain ibc.Chain) (sdk.AccAddress, error) {
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
		return nil, errors.New("account is not a module account")
	}
	if govAccount.GetAddress().String() == "" {
		return nil, errors.New("module account address is empty")
	}

	return govAccount.GetAddress(), nil
}

// ClientState queries the client state on the given chain for the provided clientID.
func ClientState(ctx context.Context, chain ibc.Chain, clientID string) (ibcexported.ClientState, error) {
	clientStateResp, err := GRPCQuery[clienttypes.QueryClientStateResponse](ctx, chain, &clienttypes.QueryClientStateRequest{
		ClientId: clientID,
	})
	if err != nil {
		return nil, err
	}

	clientStateAny := clientStateResp.ClientState

	cfg := chain.Config().EncodingConfig
	var clientState ibcexported.ClientState
	err = cfg.InterfaceRegistry.UnpackAny(clientStateAny, &clientState)
	if err != nil {
		return nil, err
	}

	return clientState, nil
}

// ClientStatus queries the status of the client by clientID
func ClientStatus(ctx context.Context, chain ibc.Chain, clientID string) (string, error) {
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

// Balance returns the balance of a specific denomination for a given account by address.
func Balance(ctx context.Context, chain ibc.Chain, address string, denom string) (math.Int, error) {
	res, err := GRPCQuery[banktypes.QueryBalanceResponse](ctx, chain, &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	})
	if err != nil {
		return math.Int{}, err
	}
	return res.Balance.Amount, nil
}

// Channel queries the channel on a given chain for the provided portID and channelID
func Channel(ctx context.Context, chain ibc.Chain, portID, channelID string) (channeltypes.Channel, error) {
	res, err := GRPCQuery[channeltypes.QueryChannelResponse](ctx, chain, &channeltypes.QueryChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	if err != nil {
		return channeltypes.Channel{}, err
	}
	return *res.Channel, nil
}

// CounterPartyPayee queries the counterparty payee of the given chain and relayer address on the specified channel.
func CounterPartyPayee(ctx context.Context, chain ibc.Chain, relayerAddress, channelID string) (string, error) {
	res, err := GRPCQuery[feetypes.QueryCounterpartyPayeeResponse](ctx, chain, &feetypes.QueryCounterpartyPayeeRequest{
		ChannelId: channelID,
		Relayer:   relayerAddress,
	})
	if err != nil {
		return "", err
	}
	return res.CounterpartyPayee, nil
}

// IncentivizedPacketsForChannel queries the incentivized packets on the specified channel.
func IncentivizedPacketsForChannel(
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

// FeeEnabledChannel queries the fee-enabled status of a channel.
func FeeEnabledChannel(ctx context.Context, chain ibc.Chain, portID, channelID string) (bool, error) {
	res, err := GRPCQuery[feetypes.QueryFeeEnabledChannelResponse](ctx, chain, &feetypes.QueryFeeEnabledChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	if err != nil {
		return false, err
	}
	return res.FeeEnabled, nil
}

// TotalEscrowForDenom queries the total amount of tokens in escrow for a denom
func TotalEscrowForDenom(ctx context.Context, chain ibc.Chain, denom string) (sdk.Coin, error) {
	res, err := GRPCQuery[transfertypes.QueryTotalEscrowForDenomResponse](ctx, chain, &transfertypes.QueryTotalEscrowForDenomRequest{
		Denom: denom,
	})
	if err != nil {
		return sdk.Coin{}, err
	}
	return res.Amount, nil
}

// PacketAcknowledgements queries the packet acknowledgements on the given chain for the provided channel (optional) list of packet commitment sequences.
func PacketAcknowledgements(ctx context.Context, chain ibc.Chain, portID, channelID string, packetCommitmentSequences []uint64) ([]*channeltypes.PacketState, error) {
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

// UpgradeError queries the upgrade error on the given chain for the provided channel.
func UpgradeError(ctx context.Context, chain ibc.Chain, portID, channelID string) (channeltypes.ErrorReceipt, error) {
	res, err := GRPCQuery[channeltypes.QueryUpgradeErrorResponse](ctx, chain, &channeltypes.QueryUpgradeErrorRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	if err != nil {
		return channeltypes.ErrorReceipt{}, err
	}
	return res.ErrorReceipt, nil
}

// InterchainAccount queries the interchain account for the given owner and connectionID.
func InterchainAccount(ctx context.Context, chain ibc.Chain, address, connectionID string) (string, error) {
	res, err := GRPCQuery[controllertypes.QueryInterchainAccountResponse](ctx, chain, &controllertypes.QueryInterchainAccountRequest{
		Owner:        address,
		ConnectionId: connectionID,
	})
	if err != nil {
		return "", err
	}
	return res.Address, nil
}

func TransferDenoms(ctx context.Context, chain ibc.Chain) (*transfertypes.QueryDenomsResponse, error) {
	return grpcQueryWithMethod[transfertypes.QueryDenomsResponse](ctx, chain, &transfertypes.QueryDenomsRequest{}, queryPathTransferDenoms)
}
