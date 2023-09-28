go 1.19

module github.com/cosmos/ibc-go/v4

retract (
	v4.4.0 // contains huckleberry vulnerability
	v4.3.0 // contains huckleberry vulnerability
	v4.2.1 // contains state machine breaking change
	v4.2.0 // contains huckleberry vulnerability
	v4.1.2 // contains state machine breaking change
	v4.1.1 // contains huckleberry vulnerability
	[v4.0.0, v4.1.0] // depends on SDK version without dragonberry fix
)

require (
	github.com/armon/go-metrics v0.4.1
	github.com/confio/ics23/go v0.9.1
	github.com/cosmos/cosmos-sdk v0.45.16
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/rakyll/statik v0.1.7
	github.com/regen-network/cosmos-proto v0.3.1
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.14.0
	github.com/stretchr/testify v1.8.1
	github.com/tendermint/tendermint v0.34.27
	github.com/tendermint/tm-db v0.6.7
	google.golang.org/genproto v0.0.0-20230125152338-dcaf20b6aeaa
	google.golang.org/grpc v1.52.3
	google.golang.org/protobuf v1.28.2-0.20220831092852-f930b1dc76e8
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/tendermint/tendermint => github.com/cometbft/cometbft v0.34.27
)
