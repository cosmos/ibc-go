package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v9/ibc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCQuery queries the chain with a query request and deserializes the response to T
func GRPCQuery[T any](ctx context.Context, chain ibc.Chain, req proto.Message, opts ...grpc.CallOption) (*T, error) {
	path, err := getProtoPath(req)
	if err != nil {
		return nil, err
	}

	return grpcQueryWithMethod[T](ctx, chain, req, path, opts...)
}

// grpcQueryWithMethod queries the chain with a query request with a specific method (grpc path) and deserializes the response to T
func grpcQueryWithMethod[T any](ctx context.Context, chain ibc.Chain, req proto.Message, method string, opts ...grpc.CallOption) (*T, error) {
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
	err = grpcConn.Invoke(ctx, method, req, resp, opts...)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func getProtoPath(req proto.Message) (string, error) {
	typeURL := "/" + proto.MessageName(req)

	switch {
	case strings.Contains(typeURL, "Query"):
		return getQueryProtoPath(typeURL)
	case strings.Contains(typeURL, "cosmos.base.tendermint"):
		return getCmtProtoPath(typeURL)
	default:
		return "", fmt.Errorf("unsupported typeURL: %s", typeURL)
	}
}

func getQueryProtoPath(queryTypeURL string) (string, error) {
	queryIndex := strings.Index(queryTypeURL, "Query")
	if queryIndex == -1 {
		return "", fmt.Errorf("invalid typeURL: %s", queryTypeURL)
	}

	// Add to the index to account for the length of "Query"
	queryIndex += len("Query")

	// Add a slash before the query
	urlWithSlash := queryTypeURL[:queryIndex] + "/" + queryTypeURL[queryIndex:]
	if !strings.HasSuffix(urlWithSlash, "Request") {
		return "", fmt.Errorf("invalid typeURL: %s", queryTypeURL)
	}

	return strings.TrimSuffix(urlWithSlash, "Request"), nil
}

func getCmtProtoPath(cmtTypeURL string) (string, error) {
	cmtIndex := strings.Index(cmtTypeURL, "Get")
	if cmtIndex == -1 {
		return "", fmt.Errorf("invalid typeURL: %s", cmtTypeURL)
	}

	// Add a slash before the commitment
	urlWithSlash := cmtTypeURL[:cmtIndex] + "Service/" + cmtTypeURL[cmtIndex:]
	if !strings.HasSuffix(urlWithSlash, "Request") {
		return "", fmt.Errorf("invalid typeURL: %s", cmtTypeURL)
	}

	return strings.TrimSuffix(urlWithSlash, "Request"), nil
}
