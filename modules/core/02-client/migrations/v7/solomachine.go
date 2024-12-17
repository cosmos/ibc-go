package v7

import (
	"context"
	"errors"

	gogoprotoany "github.com/cosmos/gogoproto/types/any"

	coreregistry "cosmossdk.io/core/registry"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// NOTE: this is a mock implementation for exported.ClientState. This implementation
// should only be registered on the InterfaceRegistry during cli command genesis migration.
// This implementation is only used to successfully unmarshal the previous solo machine
// client state and consensus state and migrate them to the new implementations. When the proto
// codec unmarshals, it calls UnpackInterfaces() to create a cached value of the any. The
// UnpackInterfaces function for IdenitifiedClientState will attempt to unpack the any to
// exported.ClientState. If the solomachine v2 type is not registered against the exported.ClientState
// the unmarshal will fail. This implementation will panic on every interface function.
// The same is done for the ConsensusState.

// Interface implementation checks.
var (
	_, _ gogoprotoany.UnpackInterfacesMessage = (*ClientState)(nil), (*ConsensusState)(nil)
	_    exported.ClientState                 = (*ClientState)(nil)
	_    exported.ConsensusState              = (*ConsensusState)(nil)
)

// RegisterInterfaces registers the solomachine v2 ClientState and ConsensusState types in the interface registry.
func RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	registry.RegisterImplementations(
		(*exported.ClientState)(nil),
		&ClientState{},
	)
	registry.RegisterImplementations(
		(*exported.ConsensusState)(nil),
		&ConsensusState{},
	)
}

// UnpackInterfaces implements the UnpackInterfaceMessages.UnpackInterfaces method
func (cs ClientState) UnpackInterfaces(unpacker gogoprotoany.AnyUnpacker) error {
	return cs.ConsensusState.UnpackInterfaces(unpacker)
}

// UnpackInterfaces implements the UnpackInterfaceMessages.UnpackInterfaces method
func (cs ConsensusState) UnpackInterfaces(unpacker gogoprotoany.AnyUnpacker) error {
	return unpacker.UnpackAny(cs.PublicKey, new(cryptotypes.PubKey))
}

// ClientType panics!
func (ClientState) ClientType() string {
	panic(errors.New("legacy solo machine is deprecated"))
}

// GetLatestHeight panics!
func (ClientState) GetLatestHeight() exported.Height {
	panic(errors.New("legacy solo machine is deprecated"))
}

// Status panics!
func (ClientState) Status(_ context.Context, _ storetypes.KVStore, _ codec.BinaryCodec) exported.Status {
	panic(errors.New("legacy solo machine is deprecated"))
}

// Validate panics!
func (ClientState) Validate() error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// Initialize panics!
func (ClientState) Initialize(_ context.Context, _ codec.BinaryCodec, _ storetypes.KVStore, _ exported.ConsensusState) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// CheckForMisbehaviour panics!
func (ClientState) CheckForMisbehaviour(_ context.Context, _ codec.BinaryCodec, _ storetypes.KVStore, _ exported.ClientMessage) bool {
	panic(errors.New("legacy solo machine is deprecated"))
}

// UpdateStateOnMisbehaviour panics!
func (*ClientState) UpdateStateOnMisbehaviour(
	_ context.Context, _ codec.BinaryCodec, _ storetypes.KVStore, _ exported.ClientMessage,
) {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyClientMessage panics!
func (*ClientState) VerifyClientMessage(
	_ context.Context, _ codec.BinaryCodec, _ storetypes.KVStore, _ exported.ClientMessage,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// UpdateState panis!
func (*ClientState) UpdateState(_ context.Context, _ codec.BinaryCodec, _ storetypes.KVStore, _ exported.ClientMessage) []exported.Height {
	panic(errors.New("legacy solo machine is deprecated"))
}

// CheckHeaderAndUpdateState panics!
func (*ClientState) CheckHeaderAndUpdateState(
	_ context.Context, _ codec.BinaryCodec, _ storetypes.KVStore, _ exported.ClientMessage,
) (exported.ClientState, exported.ConsensusState, error) {
	panic(errors.New("legacy solo machine is deprecated"))
}

// CheckMisbehaviourAndUpdateState panics!
func (ClientState) CheckMisbehaviourAndUpdateState(
	_ context.Context, _ codec.BinaryCodec, _ storetypes.KVStore, _ exported.ClientMessage,
) (exported.ClientState, error) {
	panic(errors.New("legacy solo machine is deprecated"))
}

// CheckSubstituteAndUpdateState panics!
func (ClientState) CheckSubstituteAndUpdateState(
	ctx context.Context, _ codec.BinaryCodec, _, _ storetypes.KVStore,
	_ exported.ClientState,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyUpgradeAndUpdateState panics!
func (ClientState) VerifyUpgradeAndUpdateState(
	_ context.Context, _ codec.BinaryCodec, _ storetypes.KVStore,
	_ exported.ClientState, _ exported.ConsensusState, _, _ []byte,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyClientState panics!
func (ClientState) VerifyClientState(
	store storetypes.KVStore, cdc codec.BinaryCodec,
	_ exported.Height, _ exported.Prefix, _ string, _ []byte, clientState exported.ClientState,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyClientConsensusState panics!
func (ClientState) VerifyClientConsensusState(
	storetypes.KVStore, codec.BinaryCodec,
	exported.Height, string, exported.Height, exported.Prefix,
	[]byte, exported.ConsensusState,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyPacketCommitment panics!
func (ClientState) VerifyPacketCommitment(
	context.Context, storetypes.KVStore, codec.BinaryCodec, exported.Height,
	uint64, uint64, exported.Prefix, []byte,
	string, string, uint64, []byte,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyPacketAcknowledgement panics!
func (ClientState) VerifyPacketAcknowledgement(
	context.Context, storetypes.KVStore, codec.BinaryCodec, exported.Height,
	uint64, uint64, exported.Prefix, []byte,
	string, string, uint64, []byte,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyPacketReceiptAbsence panics!
func (ClientState) VerifyPacketReceiptAbsence(
	context.Context, storetypes.KVStore, codec.BinaryCodec, exported.Height,
	uint64, uint64, exported.Prefix, []byte,
	string, string, uint64,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyNextSequenceRecv panics!
func (ClientState) VerifyNextSequenceRecv(
	context.Context, storetypes.KVStore, codec.BinaryCodec, exported.Height,
	uint64, uint64, exported.Prefix, []byte,
	string, string, uint64,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// GetTimestampAtHeight panics!
func (ClientState) GetTimestampAtHeight(
	context.Context, storetypes.KVStore, codec.BinaryCodec, exported.Height,
) (uint64, error) {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyMembership panics!
func (*ClientState) VerifyMembership(
	ctx context.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// VerifyNonMembership panics!
func (*ClientState) VerifyNonMembership(
	ctx context.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	panic(errors.New("legacy solo machine is deprecated"))
}

// ClientType panics!
func (ConsensusState) ClientType() string {
	panic(errors.New("legacy solo machine is deprecated"))
}

// GetTimestamp panics!
func (ConsensusState) GetTimestamp() uint64 {
	panic(errors.New("legacy solo machine is deprecated"))
}

// ValidateBasic panics!
func (ConsensusState) ValidateBasic() error {
	panic(errors.New("legacy solo machine is deprecated"))
}
