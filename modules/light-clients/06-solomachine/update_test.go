package solomachine_test

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *SoloMachineTestSuite) TestVerifyClientMessageHeader() {
	var (
		clientMsg   exported.ClientMessage
		clientState *solomachine.ClientState
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {

		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful header",
				func() {
					clientMsg = sm.CreateHeader(sm.Diversifier)
				},
				true,
			},
			{
				"successful header with new diversifier",
				func() {
					clientMsg = sm.CreateHeader(sm.Diversifier + "0")
				},
				true,
			},
			{
				"successful misbehaviour",
				func() {
					clientMsg = sm.CreateMisbehaviour()
				},
				true,
			},
			{
				"invalid client message type",
				func() {
					clientMsg = &ibctm.Header{}
				},
				false,
			},
			{
				"invalid header Signature",
				func() {
					h := sm.CreateHeader(sm.Diversifier)
					h.Signature = s.GetInvalidProof()
					clientMsg = h
				}, false,
			},
			{
				"invalid timestamp in header",
				func() {
					h := sm.CreateHeader(sm.Diversifier)
					h.Timestamp--
					clientMsg = h
				}, false,
			},
			{
				"signature uses wrong sequence",
				func() {
					sm.Sequence++
					clientMsg = sm.CreateHeader(sm.Diversifier)
				},
				false,
			},
			{
				"signature uses new pubkey to sign",
				func() {
					// store in temp before assinging to interface type
					cs := sm.ClientState()
					h := sm.CreateHeader(sm.Diversifier)

					publicKey, err := codectypes.NewAnyWithValue(sm.PublicKey)
					s.NoError(err)

					data := &solomachine.HeaderData{
						NewPubKey:      publicKey,
						NewDiversifier: h.NewDiversifier,
					}

					dataBz, err := s.chainA.Codec.Marshal(data)
					s.Require().NoError(err)

					// generate invalid signature
					signBytes := &solomachine.SignBytes{
						Sequence:    cs.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        dataBz,
					}

					signBz, err := s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)
					s.Require().NoError(err)
					h.Signature = sig

					clientState = cs
					clientMsg = h
				},
				false,
			},
			{
				"signature signs over old pubkey",
				func() {
					// store in temp before assinging to interface type
					cs := sm.ClientState()
					oldPubKey := sm.PublicKey
					h := sm.CreateHeader(sm.Diversifier)

					// generate invalid signature
					data := append(sdk.Uint64ToBigEndian(cs.Sequence), oldPubKey.Bytes()...)
					sig := sm.GenerateSignature(data)
					h.Signature = sig

					clientState = cs
					clientMsg = h
				},
				false,
			},
			{
				"consensus state public key is nil - header",
				func() {
					clientState.ConsensusState.PublicKey = nil
					clientMsg = sm.CreateHeader(sm.Diversifier)
				},
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				clientState = sm.ClientState()

				// setup test
				tc.setup()

				err := clientState.VerifyClientMessage(s.chainA.GetContext(), s.chainA.Codec, s.store, clientMsg)

				if tc.expPass {
					s.Require().NoError(err)
				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestVerifyClientMessageMisbehaviour() {
	var (
		clientMsg   exported.ClientMessage
		clientState *solomachine.ClientState
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {

		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful misbehaviour",
				func() {
					clientMsg = sm.CreateMisbehaviour()
				},
				true,
			},
			{
				"old misbehaviour is successful (timestamp is less than current consensus state)",
				func() {
					clientState = sm.ClientState()
					sm.Time -= 5
					clientMsg = sm.CreateMisbehaviour()
				}, true,
			},
			{
				"invalid client message type",
				func() {
					clientMsg = &ibctm.Header{}
				},
				false,
			},
			{
				"consensus state pubkey is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
					clientMsg = sm.CreateMisbehaviour()
				},
				false,
			},
			{
				"invalid SignatureOne SignatureData",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureOne.Signature = s.GetInvalidProof()
					clientMsg = m
				}, false,
			},
			{
				"invalid SignatureTwo SignatureData",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureTwo.Signature = s.GetInvalidProof()
					clientMsg = m
				}, false,
			},
			{
				"invalid SignatureOne timestamp",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureOne.Timestamp = 1000000000000
					clientMsg = m
				}, false,
			},
			{
				"invalid SignatureTwo timestamp",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureTwo.Timestamp = 1000000000000
					clientMsg = m
				}, false,
			},
			{
				"invalid first signature data",
				func() {
					// store in temp before assigning to interface type
					m := sm.CreateMisbehaviour()

					msg := []byte("DATA ONE")
					signBytes := &solomachine.SignBytes{
						Sequence:    sm.Sequence + 1,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        msg,
					}

					data, err := s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(data)

					m.SignatureOne.Signature = sig
					m.SignatureOne.Data = msg
					clientMsg = m
				},
				false,
			},
			{
				"invalid second signature data",
				func() {
					// store in temp before assigning to interface type
					m := sm.CreateMisbehaviour()

					msg := []byte("DATA TWO")
					signBytes := &solomachine.SignBytes{
						Sequence:    sm.Sequence + 1,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        msg,
					}

					data, err := s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(data)

					m.SignatureTwo.Signature = sig
					m.SignatureTwo.Data = msg
					clientMsg = m
				},
				false,
			},
			{
				"wrong pubkey generates first signature",
				func() {
					badMisbehaviour := sm.CreateMisbehaviour()

					// update public key to a new one
					sm.CreateHeader(sm.Diversifier)
					m := sm.CreateMisbehaviour()

					// set SignatureOne to use the wrong signature
					m.SignatureOne = badMisbehaviour.SignatureOne
					clientMsg = m
				}, false,
			},
			{
				"wrong pubkey generates second signature",
				func() {
					badMisbehaviour := sm.CreateMisbehaviour()

					// update public key to a new one
					sm.CreateHeader(sm.Diversifier)
					m := sm.CreateMisbehaviour()

					// set SignatureTwo to use the wrong signature
					m.SignatureTwo = badMisbehaviour.SignatureTwo
					clientMsg = m
				}, false,
			},
			{
				"signatures sign over different sequence",
				func() {
					// store in temp before assigning to interface type
					m := sm.CreateMisbehaviour()

					// Signature One
					msg := []byte("DATA ONE")
					// sequence used is plus 1
					signBytes := &solomachine.SignBytes{
						Sequence:    sm.Sequence + 1,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        msg,
					}

					data, err := s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(data)

					m.SignatureOne.Signature = sig
					m.SignatureOne.Data = msg

					// Signature Two
					msg = []byte("DATA TWO")
					// sequence used is minus 1

					signBytes = &solomachine.SignBytes{
						Sequence:    sm.Sequence - 1,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        msg,
					}
					data, err = s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

					sig = sm.GenerateSignature(data)

					m.SignatureTwo.Signature = sig
					m.SignatureTwo.Data = msg

					clientMsg = m
				},
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				clientState = sm.ClientState()

				// setup test
				tc.setup()

				err := clientState.VerifyClientMessage(s.chainA.GetContext(), s.chainA.Codec, s.store, clientMsg)

				if tc.expPass {
					s.Require().NoError(err)
				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestUpdateState() {
	var (
		clientState exported.ClientState
		clientMsg   exported.ClientMessage
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {

		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful update",
				func() {
					clientState = sm.ClientState()
					clientMsg = sm.CreateHeader(sm.Diversifier)
				},
				true,
			},
			{
				"invalid type misbehaviour",
				func() {
					clientState = sm.ClientState()
					clientMsg = sm.CreateMisbehaviour()
				},
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				tc.setup() // setup test

				if tc.expPass {
					consensusHeights := clientState.UpdateState(s.chainA.GetContext(), s.chainA.Codec, s.store, clientMsg)

					clientStateBz := s.store.Get(host.ClientStateKey())
					s.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(s.chainA.Codec, clientStateBz)

					s.Require().Len(consensusHeights, 1)
					s.Require().Equal(uint64(0), consensusHeights[0].GetRevisionNumber())
					s.Require().Equal(newClientState.(*solomachine.ClientState).Sequence, consensusHeights[0].GetRevisionHeight())

					s.Require().False(newClientState.(*solomachine.ClientState).IsFrozen)
					s.Require().Equal(clientMsg.(*solomachine.Header).NewPublicKey, newClientState.(*solomachine.ClientState).ConsensusState.PublicKey)
					s.Require().Equal(clientMsg.(*solomachine.Header).NewDiversifier, newClientState.(*solomachine.ClientState).ConsensusState.Diversifier)
					s.Require().Equal(clientMsg.(*solomachine.Header).Timestamp, newClientState.(*solomachine.ClientState).ConsensusState.Timestamp)
				} else {
					s.Require().Panics(func() {
						clientState.UpdateState(s.chainA.GetContext(), s.chainA.Codec, s.store, clientMsg)
					})
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestCheckForMisbehaviour() {
	var clientMsg exported.ClientMessage

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
		testCases := []struct {
			name     string
			malleate func()
			expPass  bool
		}{
			{
				"success",
				func() {
					clientMsg = sm.CreateMisbehaviour()
				},
				true,
			},
			{
				"normal header returns false",
				func() {
					clientMsg = sm.CreateHeader(sm.Diversifier)
				},
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				clientState := sm.ClientState()

				tc.malleate()

				foundMisbehaviour := clientState.CheckForMisbehaviour(s.chainA.GetContext(), s.chainA.Codec, s.store, clientMsg)

				if tc.expPass {
					s.Require().True(foundMisbehaviour)
				} else {
					s.Require().False(foundMisbehaviour)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestUpdateStateOnMisbehaviour() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
		testCases := []struct {
			name     string
			malleate func()
			expPass  bool
		}{
			{
				"success",
				func() {},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				clientState := sm.ClientState()

				tc.malleate()

				clientState.UpdateStateOnMisbehaviour(s.chainA.GetContext(), s.chainA.Codec, s.store, nil)

				if tc.expPass {
					clientStateBz := s.store.Get(host.ClientStateKey())
					s.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(s.chainA.Codec, clientStateBz)

					s.Require().True(newClientState.(*solomachine.ClientState).IsFrozen)
				}
			})
		}
	}
}
