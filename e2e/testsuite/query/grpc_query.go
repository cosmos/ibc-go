package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
