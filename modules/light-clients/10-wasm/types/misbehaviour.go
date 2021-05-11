package types

import (
	"fmt"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

var (
	_ exported.Misbehaviour = &Misbehaviour{}
)

func (m *Misbehaviour) ClientType() string {
	return m.Header1.ClientType()
}

func (m *Misbehaviour) GetClientID() string {
	return m.ClientId
}

func (m *Misbehaviour) ValidateBasic() error {
	if err := m.Header1.ValidateBasic(); err != nil {
		return err
	}

	if err := m.Header2.ValidateBasic(); err != nil {
		return err
	}

	if m.CodeId == nil || len(m.CodeId) == 0 {
		return fmt.Errorf("codeid cannot be empty")
	}

	return nil
}

func (m *Misbehaviour) GetHeight() exported.Height {
	return m.Header1.GetHeight()
}