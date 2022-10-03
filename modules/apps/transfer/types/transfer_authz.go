package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"

	"golang.org/x/exp/slices"
)

var (
	_ authz.Authorization = &TransferAuthorization{}
)

// NewTransferAuthorization creates a new TransferAuthorization object.
func NewTransferAuthorization(sourcePorts, sourceChannels []string, spendLimits []sdk.Coins) *TransferAuthorization {
	allocations := []PortChannelAmount{}
	for index := range sourcePorts {
		allocations = append(allocations, PortChannelAmount{
			SourcePort:    sourcePorts[index],
			SourceChannel: sourceChannels[index],
			SpendLimit:    spendLimits[index],
		})
	}
	return &TransferAuthorization{
		Allocations: allocations,
	}
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a TransferAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgTransfer{})
}

func IsAllowedAddress(receiver string, allowedAddrs []string) bool {
	for _, addr := range allowedAddresses {
		if addr == receiver {
			return true
		}
	}
	return false
}

// Accept implements Authorization.Accept.
func (a TransferAuthorization) Accept(ctx sdk.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	mTransfer, ok := msg.(*MsgTransfer)
	if !ok {
		return authz.AcceptResponse{}, sdkerrors.ErrInvalidType.Wrap("type mismatch")
	}

	for index, allocation := range a.Allocations {
		if allocation.SourceChannel == mTransfer.SourceChannel && allocation.SourcePort == mTransfer.SourcePort {
			limitLeft, isNegative := allocation.SpendLimit.SafeSub(mTransfer.Token)
			if isNegative {
				return authz.AcceptResponse{}, sdkerrors.ErrInsufficientFunds.Wrapf("requested amount is more than spend limit")
			}

			if !IsAllowedAddress(mTransfer.Receiver, allocation.AllowedAddresses) {
				return authz.AcceptResponse{}, sdkerrors.ErrInsufficientFunds.Wrapf("not allowed address for transfer")
			}

			if limitLeft.IsZero() {
				a.Allocations = slices.Delete(a.Allocations, index, index+1)
				if len(a.Allocations) == 0 {
					return authz.AcceptResponse{Accept: true, Delete: true}, nil
				}
				return authz.AcceptResponse{Accept: true, Delete: false, Updated: &TransferAuthorization{
					Allocations: a.Allocations,
				}}, nil
			}
			a.Allocations[index] = PortChannelAmount{
				SourcePort:    allocation.SourcePort,
				SourceChannel: allocation.SourceChannel,
				SpendLimit:    limitLeft,
			}

			return authz.AcceptResponse{Accept: true, Delete: false, Updated: &TransferAuthorization{
				Allocations: a.Allocations,
			}}, nil
		}
	}
	return authz.AcceptResponse{}, sdkerrors.ErrInsufficientFunds.Wrapf("requested port and channel allocation does not exist")
}

// ValidateBasic implements Authorization.ValidateBasic.
func (a TransferAuthorization) ValidateBasic() error {
	for _, allocation := range a.Allocations {
		if allocation.SpendLimit == nil {
			return sdkerrors.ErrInvalidCoins.Wrap("spend limit cannot be nil")
		}
		if !allocation.SpendLimit.IsAllPositive() {
			return sdkerrors.ErrInvalidCoins.Wrapf("spend limit cannot be negitive")
		}
		if err := host.PortIdentifierValidator(allocation.SourcePort); err != nil {
			return sdkerrors.Wrap(err, "invalid source port ID")
		}
		if err := host.ChannelIdentifierValidator(allocation.SourceChannel); err != nil {
			return sdkerrors.Wrap(err, "invalid source channel ID")
		}
	}
	return nil
}
