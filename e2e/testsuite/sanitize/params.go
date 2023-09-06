package sanitize

import govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

func GovV1Params(version string, params *govv1.Params) *govv1.Params {
	return params
}
