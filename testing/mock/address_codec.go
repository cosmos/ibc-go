package mock

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TestAddressCodec struct{}

func (TestAddressCodec) StringToBytes(text string) ([]byte, error) {
	hexBytes, err := sdk.AccAddressFromHexUnsafe(text)
	if err == nil {
		return hexBytes, nil
	}

	bech32Bytes, err := sdk.AccAddressFromBech32(text)
	if err == nil {
		return bech32Bytes, nil
	}

	return nil, errors.New("invalid address format")
}

func (TestAddressCodec) BytesToString(bz []byte) (string, error) {
	return sdk.AccAddress(bz).String(), nil
}
