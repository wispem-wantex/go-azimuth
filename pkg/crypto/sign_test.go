package crypto_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-azimuth/pkg/crypto"
)

func hex_to_bytes(s string) []byte {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}
func TestVeinToCrub(t *testing.T) {
	assert := assert.New(t)
	vein := crypto.UrbitVeinFromHex(
		"712f2fa71eac637ccd5d5bdd73229f7b85a61a89facb90573fc9623a895f3f00aed1f34e1480677c4656266" +
			"94e25e7b65afdc6e8d69fd8b4ceee64bd6f4870d842")
	assert.Equal(byte('B'), vein[64])

	crub := vein.ToCrub()
	assert.Equal(hex_to_bytes("712f2fa71eac637ccd5d5bdd73229f7b85a61a89facb90573fc9623a895f3f00"), crub.EncryptKeys.Secret[:])
	// assert.Equal(hex_to_bytes("5d3c62f5ce6738533944364e7639b8f84121760e72eeaaf2b22d5436c4bb36b4"), crub.EncryptKeys.Secret[:])
	assert.Equal(hex_to_bytes("aed1f34e1480677c465626694e25e7b65afdc6e8d69fd8b4ceee64bd6f4870d8"), crub.SignKeys.Secret[:])
	assert.Equal(hex_to_bytes("b81aa63451cc3374a1d4a988262229d9041a0f2d62318e4ecec76c5b07df82fa"), crub.SignKeys.Pub[:])
}

func TestCrubSign(t *testing.T) {
	assert := assert.New(t)
	crub := crypto.UrbitVeinFromHex(
		"712f2fa71eac637ccd5d5bdd73229f7b85a61a89facb90573fc9623a895f3f00aed1f34e1480677c465626694" +
			"e25e7b65afdc6e8d69fd8b4ceee64bd6f4870d842").ToCrub()

	signature := crub.Sign([]byte("hello, world!"))
	assert.Equal(
		hex_to_bytes("0e027836695faaf11d6d3ea47982ea850f7a871f9cff22beecca451f6729f21ef7c9442aa0810034c5ad4ee"+
			"e2fd0985c739f321a71ba2ee6e1ab3ac73b547105"),
		signature,
	)

	assert.True(crub.Verify(signature, []byte("hello, world!")))
}
