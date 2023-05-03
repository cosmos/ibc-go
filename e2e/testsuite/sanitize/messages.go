package sanitize

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"

	"github.com/cosmos/ibc-go/e2e/semverutil"
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
		return msg
	case *grouptypes.MsgSubmitProposal:
		if !groupsv1ProposalTitleAndSummary.IsSupported(tag) {
			msg.Title = ""
			msg.Summary = ""
		}
		return msg
	}
	return msg
}
