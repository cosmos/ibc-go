package tendermint_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// expected export ordering:
// processed height and processed time per height
// then all iteration keys
func (s *TendermintTestSuite) TestExportMetadata() {
	// test intializing client and exporting metadata
	path := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(path)
	clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
	clientState := path.EndpointA.GetClientState()
	height := clientState.GetLatestHeight()

	initIteration := ibctm.GetIterationKey(clientStore, height)
	s.Require().NotEqual(0, len(initIteration))
	initProcessedTime, found := ibctm.GetProcessedTime(clientStore, height)
	s.Require().True(found)
	initProcessedHeight, found := ibctm.GetProcessedHeight(clientStore, height)
	s.Require().True(found)

	gm := clientState.ExportMetadata(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID))
	s.Require().NotNil(gm, "client with metadata returned nil exported metadata")
	s.Require().Len(gm, 3, "exported metadata has unexpected length")

	s.Require().Equal(ibctm.ProcessedHeightKey(height), gm[0].GetKey(), "metadata has unexpected key")
	actualProcessedHeight, err := clienttypes.ParseHeight(string(gm[0].GetValue()))
	s.Require().NoError(err)
	s.Require().Equal(initProcessedHeight, actualProcessedHeight, "metadata has unexpected value")

	s.Require().Equal(ibctm.ProcessedTimeKey(height), gm[1].GetKey(), "metadata has unexpected key")
	s.Require().Equal(initProcessedTime, sdk.BigEndianToUint64(gm[1].GetValue()), "metadata has unexpected value")

	s.Require().Equal(ibctm.IterationKey(height), gm[2].GetKey(), "metadata has unexpected key")
	s.Require().Equal(initIteration, gm[2].GetValue(), "metadata has unexpected value")

	// test updating client and exporting metadata
	err = path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	clientState = path.EndpointA.GetClientState()
	updateHeight := clientState.GetLatestHeight()

	iteration := ibctm.GetIterationKey(clientStore, updateHeight)
	s.Require().NotEqual(0, len(initIteration))
	processedTime, found := ibctm.GetProcessedTime(clientStore, updateHeight)
	s.Require().True(found)
	processedHeight, found := ibctm.GetProcessedHeight(clientStore, updateHeight)
	s.Require().True(found)

	gm = clientState.ExportMetadata(s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID))
	s.Require().NotNil(gm, "client with metadata returned nil exported metadata")
	s.Require().Len(gm, 6, "exported metadata has unexpected length")

	// expected ordering:
	// initProcessedHeight, initProcessedTime, processedHeight, processedTime, initIteration, iteration

	// check init processed height and time
	s.Require().Equal(ibctm.ProcessedHeightKey(height), gm[0].GetKey(), "metadata has unexpected key")
	actualProcessedHeight, err = clienttypes.ParseHeight(string(gm[0].GetValue()))
	s.Require().NoError(err)
	s.Require().Equal(initProcessedHeight, actualProcessedHeight, "metadata has unexpected value")

	s.Require().Equal(ibctm.ProcessedTimeKey(height), gm[1].GetKey(), "metadata has unexpected key")
	s.Require().Equal(initProcessedTime, sdk.BigEndianToUint64(gm[1].GetValue()), "metadata has unexpected value")

	// check processed height and time after update
	s.Require().Equal(ibctm.ProcessedHeightKey(updateHeight), gm[2].GetKey(), "metadata has unexpected key")
	actualProcessedHeight, err = clienttypes.ParseHeight(string(gm[2].GetValue()))
	s.Require().NoError(err)
	s.Require().Equal(processedHeight, actualProcessedHeight, "metadata has unexpected value")

	s.Require().Equal(ibctm.ProcessedTimeKey(updateHeight), gm[3].GetKey(), "metadata has unexpected key")
	s.Require().Equal(processedTime, sdk.BigEndianToUint64(gm[3].GetValue()), "metadata has unexpected value")

	// check iteration keys
	s.Require().Equal(ibctm.IterationKey(height), gm[4].GetKey(), "metadata has unexpected key")
	s.Require().Equal(initIteration, gm[4].GetValue(), "metadata has unexpected value")

	s.Require().Equal(ibctm.IterationKey(updateHeight), gm[5].GetKey(), "metadata has unexpected key")
	s.Require().Equal(iteration, gm[5].GetValue(), "metadata has unexpected value")
}
