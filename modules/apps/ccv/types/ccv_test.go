package types_test

import (
	"testing"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestPacketDataValidateBasic(t *testing.T) {
	pk1, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	pk2, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)

	cases := []struct {
		name       string
		expError   bool
		packetData types.ValidatorSetChangePacketData
	}{
		{
			"nil packet data",
			true,
			types.NewValidatorSetChangePacketData(nil),
		},
		{
			"empty packet data",
			true,
			types.NewValidatorSetChangePacketData([]abci.ValidatorUpdate{}),
		},
		{
			"valid packet data",
			false,
			types.NewValidatorSetChangePacketData(
				[]abci.ValidatorUpdate{
					{
						PubKey: pk1,
						Power:  30,
					},
					{
						PubKey: pk2,
						Power:  20,
					},
				},
			),
		},
	}

	for _, c := range cases {
		err := c.packetData.ValidateBasic()
		if c.expError {
			require.Error(t, err, "%s invalid but passed ValidateBasic", c.name)
		} else {
			require.NoError(t, err, "%s valid but ValidateBasic returned error: %w", c.name, err)
		}
	}
}

func TestMarshalPacketData(t *testing.T) {
	pk1, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	pk2, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)

	vpd := types.NewValidatorSetChangePacketData(
		[]abci.ValidatorUpdate{
			{
				PubKey: pk1,
				Power:  30,
			},
			{
				PubKey: pk2,
				Power:  20,
			},
		},
	)

	bz, err := vpd.Marshal()
	require.NoError(t, err, "marshalling packet data returned error")

	recovered := types.ValidatorSetChangePacketData{}
	recovered.Unmarshal(bz)

	require.Equal(t, vpd, recovered, "unmarshaled packet data does not equal original value")
}
