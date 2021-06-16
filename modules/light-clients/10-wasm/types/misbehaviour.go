package types

import (
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

	return nil
}
