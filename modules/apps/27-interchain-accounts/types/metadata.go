package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	connectiontypes "github.com/cosmos/ibc-go/v3/modules/core/03-connection/types"
)

// NewMetadata creates and returns a new ICS27 Metadata instance
func NewMetadata(version, controllerConnectionID, hostConnectionID, accAddress string) Metadata {
	return Metadata{
		Version:                version,
		ControllerConnectionId: controllerConnectionID,
		HostConnectionId:       hostConnectionID,
		Address:                accAddress,
	}
}

// ValidateControllerMetadata performs validation of the provided ICS27 controller metadata parameters
func ValidateControllerMetadata(ctx sdk.Context, channelKeeper ChannelKeeper, connectionHops []string, metadata Metadata) error {
	connection, err := channelKeeper.GetConnection(ctx, connectionHops[0])
	if err != nil {
		return err
	}

	if metadata.ControllerConnectionId != connectionHops[0] {
		return sdkerrors.Wrapf(connectiontypes.ErrInvalidConnection, "expected %s, got %s", connectionHops[0], metadata.ControllerConnectionId)
	}

	if metadata.HostConnectionId != connection.GetCounterparty().GetConnectionID() {
		return sdkerrors.Wrapf(connectiontypes.ErrInvalidConnection, "expected %s, got %s", connection.GetCounterparty().GetConnectionID(), metadata.HostConnectionId)
	}

	if metadata.Address != "" {
		if err := ValidateAccountAddress(metadata.Address); err != nil {
			return err
		}
	}

	if metadata.Version != Version {
		return sdkerrors.Wrapf(ErrInvalidVersion, "expected %s, got %s", Version, metadata.Version)
	}

	return nil
}

// ValidateHostMetadata performs validation of the provided ICS27 host metadata parameters
func ValidateHostMetadata(ctx sdk.Context, channelKeeper ChannelKeeper, connectionHops []string, metadata Metadata) error {
	connection, err := channelKeeper.GetConnection(ctx, connectionHops[0])
	if err != nil {
		return err
	}

	if metadata.ControllerConnectionId != connection.GetCounterparty().GetConnectionID() {
		return sdkerrors.Wrapf(connectiontypes.ErrInvalidConnection, "expected %s, got %s", connection.GetCounterparty().GetConnectionID(), metadata.ControllerConnectionId)
	}

	if metadata.HostConnectionId != connectionHops[0] {
		return sdkerrors.Wrapf(connectiontypes.ErrInvalidConnection, "expected %s, got %s", connectionHops[0], metadata.HostConnectionId)
	}

	if metadata.Address != "" {
		if err := ValidateAccountAddress(metadata.Address); err != nil {
			return err
		}
	}

	if metadata.Version != Version {
		return sdkerrors.Wrapf(ErrInvalidVersion, "expected %s, got %s", Version, metadata.Version)
	}

	return nil
}
