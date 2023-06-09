package testvalues

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"

	"github.com/cosmos/ibc-go/e2e/semverutil"
	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
)

const (
	StartingTokenAmount int64         = 100_000_000
	IBCTransferAmount   int64         = 10_000
	InvalidAddress      string        = "<invalid-address>"
	VotingPeriod        time.Duration = time.Second * 30
)

// ImmediatelyTimeout returns an ibc.IBCTimeout which will cause an IBC transfer to timeout immediately.
func ImmediatelyTimeout() *ibc.IBCTimeout {
	return &ibc.IBCTimeout{
		NanoSeconds: 1,
	}
}

func DefaultFee(denom string) feetypes.Fee {
	return feetypes.Fee{
		RecvFee:    sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(50))),
		AckFee:     sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(25))),
		TimeoutFee: sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(10))),
	}
}

func DefaultTransferAmount(denom string) sdk.Coin {
	return sdk.Coin{Denom: denom, Amount: sdk.NewInt(IBCTransferAmount)}
}

func TendermintClientID(id int) string {
	return fmt.Sprintf("07-tendermint-%d", id)
}

func SolomachineClientID(id int) string {
	return fmt.Sprintf("06-solomachine-%d", id)
}

// GovGenesisFeatureReleases represents the releases the governance module genesis
// was upgraded from v1beta1 to v1.
var GovGenesisFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v7",
}

// IcadGovGenesisFeatureReleases represents the releases of icad where the governance module genesis
// was upgraded from v1beta1 to v1.
var IcadGovGenesisFeatureReleases = semverutil.FeatureReleases{
	MinorVersions: []string{
		"v0.5",
	},
}

// IcadNewGenesisCommandsFeatureReleases represents the releases of icad using the new genesis commands.
var IcadNewGenesisCommandsFeatureReleases = semverutil.FeatureReleases{
	MinorVersions: []string{
		"v0.5",
	},
}

// SimdNewGenesisCommandsFeatureReleases represents the releases the simd binary started using the new genesis command.
var SimdNewGenesisCommandsFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
}

// TransferSelfParamsFeatureReleases represents the releases the transfer module started managing its own params.
var SelfParamsFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8",
}

// MemoFeatureReleases represents the releases the memo field was released in.
var MemoFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v6",
	MinorVersions: []string{
		"v2.5",
		"v3.4",
		"v4.2",
		"v5.1",
	},
}

// TotalEscrowFeatureReleases represents the releases the total escrow state entry was released in.
var TotalEscrowFeatureReleases = semverutil.FeatureReleases{
	MinorVersions: []string{
		"v7.1",
	},
}

// IbcErrorsFeatureReleases represents the releases the IBC module level errors was released in.
var IbcErrorsFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v8.0",
}
