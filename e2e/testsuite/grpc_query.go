package testsuite

import (
	"context"
	"fmt"
	"strings"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

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
	typeUrl := "/" + proto.MessageName(req)

	queryIndex := strings.Index(typeUrl, "Query")
	if queryIndex == -1 {
		return "", fmt.Errorf("invalid typeUrl: %s", typeUrl)
	}

	// Add to the index to account for the length of "Query"
	queryIndex += len("Query")

	// Add a slash before the query
	urlWithSlash := typeUrl[:queryIndex] + "/" + typeUrl[queryIndex:]
	if !strings.HasSuffix(urlWithSlash, "Request") {
		return "", fmt.Errorf("invalid typeUrl: %s", typeUrl)
	}

	return strings.TrimSuffix(urlWithSlash, "Request"), nil
}

// QueryModuleAccountAddress returns the address of the given module on the given chain.
// Added because interchaintest's method doesn't work.
func (s *E2ETestSuite) QueryModuleAccountAddress(ctx context.Context, moduleName string, chain ibc.Chain) (sdk.AccAddress, error) {
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
