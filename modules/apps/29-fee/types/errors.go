package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// 29-fee sentinel errors
var (
	ErrInvalidVersion                = sdkerrors.Register(ModuleName, 2, "invalid ICS29 middleware version")
	ErrRefundAccNotFound             = sdkerrors.Register(ModuleName, 3, "no account found for given refund address")
	ErrBalanceNotFound               = sdkerrors.Register(ModuleName, 4, "balance not found for given account address")
	ErrFeeNotFound                   = sdkerrors.Register(ModuleName, 5, "there is no fee escrowed for the given packetID")
	ErrRelayersNotNil                = sdkerrors.Register(ModuleName, 6, "relayers must be nil. This feature is not supported")
	ErrCounterpartyPayeeEmpty        = sdkerrors.Register(ModuleName, 7, "counterparty payee must not be empty")
	ErrForwardRelayerAddressNotFound = sdkerrors.Register(ModuleName, 8, "forward relayer address not found")
	ErrFeeNotEnabled                 = sdkerrors.Register(ModuleName, 9, "fee module is not enabled for this channel. If this error occurs after channel setup, fee module may not be enabled")
	ErrRelayerNotFoundForAsyncAck    = sdkerrors.Register(ModuleName, 10, "relayer address must be stored for async WriteAcknowledgement")
	ErrFeeModuleLocked               = sdkerrors.Register(ModuleName, 11, "the fee module is currently locked, a severe bug has been detected")
)
