package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

// NewAccountIdentifier creates a new AccountIdentifier with the given clientId, sender, and salt.
func NewAccountIdentifier(clientId, sender string, salt []byte) AccountIdentifier {
	return AccountIdentifier{
		ClientId: clientId,
		Sender:   sender,
		Salt:     salt,
	}
}

// NewICS27Account creates a new ICS27Account with the given address and accountId.
func NewICS27Account(address string, accountId *AccountIdentifier) ICS27Account {
	return ICS27Account{
		Address:   address,
		AccountId: accountId,
	}
}

// BuildAddressPredictable generates an account address for the gmp module with len = types.AccountAddrLen using the
// Cosmos SDK address.Module function.
// Internally a key is built containing:
// (len(clientId) | clientId | len(sender) | sender | len(salt) | salt).
//
// All method parameter values must be valid and not nil.
//
// This function was copied from wasmd and modified.
// <https://github.com/CosmWasm/wasmd/blob/632fc333d01a84fa5426de6783f7797ad2825e25/x/wasm/keeper/addresses.go#L49>
func BuildAddressPredictable(accountId *AccountIdentifier) (sdk.AccAddress, error) {
	if err := host.ClientIdentifierValidator(accountId.ClientId); err != nil {
		return nil, errorsmod.Wrapf(err, "invalid client ID %s", accountId.ClientId)
	}
	if strings.TrimSpace(accountId.Sender) == "" {
		return nil, errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "missing sender address")
	}

	clientIdBz := uint64LengthPrefix([]byte(accountId.ClientId))
	senderBz := uint64LengthPrefix([]byte(accountId.Sender))
	saltBz := uint64LengthPrefix(accountId.Salt)
	key := make([]byte, len(clientIdBz)+len(senderBz)+len(saltBz))
	copy(key[0:], clientIdBz)
	copy(key[len(clientIdBz):], senderBz)
	copy(key[len(clientIdBz)+len(senderBz):], saltBz)
	return address.Module(accountsKey, key)[:AccountAddrLen], nil
}

// uint64LengthPrefix prepend big endian encoded byte length
func uint64LengthPrefix(bz []byte) []byte {
	return append(sdk.Uint64ToBigEndian(uint64(len(bz))), bz...)
}
