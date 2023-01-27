package wasm_test

import (
	"encoding/hex"
	"os"

	wasm "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm"
)

func (suite *WasmTestSuite) TestPushNewWasmCode() {
	data, err := os.ReadFile("test_data/example.wasm")
	suite.Require().NoError(err)

	//test pushing a valid wasm code
	codeId, err := suite.wasmKeeper.PushNewWasmCode(suite.ctx, data)
	suite.Require().NoError(err)
	suite.Require().NotNil(codeId)

	//test wasmcode duplication
	codeId, err = suite.wasmKeeper.PushNewWasmCode(suite.ctx, data)
	suite.Require().Error(err)

	//test invalid wasm code
	codeId, err = suite.wasmKeeper.PushNewWasmCode(suite.ctx, []byte{})
	suite.Require().Error(err)
}

func (suite *WasmTestSuite) TestQueryWasmCode() {
	data, err := os.ReadFile("test_data/example2.wasm")
	suite.Require().NoError(err)

	//push a new wasm code
	codeId, err := suite.wasmKeeper.PushNewWasmCode(suite.ctx, data)
	suite.Require().NoError(err)
	suite.Require().NotNil(codeId)

	//test invalid query request
	_, err = suite.wasmKeeper.WasmCode(suite.ctx, &wasm.WasmCodeQuery{})
	suite.Require().Error(err)

	_, err = suite.wasmKeeper.WasmCode(suite.ctx, &wasm.WasmCodeQuery{CodeId: "test"})
	suite.Require().Error(err)

	//test valid query request
	res, err := suite.wasmKeeper.WasmCode(suite.ctx, &wasm.WasmCodeQuery{CodeId: hex.EncodeToString(codeId)})
	suite.Require().NoError(err)
	suite.Require().NotNil(res.Code)
}
