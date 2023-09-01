package ibctesting

import (
	"time"

	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

type ClientConfig interface {
	GetClientType() string
}

type TendermintConfig struct {
	TrustLevel      ibctm.Fraction
	TrustingPeriod  time.Duration
	UnbondingPeriod time.Duration
	MaxClockDrift   time.Duration
}

func NewTendermintConfig() *TendermintConfig {
	return &TendermintConfig{
		TrustLevel:      DefaultTrustLevel,
		TrustingPeriod:  TrustingPeriod,
		UnbondingPeriod: UnbondingPeriod,
		MaxClockDrift:   MaxClockDrift,
	}
}

func (tmcfg *TendermintConfig) GetClientType() string {
	return exported.Tendermint
}

type WasmTendermintConfig struct {
	TendermintConfig
	CodeHash []byte
}

// NewWasmTendermintConfig create a new Tendermint Wasm client configuration.
// It assumes the Wasm contract uses the same default values as a regular
// Tendermint light client.
func NewWasmTendermintConfig(codeHash []byte) *WasmTendermintConfig {
	return &WasmTendermintConfig{
		CodeHash:         codeHash,
		TendermintConfig: *NewTendermintConfig(),
	}
}

func (wasmcfg *WasmTendermintConfig) GetClientType() string {
	return exported.Wasm
}

type ConnectionConfig struct {
	DelayPeriod uint64
	Version     *connectiontypes.Version
}

func NewConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		DelayPeriod: DefaultDelayPeriod,
		Version:     ConnectionVersion,
	}
}

type ChannelConfig struct {
	PortID  string
	Version string
	Order   channeltypes.Order
}

func NewChannelConfig() *ChannelConfig {
	return &ChannelConfig{
		PortID:  mock.PortID,
		Version: DefaultChannelVersion,
		Order:   channeltypes.UNORDERED,
	}
}
