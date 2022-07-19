package types

func NewMsgCrossChainQuery(id, path string, localTimeoutHeight, localTimeoutStamp, queryHeight uint64, clientId string) *MsgCrossChainQuery {
	return &MsgCrossChainQuery{
		Id:                 id,
		Path:               path,
		LocalTimeoutHeight: localTimeoutHeight,
		LocalTimeoutStamp:  localTimeoutStamp,
		QueryHeight:        queryHeight,
		ClientId:           clientId,
	}
}

func NewMsgCrossChainQueryResult(id string, result QueryResult, data []byte) *MsgCrossChainQueryResult {
	return &MsgCrossChainQueryResult{
		Id:     id,
		Result: result,
		Data:   data,
	}
}
