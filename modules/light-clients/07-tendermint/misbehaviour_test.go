package tendermint_test

import (
	"errors"
	"time"

	errorsmod "cosmossdk.io/errors"

	"github.com/cometbft/cometbft/crypto/tmhash"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *TendermintTestSuite) TestMisbehaviour() {
	heightMinus1 := clienttypes.NewHeight(0, height.RevisionHeight-1)

	misbehaviour := &ibctm.Misbehaviour{
		Header1:  s.header,
		Header2:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now, s.valSet, s.valSet, s.valSet, s.signers),
		ClientId: clientID,
	}

	s.Require().Equal(exported.Tendermint, misbehaviour.ClientType())
}

func (s *TendermintTestSuite) TestMisbehaviourValidateBasic() {
	altPrivVal := cmttypes.NewMockPV()
	altPubKey, err := altPrivVal.GetPubKey()
	s.Require().NoError(err)

	revisionHeight := int64(height.RevisionHeight)

	altVal := cmttypes.NewValidator(altPubKey, revisionHeight)

	// Create alternative validator set with only altVal
	altValSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{altVal})

	// Create signer array and ensure it is in same order as bothValSet
	bothValSet, bothSigners := getBothSigners(s, altVal, altPrivVal)

	altSignerArr := []cmttypes.PrivValidator{altPrivVal}

	heightMinus1 := clienttypes.NewHeight(0, height.RevisionHeight-1)

	testCases := []struct {
		name                 string
		misbehaviour         *ibctm.Misbehaviour
		malleateMisbehaviour func(misbehaviour *ibctm.Misbehaviour) error
		expErr               error
	}{
		{
			"valid fork misbehaviour, two headers at same height have different time",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now.Add(time.Minute), s.valSet, s.valSet, s.valSet, s.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			nil,
		},
		{
			"valid time misbehaviour, both headers at different heights are at same time",
			&ibctm.Misbehaviour{
				Header1:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight+5), heightMinus1, s.now, s.valSet, s.valSet, s.valSet, s.signers),
				Header2:  s.header,
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			nil,
		},
		{
			"misbehaviour Header1 is nil",
			ibctm.NewMisbehaviour(clientID, nil, s.header),
			func(m *ibctm.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibctm.ErrInvalidHeader, "misbehaviour Header1 cannot be nil"),
		},
		{
			"misbehaviour Header2 is nil",
			ibctm.NewMisbehaviour(clientID, s.header, nil),
			func(m *ibctm.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibctm.ErrInvalidHeader, "misbehaviour Header2 cannot be nil"),
		},
		{
			"valid misbehaviour with different trusted headers",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), clienttypes.NewHeight(0, height.RevisionHeight-3), s.now.Add(time.Minute), s.valSet, s.valSet, bothValSet, s.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			nil,
		},
		{
			"trusted height is 0 in Header1",
			&ibctm.Misbehaviour{
				Header1:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), clienttypes.ZeroHeight(), s.now.Add(time.Minute), s.valSet, s.valSet, s.valSet, s.signers),
				Header2:  s.header,
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibctm.ErrInvalidHeaderHeight, "misbehaviour Header1 cannot have zero revision height"),
		},
		{
			"trusted height is 0 in Header2",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), clienttypes.ZeroHeight(), s.now.Add(time.Minute), s.valSet, s.valSet, s.valSet, s.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibctm.ErrInvalidHeaderHeight, "misbehaviour Header2 cannot have zero revision height"),
		},
		{
			"trusted valset is nil in Header1",
			&ibctm.Misbehaviour{
				Header1:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now.Add(time.Minute), s.valSet, s.valSet, nil, s.signers),
				Header2:  s.header,
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibctm.ErrInvalidValidatorSet, "trusted validator set in Header1 cannot be empty"),
		},
		{
			"trusted valset is nil in Header2",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now.Add(time.Minute), s.valSet, s.valSet, nil, s.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibctm.ErrInvalidValidatorSet, "trusted validator set in Header2 cannot be empty"),
		},
		{
			"invalid client ID ",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now, s.valSet, s.valSet, s.valSet, s.signers),
				ClientId: "GAI",
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			errors.New("identifier GAI has invalid length"),
		},
		{
			"chainIDs do not match",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader("ethermint", int64(height.RevisionHeight), heightMinus1, s.now, s.valSet, s.valSet, s.valSet, s.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			errorsmod.Wrap(clienttypes.ErrInvalidMisbehaviour, "headers must have identical chainIDs"),
		},
		{
			"header2 height is greater",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader(chainID, 6, clienttypes.NewHeight(0, height.RevisionHeight+1), s.now, s.valSet, s.valSet, s.valSet, s.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error { return nil },
			errors.New("Header1 height is less than Header2 height"),
		},
		{
			"header 1 doesn't have 2/3 majority",
			&ibctm.Misbehaviour{
				Header1:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now, bothValSet, bothValSet, s.valSet, bothSigners),
				Header2:  s.header,
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error {
				// voteSet contains only altVal which is less than 2/3 of total power (height/1height)
				wrongVoteSet := cmttypes.NewVoteSet(chainID, int64(misbehaviour.Header1.GetHeight().GetRevisionHeight()), 1, cmtproto.PrecommitType, altValSet)
				blockID, err := cmttypes.BlockIDFromProto(&misbehaviour.Header1.Commit.BlockID)
				if err != nil {
					return err
				}

				extCommit, err := cmttypes.MakeExtCommit(*blockID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), misbehaviour.Header1.Commit.Round, wrongVoteSet, altSignerArr, s.now, false)
				misbehaviour.Header1.Commit = extCommit.ToCommit().ToProto()
				return err
			},
			errors.New("validator set did not commit to header"),
		},
		{
			"header 2 doesn't have 2/3 majority",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now, bothValSet, bothValSet, s.valSet, bothSigners),
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error {
				// voteSet contains only altVal which is less than 2/3 of total power (height/1height)
				wrongVoteSet := cmttypes.NewVoteSet(chainID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), 1, cmtproto.PrecommitType, altValSet)
				blockID, err := cmttypes.BlockIDFromProto(&misbehaviour.Header2.Commit.BlockID)
				if err != nil {
					return err
				}

				extCommit, err := cmttypes.MakeExtCommit(*blockID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), misbehaviour.Header2.Commit.Round, wrongVoteSet, altSignerArr, s.now, false)
				misbehaviour.Header2.Commit = extCommit.ToCommit().ToProto()
				return err
			},
			errors.New("validator set did not commit to header"),
		},
		{
			"validators sign off on wrong commit",
			&ibctm.Misbehaviour{
				Header1:  s.header,
				Header2:  s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now, bothValSet, bothValSet, s.valSet, bothSigners),
				ClientId: clientID,
			},
			func(misbehaviour *ibctm.Misbehaviour) error {
				tmBlockID := ibctesting.MakeBlockID(tmhash.Sum([]byte("other_hash")), 3, tmhash.Sum([]byte("other_partset")))
				misbehaviour.Header2.Commit.BlockID = tmBlockID.ToProto()
				return nil
			},
			errors.New("header 2 failed validation"),
		},
	}

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.malleateMisbehaviour(tc.misbehaviour)
			s.Require().NoError(err)
			err = tc.misbehaviour.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
			} else {
				s.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
