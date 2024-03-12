package testsuite

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	"github.com/cosmos/gogoproto/proto"
)

// Queries the chain with a query request and deserializes the response to T
func GRPCQuery[T any](ctx context.Context, chain ibc.Chain, req proto.Message, opts ...grpc.CallOption) (*T, error) {
	path, err := getProtoPath(req)
	if err != nil {
		return nil, err
	}

	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		chain.GetHostGRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	defer grpcConn.Close()

	resp := new(T)
	err = grpcConn.Invoke(ctx, path, req, resp, opts...)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func getProtoPath(req proto.Message) (string, error) {
	typeURL := "/" + proto.MessageName(req)

	queryIndex := strings.Index(typeURL, "Query")
	if queryIndex == -1 {
		return "", fmt.Errorf("invalid typeURL: %s", typeURL)
	}

	// Add to the index to account for the length of "Query"
	queryIndex += len("Query")

	// Add a slash before the query
	urlWithSlash := typeURL[:queryIndex] + "/" + typeURL[queryIndex:]
	if !strings.HasSuffix(urlWithSlash, "Request") {
		return "", fmt.Errorf("invalid typeURL: %s", typeURL)
	}

	return strings.TrimSuffix(urlWithSlash, "Request"), nil
}

// QueryModuleAccountAddress returns the address of the given module on the given chain.
// Added because interchaintest's method doesn't work.
func (*E2ETestSuite) QueryModuleAccountAddress(ctx context.Context, moduleName string, chain ibc.Chain) (sdk.AccAddress, error) {
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
func (*E2ETestSuite) QueryClientState(ctx context.Context, chain ibc.Chain, clientID string) (ibcexported.ClientState, error) {
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
func (*E2ETestSuite) QueryClientStatus(ctx context.Context, chain ibc.Chain, clientID string) (string, error) {
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
func (*E2ETestSuite) GetValidatorSetByHeight(ctx context.Context, chain ibc.Chain, height uint64) ([]*cmtservice.Validator, error) {
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
func (*E2ETestSuite) QueryBalance(ctx context.Context, chain ibc.Chain, address string, denom string) (math.Int, error) {
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
func (s *E2ETestSuite) QueryChannel(ctx context.Context, chain ibc.Chain, portID, channelID string) (channeltypes.Channel, error) {
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
func (s *E2ETestSuite) QueryCounterPartyPayee(ctx context.Context, chain ibc.Chain, relayerAddress, channelID string) (string, error) {
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
func (s *E2ETestSuite) QueryIncentivizedPacketsForChannel(
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
func (s *E2ETestSuite) QueryFeeEnabledChannel(ctx context.Context, chain ibc.Chain, portID, channelID string) (bool, error) {
	res, err := GRPCQuery[feetypes.QueryFeeEnabledChannelResponse](ctx, chain, &feetypes.QueryFeeEnabledChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	})

	if err != nil {
		return false, err
	}
	return res.FeeEnabled, nil
}
