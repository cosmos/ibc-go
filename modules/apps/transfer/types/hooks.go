package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TransferHooks interface {
	AfterSendTransfer(
		ctx sdk.Context,
		sourcePort, sourceChannel string,
		token sdk.Coin,
		sender sdk.AccAddress,
		receiver string,
		isSource bool)
	AfterRecvTransfer(
		ctx sdk.Context,
		destPort, destChannel string,
		token sdk.Coin,
		receiver string,
		isSource bool)
	AfterRefundTransfer(
		ctx sdk.Context,
		sourcePort, sourceChannel string,
		token sdk.Coin,
		sender string,
		isSource bool)
}

type MultiTransferHooks []TransferHooks

func NewMultiTransferHooks(hooks ...TransferHooks) MultiTransferHooks {
	return hooks
}

func (mths MultiTransferHooks) AfterSendTransfer(
	ctx sdk.Context,
	sourcePort, sourceChannel string,
	token sdk.Coin,
	sender sdk.AccAddress,
	receiver string,
	isSource bool) {
	for i := range mths {
		mths[i].AfterSendTransfer(ctx, sourcePort, sourceChannel, token, sender, receiver, isSource)
	}
}

func (mths MultiTransferHooks) AfterRecvTransfer(
	ctx sdk.Context,
	destPort, destChannel string,
	token sdk.Coin,
	receiver string,
	isSource bool) {
	for i := range mths {
		mths[i].AfterRecvTransfer(ctx, destPort, destChannel, token, receiver, isSource)
	}
}

func (mths MultiTransferHooks) AfterRefundTransfer(
	ctx sdk.Context,
	sourcePort, sourceChannel string,
	token sdk.Coin,
	sender string,
	isSource bool) {
	for i := range mths {
		mths[i].AfterRefundTransfer(ctx, sourcePort, sourceChannel, token, sender, isSource)
	}
}
