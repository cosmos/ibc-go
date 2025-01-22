package sanitize

import (
	govtypesv1 "cosmossdk.io/x/gov/types/v1"
	grouptypes "cosmossdk.io/x/group"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtcrypto "github.com/cometbft/cometbft/api/cometbft/crypto/v1"
	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"

	"github.com/cosmos/ibc-go/e2e/semverutil"
	icacontrollertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
)

var (
	// groupsv1ProposalTitleAndSummary represents the releases that support the new title and summary fields.
	groupsv1ProposalTitleAndSummary = semverutil.FeatureReleases{
		MajorVersion: "v7",
	}
	// govv1ProposalTitleAndSummary represents the releases that support the new title and summary fields.
	govv1ProposalTitleAndSummary = semverutil.FeatureReleases{
		MajorVersion: "v7",
	}
	// icaUnorderedChannelFeatureReleases represents the releasees that support the new ordering field.
	icaUnorderedChannelFeatureReleases = semverutil.FeatureReleases{
		MajorVersion: "v9",
		MinorVersions: []string{
			"v7.5",
			"v8.1",
		},
	}
	// groupsv1ProposalProposalType represents the releases that support the new proposal type field.
	govv1ProposalProposalType = semverutil.FeatureReleases{
		MajorVersion: "v10",
	}
	// cometBFTv1Validator represents the releases that support the new validator fields.
	cometBFTv1Validator = semverutil.FeatureReleases{
		MajorVersion: "v10",
	}
)

// Messages removes any fields that are not supported by the chain version.
// For example, any fields that have been added in later sdk releases.
func Messages(tag string, msgs ...sdk.Msg) []sdk.Msg {
	sanitizedMsgs := make([]sdk.Msg, len(msgs))
	for i := range msgs {
		sanitizedMsgs[i] = removeUnknownFields(tag, msgs[i])
	}
	return sanitizedMsgs
}

// removeUnknownFields removes any fields that are not supported by the chain version.
// The input message is returned if no changes are made.
func removeUnknownFields(tag string, msg sdk.Msg) sdk.Msg {
	switch msg := msg.(type) {
	case *govtypesv1.MsgSubmitProposal:
		if !govv1ProposalTitleAndSummary.IsSupported(tag) {
			msg.Title = ""
			msg.Summary = ""
		}
		if !govv1ProposalProposalType.IsSupported(tag) {
			msg.ProposalType = govtypesv1.ProposalType_PROPOSAL_TYPE_UNSPECIFIED
		}
		// sanitize messages contained in the x/gov proposal
		msgs, err := msg.GetMsgs()
		if err != nil {
			panic(err)
		}
		sanitizedMsgs := Messages(tag, msgs...)
		if err := msg.SetMsgs(sanitizedMsgs); err != nil {
			panic(err)
		}
		return msg
	case *grouptypes.MsgSubmitProposal:
		if !groupsv1ProposalTitleAndSummary.IsSupported(tag) {
			msg.Title = ""
			msg.Summary = ""
		}
		// sanitize messages contained in the x/group proposal
		msgs, err := msg.GetMsgs()
		if err != nil {
			panic(err)
		}
		sanitizedMsgs := Messages(tag, msgs...)
		if err := msg.SetMsgs(sanitizedMsgs); err != nil {
			panic(err)
		}
		return msg
	case *icacontrollertypes.MsgRegisterInterchainAccount:
		if !icaUnorderedChannelFeatureReleases.IsSupported(tag) {
			msg.Ordering = channeltypes.NONE
		}
	case *clienttypes.MsgUpdateClient:
		if !cometBFTv1Validator.IsSupported(tag) {
			clientMessage, err := clienttypes.UnpackClientMessage(msg.ClientMessage)
			if err != nil {
				panic(err)
			}
			header, ok := clientMessage.(*ibctm.Header)
			if !ok {
				return msg
			}

			convertCometBFTValidatorV1(header.ValidatorSet.Proposer)
			for _, validator := range header.ValidatorSet.Validators {
				convertCometBFTValidatorV1(validator)
			}

			convertCometBFTValidatorV1(header.TrustedValidators.Proposer)
			for _, validator := range header.TrustedValidators.Validators {
				convertCometBFTValidatorV1(validator)
			}

			// repack the client message
			clientMessageAny, err := clienttypes.PackClientMessage(header)
			if err != nil {
				panic(err)
			}
			msg.ClientMessage = clientMessageAny
		}
	}
	return msg
}

func convertCometBFTValidatorV1(validator *cmtproto.Validator) {
	validator.PubKey = &cmtcrypto.PublicKey{
		Sum: &cmtcrypto.PublicKey_Ed25519{
			Ed25519: validator.PubKeyBytes,
		},
	}
	validator.PubKeyBytes = nil
	validator.PubKeyType = ""
}
