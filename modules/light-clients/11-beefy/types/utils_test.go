package types_test

import (
	"encoding/hex"
	"log"
	"testing"

	substrate "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
)

func TestEcRecover(t *testing.T) {
	payload, err := hex.DecodeString("b44ddc7af2d75203036f2ab747701de9d54b9b31461df4c8afcc63d12282c733")
	if err != nil {
		panic(err)
	}
	commitment := substrate.Commitment{
		Payload: substrate.NewH256(payload),
		BlockNumber: substrate.NewU32(785),
		ValidatorSetID: substrate.NewU64(0),
	}
	commitmentBytes, err := types.Encode(commitment)
	if err != nil {
		panic(err)
	}
	digest := crypto.Keccak256(commitmentBytes)

	signatures := []string{
		"c15f45a0c5246a92fd797cf45f716e7d12aad3919b6bae7ce76f9f78851048c516a9cbd8d2decf12dcb152c0a30ac603d09f80a57e58273fc19d357427e1925f01",
		"e477ea675dd5428cbddc51b4d2aa070d79504da5923d4c2149b9fc182c1558ce0b63299eed8f1695f9b3bfd0adc2a101dbcc49b85a7cd18c093087e09e8f3d2700",
		"984b00f53766e4ba63cb48462f141aaa07cde00196d8f564a0d9d29e9324f4e8731c3ff0288359d07909d800a3571920b94f64dfba1dea3932d0a8888c9775dc00",
		"1eadc75000162919a9cfff0d2f6b1b8892fa5c3abdeba0183224a8045e1aa49660766934b0a54d81358f439217929e3ff5d66f7a84f6f69f0b04fa6947fdc74901",
	}

	for _, sig := range signatures {
		sig, err := hex.DecodeString(sig)
		if err != nil {
			panic(err)
		}

		pubkeyBytes, err := crypto.Ecrecover(digest, sig)
		if err != nil {
			log.Fatal(err)
		}
		address := hex.EncodeToString(crypto.Keccak256(pubkeyBytes[1:])[12:])
		
		t.Logf("address: %s", address)
	}
}
