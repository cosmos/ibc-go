package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/interchaintest/v10/ibc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/protobuf/proto"

	msgv1 "cosmossdk.io/api/cosmos/msg/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"

	"github.com/cosmos/ibc-go/e2e/testvalues"
)

var queryReqToPath = make(map[string]string)

func PopulateQueryReqToPath(ctx context.Context, chain ibc.Chain) error {
	if !testvalues.ReflectionServiceFeatureReleases.IsSupported(chain.Config().Images[0].Version) {
		return nil
	}
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
	var path string
	if testvalues.ReflectionServiceFeatureReleases.IsSupported(chain.Config().Images[0].Version) {
		var ok bool
		path, ok = queryReqToPath[proto.MessageName(req)]
		if !ok {
			return nil, fmt.Errorf("no path found for %s", proto.MessageName(req))
		}
	} else {
		var err error
		path, err = getProtoPath(req)
		if err != nil {
			return nil, err
		}
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

// TODO: Remove all of the below when v6 -> v7 upgrade is not supported anymore:
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
