package mock

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

const (
	ModuleName = "mock"

	MemStoreKey = "memory:mock"

	PortID = ModuleName

	Version = "mock-version"
)

var (
	MockAcknowledgement             = channeltypes.NewResultAcknowledgement([]byte("mock acknowledgement"))
	MockFailAcknowledgement         = channeltypes.NewErrorAcknowledgement(fmt.Errorf("mock failed acknowledgement"))
	MockPacketData                  = []byte("mock packet data")
	MockFailPacketData              = []byte("mock failed packet data")
	MockAsyncPacketData             = []byte("mock async packet data")
	MockRecvCanaryCapabilityName    = "mock receive canary capability name"
	MockAckCanaryCapabilityName     = "mock acknowledgement canary capability name"
	MockTimeoutCanaryCapabilityName = "mock timeout canary capability name"
	UpgradeVersion                  = fmt.Sprintf("%s-v2", Version)
	// MockApplicationCallbackError should be returned when an application callback should fail. It is possible to
	// test that this error was returned using ErrorIs.
	MockApplicationCallbackError error = &applicationCallbackError{}
	MockFeeVersion                     = string(feetypes.ModuleCdc.MustMarshalJSON(&feetypes.Metadata{FeeVersion: feetypes.Version, AppVersion: Version}))
)

var (
	TestKey   = []byte("test-key")
	TestValue = []byte("test-value")
)

// PortKeeper defines the expected IBC PortKeeper interface.
type PortKeeper interface {
	BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability
	IsBound(ctx sdk.Context, portID string) bool
}

// ScopedMockKeeper embeds x/capability's ScopedKeeper used for depinject module outputs.
type ScopedMockKeeper struct{ capabilitykeeper.ScopedKeeper }

var _ exported.Acknowledgement = (*EmptyAcknowledgement)(nil)

// EmptyAcknowledgement implements the exported.Acknowledgement interface and always returns an empty byte string as Response
type EmptyAcknowledgement struct {
	Response []byte
}

// NewEmptyAcknowledgement returns a new instance of EmptyAcknowledgement
func NewEmptyAcknowledgement() EmptyAcknowledgement {
	return EmptyAcknowledgement{
		Response: []byte{},
	}
}

// Success implements the Acknowledgement interface
func (EmptyAcknowledgement) Success() bool {
	return true
}

// Acknowledgement implements the Acknowledgement interface
func (EmptyAcknowledgement) Acknowledgement() []byte {
	return []byte{}
}

var _ exported.Path = KeyPath{}

// KeyPath defines a placeholder struct which implements the exported.Path interface
type KeyPath struct{}

// String implements the exported.Path interface
func (KeyPath) String() string {
	return ""
}

// Empty implements the exported.Path interface
func (KeyPath) Empty() bool {
	return false
}

var _ exported.Height = Height{}

// Height defines a placeholder struct which implements the exported.Height interface
type Height struct {
	exported.Height
}
