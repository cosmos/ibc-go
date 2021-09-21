package types

import (
	"encoding/json"
	"fmt"
	"strings"

	crypto "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	yaml "gopkg.in/yaml.v2"

	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
)

const (
	ICAPrefix string = "ics-27"
)

// GenerateAddress returns an sdk.AccAddress derived using the provided module account address and port identifier
func GenerateAddress(moduleAccAddr sdk.AccAddress, portID string) sdk.AccAddress {
	return sdk.AccAddress(address.Derive(moduleAccAddr, []byte(portID)))
}

// ParseAddressFromVersion trims the interchainaccounts version prefix and returns the associated account address
func ParseAddressFromVersion(version string) string {
	return strings.TrimPrefix(version, fmt.Sprint(VersionPrefix, Delimiter))
}

// GeneratePortID generates the portID for a specific owner
// on the controller chain in the format:
//
// 'ics-27-<connectionSequence>-<counterpartyConnectionSequence>-<owner-address>'
// https://github.com/seantking/ibc/tree/sean/ics-27-updates/spec/app/ics-027-interchain-accounts#registering--controlling-flows
// TODO: update link to spec
func GeneratePortID(owner, connectionID, counterpartyConnectionID string) (string, error) {
	if strings.TrimSpace(owner) == "" {
		return "", sdkerrors.Wrap(ErrInvalidAccountAddress, "owner address cannot be empty")
	}

	connectionSeq, err := connectiontypes.ParseConnectionSequence(connectionID)
	if err != nil {
		return "", sdkerrors.Wrap(err, "invalid connection identifier")
	}

	counterpartyConnectionSeq, err := connectiontypes.ParseConnectionSequence(counterpartyConnectionID)
	if err != nil {
		return "", sdkerrors.Wrap(err, "invalid counterparty connection identifier")
	}

	return fmt.Sprintf("%s-%d-%d-%s", ICAPrefix, connectionSeq, counterpartyConnectionSeq, owner), nil
}

type InterchainAccountI interface {
	authtypes.AccountI
}

var (
	_ authtypes.GenesisAccount = (*InterchainAccount)(nil)
	_ InterchainAccountI       = (*InterchainAccount)(nil)
)

func NewInterchainAccount(ba *authtypes.BaseAccount, accountOwner string) *InterchainAccount {
	return &InterchainAccount{
		BaseAccount:  ba,
		AccountOwner: accountOwner,
	}
}

// SetPubKey - Implements AccountI
func (InterchainAccount) SetPubKey(pubKey crypto.PubKey) error {
	return fmt.Errorf("not supported for interchain accounts")
}

// SetSequence - Implements AccountI
func (InterchainAccount) SetSequence(seq uint64) error {
	return fmt.Errorf("not supported for interchain accounts")
}

func (ia InterchainAccount) Validate() error {
	return ia.BaseAccount.Validate()
}

type ibcAccountPretty struct {
	Address       sdk.AccAddress `json:"address" yaml:"address"`
	PubKey        string         `json:"public_key" yaml:"public_key"`
	AccountNumber uint64         `json:"account_number" yaml:"account_number"`
	Sequence      uint64         `json:"sequence" yaml:"sequence"`
	AccountOwner  string         `json:"address" yaml:"account_owner"`
}

func (ia InterchainAccount) String() string {
	out, _ := ia.MarshalYAML()
	return out.(string)
}

// MarshalYAML returns the YAML representation of a InterchainAccount.
func (ia InterchainAccount) MarshalYAML() (interface{}, error) {
	accAddr, err := sdk.AccAddressFromBech32(ia.Address)
	if err != nil {
		return nil, err
	}

	bs, err := yaml.Marshal(ibcAccountPretty{
		Address:       accAddr,
		PubKey:        "",
		AccountNumber: ia.AccountNumber,
		Sequence:      ia.Sequence,
		AccountOwner:  ia.AccountOwner,
	})

	if err != nil {
		return nil, err
	}

	return string(bs), nil
}

// MarshalJSON returns the JSON representation of a InterchainAccount.
func (ia InterchainAccount) MarshalJSON() ([]byte, error) {
	accAddr, err := sdk.AccAddressFromBech32(ia.Address)
	if err != nil {
		return nil, err
	}

	return json.Marshal(ibcAccountPretty{
		Address:       accAddr,
		PubKey:        "",
		AccountNumber: ia.AccountNumber,
		Sequence:      ia.Sequence,
		AccountOwner:  ia.AccountOwner,
	})
}

// UnmarshalJSON unmarshals raw JSON bytes into a ModuleAccount.
func (ia *InterchainAccount) UnmarshalJSON(bz []byte) error {
	var alias ibcAccountPretty
	if err := json.Unmarshal(bz, &alias); err != nil {
		return err
	}

	ia.BaseAccount = authtypes.NewBaseAccount(alias.Address, nil, alias.AccountNumber, alias.Sequence)
	ia.AccountOwner = alias.AccountOwner

	return nil
}
