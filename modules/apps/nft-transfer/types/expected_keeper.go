package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/cosmos-sdk/x/auth/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// Class defines the expected nft class
type Class interface {
	GetId() string
	GetUri() string
}

// NFT defines the expected nft
type NFT interface {
	GetId() string
	GetUri() string
}

// NFTKeeper defines the expected nft keeper
// TODO
type NFTKeeper interface {
	SaveClass(ctx sdk.Context, classID, classURI string) error
	Mint(ctx sdk.Context, classID, tokenID, tokenURI, receiver string) error
	Transfer(ctx sdk.Context, classID, tokenID, receiver string) error
	Burn(ctx sdk.Context, classID, tokenID string) error

	GetOwner(ctx sdk.Context, classID string, nftID string) sdk.AccAddress
	HasClass(ctx sdk.Context, classID string) bool
	GetClass(ctx sdk.Context, classID string) (Class, bool)
	GetNFT(ctx sdk.Context, classID, tokenID string) (NFT, bool)
}

// ICS4Wrapper defines the expected ICS4Wrapper for middleware
type ICS4Wrapper interface {
	SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet ibcexported.PacketI) error
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
}

// PortKeeper defines the expected IBC port keeper
type PortKeeper interface {
	BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability
}

// AccountKeeper defines the contract required for account APIs.
type AccountKeeper interface {
	NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	// Set an account in the store.
	SetAccount(sdk.Context, types.AccountI)
	HasAccount(ctx sdk.Context, addr sdk.AccAddress) bool
}
