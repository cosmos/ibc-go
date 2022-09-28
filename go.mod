go 1.18

module github.com/cosmos/ibc-go/v6

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require (
	cosmossdk.io/math v1.0.0-beta.3
	github.com/armon/go-metrics v0.4.1
	github.com/confio/ics23/go v0.7.0
	github.com/cosmos/cosmos-sdk v0.46.1
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/rakyll/statik v0.1.7
	github.com/regen-network/cosmos-proto v0.3.1
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.5.0
	github.com/spf13/viper v1.13.0
	github.com/stretchr/testify v1.8.0
	github.com/tendermint/tendermint v0.34.21
	github.com/tendermint/tm-db v0.6.7
	google.golang.org/genproto v0.0.0-20220725144611-272f38e5d71b
	google.golang.org/grpc v1.49.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/gin-gonic/gin v1.7.0 // indirect
	github.com/zondax/hid v0.9.1-0.20220302062450-5552068d2266 // indirect
)
