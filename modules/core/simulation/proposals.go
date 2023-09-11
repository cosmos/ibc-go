package simulation

import (
	"math/rand"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

// Simulation operation weights constants
const (
	DefaultWeight int = 100

	OpWeightMsgUpdateParams       = "op_weight_msg_update_params"                 // #nosec
	OpWeightMsgRecoverClient      = "op_weight_msg_recover_client"                // #nosec
	OpWeightMsgIBCSoftwareUpgrade = "op_weight_msg_schedule_ibc_software_upgrade" // #nosec
)

// ProposalMsgs defines the module weighted proposals' contents
func ProposalMsgs() []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			OpWeightMsgUpdateParams,
			DefaultWeight,
			SimulateClientMsgUpdateParams,
		),
		simulation.NewWeightedProposalMsg(
			OpWeightMsgUpdateParams,
			DefaultWeight,
			SimulateConnectionMsgUpdateParams,
		),
		simulation.NewWeightedProposalMsg(
			OpWeightMsgRecoverClient,
			DefaultWeight,
			SimulateClientMsgRecoverClient,
		),
		simulation.NewWeightedProposalMsg(
			OpWeightMsgIBCSoftwareUpgrade,
			DefaultWeight,
			SimulateClientMsgScheduleIBCSoftwareUpgrade,
		),
	}
}

// SimulateClientMsgUpdateParams returns a random MsgUpdateParams for 02-client
func SimulateClientMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var signer sdk.AccAddress = address.Module("gov")
	params := types.DefaultParams()
	params.AllowedClients = []string{"06-solomachine", "07-tendermint"}

	return &types.MsgUpdateParams{
		Signer: signer.String(),
		Params: params,
	}
}

// SimulateClientMsgRecoverClient returns a random MsgRecoverClient for 02-client
func SimulateClientMsgRecoverClient(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var signer sdk.AccAddress = address.Module("gov")

	return &types.MsgRecoverClient{
		Signer:             signer.String(),
		SubjectClientId:    "07-tendermint-0",
		SubstituteClientId: "07-tendermint-1",
	}
}

// SimulateClientMsgScheduleIBCSoftwareUpgrade returns a random MsgScheduleIBCSoftwareUpgrade for 02-client
func SimulateClientMsgScheduleIBCSoftwareUpgrade(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var signer sdk.AccAddress = address.Module("gov")

	chainID := "chain-a-0"
	trustingPeriod := time.Hour * 24 * 7 * 2
	// two week unbonding period instead of default three week unbonding period
	ubdPeriod := time.Hour * 24 * 7 * 2
	maxClockDrift := time.Second * 10
	upgradePath := []string{"upgrade", "upgradedIBCState"}

	upgradedClientState := ibctm.NewClientState(chainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.ZeroHeight(), commitmenttypes.GetSDKSpecs(), upgradePath)
	anyClient, err := types.PackClientState(upgradedClientState)
	if err != nil {
		panic(err)
	}

	return &types.MsgIBCSoftwareUpgrade{
		Signer: signer.String(),
		Plan: upgradetypes.Plan{
			Name:   "upgrade",
			Height: 100,
		},
		UpgradedClientState: anyClient,
	}
}

// SimulateConnectionMsgUpdateParams returns a random MsgUpdateParams 03-connection
func SimulateConnectionMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var signer sdk.AccAddress = address.Module("gov")
	params := connectiontypes.DefaultParams()
	params.MaxExpectedTimePerBlock = uint64(100)

	return &connectiontypes.MsgUpdateParams{
		Signer: signer.String(),
		Params: params,
	}
}
