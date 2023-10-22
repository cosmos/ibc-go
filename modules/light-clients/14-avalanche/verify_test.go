package avalanche_test

import (
	"context"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/validators"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/crypto/bls"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	ibcava "github.com/cosmos/ibc-go/v8/modules/light-clients/14-avalanche"
)

const pChainHeight uint64 = 1337

var (
	_ utils.Sortable[*testValidator] = (*testValidator)(nil)

	sourceChainID = ids.GenerateTestID()
	subnetID      = ids.GenerateTestID()

	testVdrs []*testValidator
)

type testValidator struct {
	nodeID ids.NodeID
	sk     *bls.SecretKey
	vdr    *warp.Validator
}

func (v *testValidator) Less(o *testValidator) bool {
	return v.vdr.Less(o.vdr)
}

func newTestValidator() *testValidator {
	sk, err := bls.NewSecretKey()
	if err != nil {
		panic(err)
	}

	nodeID := ids.GenerateTestNodeID()
	pk := bls.PublicFromSecretKey(sk)
	return &testValidator{
		nodeID: nodeID,
		sk:     sk,
		vdr: &warp.Validator{
			PublicKey:      pk,
			PublicKeyBytes: bls.PublicKeyToBytes(pk),
			Weight:         3,
			NodeIDs:        []ids.NodeID{nodeID},
		},
	}
}

func (suite *AvalancheTestSuite) TestSignatureVerification() {
	testVdrs = []*testValidator{
		newTestValidator(),
		newTestValidator(),
		newTestValidator(),
	}
	utils.Sort(testVdrs)
	vdrs := map[ids.NodeID]*validators.GetValidatorOutput{
		testVdrs[0].nodeID: {
			NodeID:    testVdrs[0].nodeID,
			PublicKey: testVdrs[0].vdr.PublicKey,
			Weight:    testVdrs[0].vdr.Weight,
		},
		testVdrs[1].nodeID: {
			NodeID:    testVdrs[1].nodeID,
			PublicKey: testVdrs[1].vdr.PublicKey,
			Weight:    testVdrs[1].vdr.Weight,
		},
		testVdrs[2].nodeID: {
			NodeID:    testVdrs[2].nodeID,
			PublicKey: testVdrs[2].vdr.PublicKey,
			Weight:    testVdrs[2].vdr.Weight,
		},
	}

	networkID := uint32(1)

	tests := []struct {
		name      string
		stateF    func(*gomock.Controller) validators.State
		quorumNum uint64
		quorumDen uint64
		msgF      func(*require.Assertions) ([]byte, [96]byte, *warp.UnsignedMessage)
		valid     bool
	}{
		{
			name: "valid signature",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, subnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) ([]byte, [96]byte, *warp.UnsignedMessage) {
				unsignedMsg, err := warp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					[]byte{1, 2, 3},
				)
				require.NoError(err)

				// [signers] has weight from [vdr[1], vdr[2]],
				// which is 6, which is greater than 4.5
				signers := set.NewBits()
				signers.Add(1)
				signers.Add(2)
				signersInput := signers.Bytes()

				unsignedBytes := unsignedMsg.Bytes()
				vdr1Sig := bls.Sign(testVdrs[1].sk, unsignedBytes)
				vdr2Sig := bls.Sign(testVdrs[2].sk, unsignedBytes)
				aggSig, err := bls.AggregateSignatures([]*bls.Signature{vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(aggSig))

				require.NoError(err)
				return signersInput, aggSigBytes, unsignedMsg
			},
			valid: true,
		},
		{
			name: "invalid quorumNum quorumDen",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validators.NewMockState(ctrl)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, subnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 2,
			quorumDen: 2,
			msgF: func(require *require.Assertions) ([]byte, [96]byte, *warp.UnsignedMessage) {
				unsignedMsg, err := warp.NewUnsignedMessage(
					networkID,
					sourceChainID,
					[]byte{1, 2, 3},
				)
				require.NoError(err)

				// [signers] has weight from [vdr[1], vdr[2]],
				// which is 6, which is greater than 4.5
				signers := set.NewBits()
				signers.Add(1)
				signers.Add(2)
				signers_input := signers.Bytes()

				unsignedBytes := unsignedMsg.Bytes()
				vdr1Sig := bls.Sign(testVdrs[1].sk, unsignedBytes)
				vdr2Sig := bls.Sign(testVdrs[2].sk, unsignedBytes)
				aggSig, err := bls.AggregateSignatures([]*bls.Signature{vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [bls.SignatureLen]byte{}
				copy(aggSigBytes[:], bls.SignatureToBytes(aggSig))

				require.NoError(err)
				return signers_input, aggSigBytes, unsignedMsg
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			require := require.New(suite.T())
			ctrl := gomock.NewController(suite.T())
			defer ctrl.Finish()

			signersInput, signature, unsignedMsg := tt.msgF(require)
			pChainState := tt.stateF(ctrl)

			unsignedBytes := unsignedMsg.Bytes()

			vdrsIn, totalWeight, err := warp.GetCanonicalValidatorSet(context.Background(), pChainState, pChainHeight, subnetID)
			if err != nil {
				panic(err)
			}

			err = ibcava.VerifyBls(signersInput, signature, unsignedBytes, vdrsIn, totalWeight,
				tt.quorumNum,
				tt.quorumDen)

			if tt.valid && err != nil {
				require.Error(err)
			}
		})
	}
}
