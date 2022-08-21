package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgSubmitCrossChainQuery       = "submitCrossChainQuery"
	TypeMsgSubmitCrossChainQueryResult = "submitCrossChainQueryResult"
)

func NewMsgSubmitCrossChainQuery(path string, localTimeoutHeight, localTimeoutStamp, queryHeight uint64, clientId string) *MsgSubmitCrossChainQuery {
	return &MsgSubmitCrossChainQuery{
		Path:               path,
		LocalTimeoutHeight: localTimeoutHeight,
		LocalTimeoutStamp:  localTimeoutStamp,
		QueryHeight:        queryHeight,
		ClientId:           clientId,
	}
}

// ValidateBasic implements sdk.Msg and performs basic stateless validation
func (msg MsgSubmitCrossChainQuery) ValidateBasic() error {

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgSubmitCrossChainQuery) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

// Route implements sdk.Msg
func (msg MsgSubmitCrossChainQuery) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg MsgSubmitCrossChainQuery) Type() string {
	return TypeMsgSubmitCrossChainQuery
}

// GetSignBytes implements sdk.Msg.
func (msg MsgSubmitCrossChainQuery) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}

func NewMsgSubmitCrossChainQueryResult(id string, result QueryResult, data []byte) *MsgSubmitCrossChainQueryResult {
	return &MsgSubmitCrossChainQueryResult{
		Id:     id,
		Result: result,
		Data:   data,
	}
}

// ValidateBasic implements sdk.Msg and performs basic stateless validation
func (msg MsgSubmitCrossChainQueryResult) ValidateBasic() error {
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgSubmitCrossChainQueryResult) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Relayer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

// Route implements sdk.Msg
func (msg MsgSubmitCrossChainQueryResult) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg MsgSubmitCrossChainQueryResult) Type() string {
	return TypeMsgSubmitCrossChainQueryResult
}

// GetSignBytes implements sdk.Msg.
func (msg MsgSubmitCrossChainQueryResult) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}
