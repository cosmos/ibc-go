package types

import (
	"fmt"
	"strings"

	gogoproto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	queryv1 "cosmossdk.io/api/cosmos/query/v1"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"

	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

// IsModuleSafeQuery checks whether the method with the given grpcServicePath has the
// `(cosmos.query.v1.module_query_safe) = true` proto annotation.
//
// For example, `/cosmos.bank.v1beta1.Query/Balance` is module safe, but
// `/cosmos.reflection.v1.ReflectionService/FileDescriptors` is not.
func IsModuleQuerySafe(logger log.Logger, grpcServicePath string) bool {
	methodPath, err := toMethodPath(grpcServicePath)
	if err != nil {
		logger.Debug("failed to convert gRPC service path to method path", "grpcServicePath", grpcServicePath, "err", err)
		return false
	}

	protoFiles, err := gogoproto.MergedRegistry()
	if err != nil {
		// This should never happen
		panic(err)
	}
	if protoFiles == nil {
		protoFiles = protoregistry.GlobalFiles
	}

	fullName := protoreflect.FullName(methodPath)
	if !fullName.IsValid() {
		logger.Debug("invalid method path", "methodPath", methodPath)
		return false
	}

	serviceDesc, err := protoFiles.FindDescriptorByName(fullName)
	if err != nil {
		logger.Debug("failed to find the descriptor", "methodPath", methodPath, "err", err)
		return false
	}

	methodDesc, ok := serviceDesc.(protoreflect.MethodDescriptor)
	if !ok {
		logger.Debug("invalid method descriptor", "methodPath", methodPath)
		return false
	}

	return isModuleQuerySafe(methodDesc)
}

// isModuleQuerySafe checks whether the service has the
// `(cosmos.query.v1.module_query_safe) = true` proto annotation.
func isModuleQuerySafe(sd protoreflect.MethodDescriptor) bool {
	ext := proto.GetExtension(sd.Options(), queryv1.E_ModuleQuerySafe)
	isModuleQuerySafe, ok := ext.(bool)
	if !ok {
		return false
	}

	return isModuleQuerySafe
}

// toMethodPath converts a gRPC service path to a protobuf method path.
//
// For example, `/cosmos.bank.v1beta1.Query/Balance` becomes `cosmos.bank.v1beta1.Query.Balance`.
func toMethodPath(grpcServicePath string) (string, error) {
	if !strings.HasPrefix(grpcServicePath, "/") {
		return "", errorsmod.Wrap(ibcerrors.ErrInvalidRequest, fmt.Sprintf("invalid gRPC service path: %s", grpcServicePath))
	}

	// Remove the leading slash
	grpcServicePath = grpcServicePath[1:]

	// Convert the remaining slashes to dots
	return strings.ReplaceAll(grpcServicePath, "/", "."), nil
}
