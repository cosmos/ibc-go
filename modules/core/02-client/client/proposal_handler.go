package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"github.com/cosmos/ibc-go/v7/modules/core/02-client/client/cli"
)

var UpgradeProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitUpgradeProposal)
