package validate

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// GRPCRequest validates that the portID and channelID of a gRPC Request are valid identifiers.
func GRPCRequest(portID, channelID string) error {
	if err := host.PortIdentifierValidator(portID); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if err := host.ChannelIdentifierValidator(channelID); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	return nil
}
