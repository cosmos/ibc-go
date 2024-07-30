package mock

import "github.com/cosmos/ibc-go/v8/modules/core/exported"

var _ exported.ClientMessage = (*MockHeader)(nil)

func (*MockHeader) ClientType() string {
	return ModuleName
}

func (*MockHeader) ValidateBasic() error {
	return nil
}
