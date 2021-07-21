package types

type TxEncoder func(data interface{}) ([]byte, error)
