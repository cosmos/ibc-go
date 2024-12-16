package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// TestMigrateDefaultParams tests the migration for the channel params
func (suite *KeeperTestSuite) TestMigrateDefaultParams() {
	testCases := []struct {
		name           string
		expectedParams channeltypes.Params
	}{
		{
			"success: default params",
			channeltypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			ctx := suite.chainA.GetContext()
			migrator := keeper.NewMigrator(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper)
			err := migrator.MigrateParams(ctx)
			suite.Require().NoError(err)

			params := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetParams(ctx)
			suite.Require().Equal(tc.expectedParams, params)
		})
	}
}

// TestMigrateNextSequenceSend will test the migration from the v1 NextSeqSend keys
// to the v2 format.
func (suite *KeeperTestSuite) TestMigrateNextSequenceSend() {
	seq1 := types.NewPacketSequence("transfer", "channel-0", 1)
	seq2 := types.NewPacketSequence("mock", "channel-2", 2)
	seq3 := types.NewPacketSequence("ica", "channel-4", 3)

	expSeqs := []types.PacketSequence{seq1, seq2, seq3}

	store := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetKey(exported.StoreKey))
	for _, es := range expSeqs {
		bz := sdk.Uint64ToBigEndian(es.Sequence)
		store.Set(host.NextSequenceSendKey(es.PortId, es.ChannelId), bz)
	}

	k := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper

	ctx := suite.chainA.GetContext()
	seqs := k.GetAllPacketSendSeqs(ctx)
	suite.Require().Equal([]types.PacketSequence(nil), seqs, "sequences already exist in correct key format")

	migrator := keeper.NewMigrator(k)

	migrator.MigrateNextSequenceSend(ctx)

	expV2Seqs := []types.PacketSequence{}
	for _, es := range expSeqs {
		expV2Seqs = append(expV2Seqs, types.NewPacketSequence("", es.ChannelId, es.Sequence))
	}

	seqs = k.GetAllPacketSendSeqs(ctx)
	suite.Require().Equal(expV2Seqs, seqs, "new sequence keys not stored correctly")
}
