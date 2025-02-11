package testvalues

import (
	"fmt"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/e2e/semverutil"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
)

const (
	StartingTokenAmount             int64  = 500_000_000_000
	IBCTransferAmount               int64  = 10_000
	InvalidAddress                  string = "<invalid-address>"
	DefaultGovV1ProposalTokenAmount        = 500_000_000
)

// VotingPeriod may differ per test.
var VotingPeriod = time.Second * 30

// ImmediatelyTimeout returns an ibc.IBCTimeout which will cause an IBC transfer to timeout immediately.
func ImmediatelyTimeout() *ibc.IBCTimeout {
	return &ibc.IBCTimeout{
		NanoSeconds: 1,
	}
}

func DefaultFee(denom string) feetypes.Fee {
	return feetypes.Fee{
		RecvFee:    sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(50))),
		AckFee:     sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(25))),
		TimeoutFee: sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10))),
	}
}

func DefaultTransferAmount(denom string) sdk.Coin {
	return sdk.Coin{Denom: denom, Amount: sdkmath.NewInt(IBCTransferAmount)}
}

func DefaultTransferCoins(denom string) sdk.Coins {
	return sdk.NewCoins(DefaultTransferAmount(denom))
}

func TransferAmount(amount int64, denom string) sdk.Coin {
	return sdk.Coin{Denom: denom, Amount: sdkmath.NewInt(amount)}
}

func TendermintClientID(id int) string {
	return fmt.Sprintf("07-tendermint-%d", id)
}

func SolomachineClientID(id int) string {
	return fmt.Sprintf("06-solomachine-%d", id)
}

// FeeMiddlewareFeatureReleases represents the releases the support for fee middleware was released in.
var FeeMiddlewareFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v4",
}

// TokenMetadataFeatureReleases represents the releases the token metadata was released in.
var TokenMetadataFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
}

// GovGenesisFeatureReleases represents the releases the governance module genesis
// was upgraded from v1beta1 to v1.
var GovGenesisFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v7",
}

// SelfParamsFeatureReleases represents the releases the transfer module started managing its own params.
var SelfParamsFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
}

// TotalEscrowFeatureReleases represents the releases the total escrow state entry was released in.
var TotalEscrowFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
	MinorVersions: []string{
		"v7.1",
	},
}

// IbcErrorsFeatureReleases represents the releases the IBC module level errors was released in.
var IbcErrorsFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
}

// LocalhostClientFeatureReleases represents the releases the localhost client was released in.
var LocalhostClientFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
	MinorVersions: []string{
		"v7.1",
	},
}

// AllowAllClientsWildcardFeatureReleases represents the releases the allow all clients wildcard was released in.
var AllowAllClientsWildcardFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v9",
	MinorVersions: []string{
		"v8.1",
	},
}

// ChannelParamsFeatureReleases represents the releases the params for 04-channel was released in.
var ChannelParamsFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v9",
	MinorVersions: []string{
		"v8.1",
	},
}

// GovV1MessagesFeatureReleases represents the releases the support for x/gov v1 messages was released in.
var GovV1MessagesFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
}

// CapitalEfficientFeeEscrowFeatureReleases represents the releases the support for capital efficient fee escrow was released in.
var CapitalEfficientFeeEscrowFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v9",
	MinorVersions: []string{
		"v8.1",
	},
}

// TransactionEventQueryFeatureReleases represents the releases the support for --query flag
// in "query txs" for searching transactions that match exact events (since Cosmos SDK v0.50) was released in.
var TransactionEventQueryFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
}

// ICS20v2FeatureReleases represents the releases the support for ICS20 v2 was released in.
var ICS20v2FeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v9",
}
