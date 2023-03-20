package types_test

func (suite *WasmTestSuite) TestExportGenesis() {
	suite.SetupWithEmptyClient()
	gm := suite.clientState.ExportMetadata(suite.store)
	suite.Require().NotNil(gm, "client returned nil")
	suite.Require().Len(gm, 0, "exported metadata has unexpected length")
}