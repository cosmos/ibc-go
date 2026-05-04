package types
import (
	"strings"

	errorsmod "cosmossdk.io/errors"
)

// ValidateBasic performs stateless validation on MsgRegisterIFTBridge.
// It also normalizes Solana addresses by trimming whitespace.
func (m *MsgRegisterIFTBridge) ValidateBasic() error {
	if strings.TrimSpace(m.Signer) == "" {
		return errorsmod.Wrap(ErrInvalidSigner, "signer cannot be empty")
	}
	if !strings.HasPrefix(m.Denom, "factory/") {
		return errorsmod.Wrapf(ErrInvalidDenom, "denom must be a tokenfactory denom, got %s", m.Denom)
	}
	if strings.TrimSpace(m.ClientId) == "" {
		return errorsmod.Wrap(ErrInvalidClientID, "client id cannot be empty")
	}
	m.CounterpartyIftAddress = strings.TrimSpace(m.CounterpartyIftAddress)
	if m.CounterpartyIftAddress == "" {
		return errorsmod.Wrap(ErrInvalidReceiver, "counterparty IFT address cannot be empty")
	}
	if err := ValidateConstructorString(m.IftSendCallConstructor); err != nil {
		return err
	}
	return ValidateCounterpartyAddress(m.IftSendCallConstructor, m.CounterpartyIftAddress)
}
