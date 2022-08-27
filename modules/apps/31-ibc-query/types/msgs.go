package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgSubmitCrossChainQuery       = "submitCrossChainQuery"
	TypeMsgSubmitCrossChainQueryResult = "submitCrossChainQueryResult"
)

// NewMsgSubmitCrossChainQuery creates a new instance of NewMsgSubmitCrossChainQuery
func NewMsgSubmitCrossChainQuery(id string, path string, localTimeoutHeight uint64, localTimeoutStamp uint64, queryHeight uint64, clientId string, creator string) *MsgSubmitCrossChainQuery {
	return &MsgSubmitCrossChainQuery{
		Id:                 id,
		Path:               path,
		LocalTimeoutHeight: localTimeoutHeight,
		LocalTimeoutStamp:  localTimeoutStamp,
		QueryHeight:        queryHeight,
		ClientId:           clientId,
		Sender:             creator,
	}
}

func (q MsgSubmitCrossChainQuery) GetQueryId() string { return q.Id }

func (q MsgSubmitCrossChainQuery) GetPath() string { return q.Path }

func (q MsgSubmitCrossChainQuery) GetTimeoutHeight() uint64 { return q.LocalTimeoutHeight }

func (q MsgSubmitCrossChainQuery) GetTimeoutTimestamp() uint64 { return q.LocalTimeoutStamp }

func (q MsgSubmitCrossChainQuery) GetQueryHeight() uint64 { return q.QueryHeight }

func (q MsgSubmitCrossChainQuery) GetClientID() string { return q.ClientId }


// ValidateBasic implements sdk.Msg and performs basic stateless validation
func (q MsgSubmitCrossChainQuery) ValidateBasic() error {

	return nil
}

// GetSigners implements sdk.Msg
func (q MsgSubmitCrossChainQuery) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(q.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

// Route implements sdk.Msg
func (q MsgSubmitCrossChainQuery) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (q MsgSubmitCrossChainQuery) Type() string {
	return TypeMsgSubmitCrossChainQuery
}

// GetSignBytes implements sdk.Msg.
func (q MsgSubmitCrossChainQuery) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&q))
}

// NewMsgSubmitCrossChainQueryResult creates a new instance of MsgSubmitCrossChainQueryResult
func NewMsgSubmitCrossChainQueryResult(id string, result QueryResult, data []byte) *MsgSubmitCrossChainQueryResult {
	return &MsgSubmitCrossChainQueryResult{
		Id:     id,
		Result: result,
		Data:   data,
	}
}

// ValidateBasic implements sdk.Msg and performs basic stateless validation
func (q MsgSubmitCrossChainQueryResult) ValidateBasic() error {
	return nil
}

// GetSigners implements sdk.Msg
func (q MsgSubmitCrossChainQueryResult) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(q.Relayer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

// Route implements sdk.Msg
func (q MsgSubmitCrossChainQueryResult) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (q MsgSubmitCrossChainQueryResult) Type() string {
	return TypeMsgSubmitCrossChainQueryResult
}

// GetSignBytes implements sdk.Msg.
func (q MsgSubmitCrossChainQueryResult) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&q))
}
