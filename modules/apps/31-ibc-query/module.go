package ibc_query

import "github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return types.ModuleName
}
