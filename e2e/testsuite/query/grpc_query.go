package query

import (
	"context"
	"fmt"

	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/protobuf/proto"

	msgv1 "cosmossdk.io/api/cosmos/msg/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
)

var queryReqToPath = make(map[string]string)

func PopulateQueryReqToPath(ctx context.Context, chain ibc.Chain) error {
	resp, err := queryFileDescriptors(ctx, chain)
	if err != nil {
		return err
	}

	for _, fileDescriptor := range resp.Files {
		for _, service := range fileDescriptor.GetService() {
			// Skip services that are annotated with the "cosmos.msg.v1.service" option.
			if ext := pb.GetExtension(service.GetOptions(), msgv1.E_Service); ext != nil {
				if ok, extBool := ext.(bool); ok && extBool {
					continue
				}
			}

			for _, method := range service.GetMethod() {
				// trim the first character from input which is a dot
				queryReqToPath[method.GetInputType()[1:]] = fileDescriptor.GetPackage() + "." + service.GetName() + "/" + method.GetName()
			}
		}
	}

	return nil
}

// GRPCQuery queries the chain with a query request and deserializes the response to T
func GRPCQuery[T any](ctx context.Context, chain ibc.Chain, req proto.Message, opts ...grpc.CallOption) (*T, error) {
	path, ok := queryReqToPath[proto.MessageName(req)]
	if !ok {
		return nil, fmt.Errorf("no path found for %s", proto.MessageName(req))
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

func queryFileDescriptors(ctx context.Context, chain ibc.Chain) (*reflectionv1.FileDescriptorsResponse, error) {
	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		chain.GetHostGRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	defer grpcConn.Close()

	resp := new(reflectionv1.FileDescriptorsResponse)
	err = grpcConn.Invoke(
		ctx, reflectionv1.ReflectionService_FileDescriptors_FullMethodName,
		&reflectionv1.FileDescriptorsRequest{}, resp,
	)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
