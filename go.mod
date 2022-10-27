go 1.15

module github.com/cosmos/ibc-go/v2

retract [v2.2.0, v2.2.2] // depends on SDK version without dragonberry fix

require (
	github.com/armon/go-metrics v0.3.10
	github.com/confio/ics23/go v0.7.0
	github.com/cosmos/cosmos-sdk v0.45.10
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/pkg/errors v0.9.1
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.5.0
	github.com/spf13/viper v1.13.0
	github.com/stretchr/testify v1.8.0
	github.com/tendermint/tendermint v0.34.22
	github.com/tendermint/tm-db v0.6.6
	google.golang.org/genproto v0.0.0-20220725144611-272f38e5d71b
	google.golang.org/grpc v1.50.0
	google.golang.org/protobuf v1.28.1
)

require github.com/regen-network/cosmos-proto v0.3.1

replace (
	// dragonberry replace for ics23
	github.com/confio/ics23/go => github.com/cosmos/cosmos-sdk/ics23/go v0.8.0

	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
)
