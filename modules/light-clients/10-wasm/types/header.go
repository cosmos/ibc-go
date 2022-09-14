package types

import (
	"fmt"

	"github.com/cosmos/ibc-go/v5/modules/core/exported"
)

// TODO: figure out what to do with the header interface
// var _ exported.Header = (*Header)(nil)

func (m *Header) ClientType() string {
	return exported.Wasm
}

func (m *Header) GetHeight() exported.Height {
	return m.Height
}

func (m *Header) ValidateBasic() error {
	if m.Data == nil || len(m.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	return nil
}
