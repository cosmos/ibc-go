package types

func NewCounterpartyInfo(counterpartyMessagingKey [][]byte) CounterpartyInfo {
	return CounterpartyInfo{
		CounterpartyMessagingKey: counterpartyMessagingKey,
	}
}
