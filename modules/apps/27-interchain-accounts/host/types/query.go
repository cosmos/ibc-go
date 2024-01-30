package types

import (
	gogoproto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	queryv1 "cosmossdk.io/api/cosmos/query/v1"
	errorsmod "cosmossdk.io/errors"

	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

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

// IsModuleSafeQuery checks whether the method with the given methodPath has the
// `(cosmos.query.v1.module_query_safe) = true` proto annotation.
func IsModuleQuerySafe(methodPath string) (bool, error) {
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
		return false, errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "invalid query method path")
	}

	serviceDesc, err := protoFiles.FindDescriptorByName(fullName)
	if err != nil {
		return false, errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "failed to find descriptor")
	}

	return isModuleQuerySafe(serviceDesc.(protoreflect.MethodDescriptor)), nil
}
