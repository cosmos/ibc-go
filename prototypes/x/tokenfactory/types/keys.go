package types

import "cosmossdk.io/collections"

const (
	ModuleName = "tokenfactory"
	StoreKey   = ModuleName
)

var (
	ParamsKey = collections.NewPrefix(0)

	DenomAuthorityMetadataPrefix = collections.NewPrefix(1)
	CreatorPrefixKey             = collections.NewPrefix(2)
)

func DenomPrefixStore(denom string) []byte {
	return append(DenomAuthorityMetadataPrefix, []byte(denom)...)
}

func CreatorPrefix(creator string) []byte {
	return append(CreatorPrefixKey, []byte(creator)...)
}

func CreatorPrefixStore(creator, denom string) []byte {
	return append(CreatorPrefix(creator), []byte(denom)...)
}
