package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
)

func reverse(data []byte) []byte {
	reversed := make([]byte, len(data))
	for i, b := range data {
		reversed[len(data)-1-i] = b
	}
	return reversed
}

type UrbitVein [65]byte // ship private key
func UrbitVeinFromHex(s string) (ret UrbitVein) {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	if len(data) != 65 {
		panic(s)
	}
	copy(ret[:], data[:])
	return
}

// Effectively equivalent to `nol:nu:crub:crypto` from `zuse.hoon`
func (v UrbitVein) ToCrub() (ret UrbitCrub) {
	if v[64] != 'B' {
		panic(v)
	}
	copy(ret.EncryptKeys.Secret[:], v[0:32])
	copy(ret.EncryptKeys.Pub[:], reverse(ed25519.NewKeyFromSeed(reverse(ret.EncryptKeys.Secret[:])).Public().(ed25519.PublicKey)))
	copy(ret.SignKeys.Secret[:], v[32:64])
	copy(ret.SignKeys.Pub[:], reverse(ed25519.NewKeyFromSeed(reverse(ret.SignKeys.Secret[:])).Public().(ed25519.PublicKey)))
	return
}

type UrbitCrub struct {
	SignKeys struct {
		Pub    [32]byte
		Secret [32]byte
	}
	EncryptKeys struct {
		Pub    [32]byte
		Secret [32]byte
	}
}

func (c UrbitCrub) Sign(msg []byte) []byte {
	privkey := ed25519.NewKeyFromSeed(reverse(c.SignKeys.Secret[:]))
	fmt.Printf("privkey (sign): %x\n", privkey)
	signature := ed25519.Sign(privkey, msg)
	fmt.Printf("signautre (sign): %x\n", signature)
	fmt.Printf("public: %x\n", privkey.Public())
	return reverse(signature)
}

func (c UrbitCrub) Verify(signature []byte, msg []byte) bool {
	return ed25519.Verify(reverse(c.SignKeys.Pub[:]), msg, reverse(signature))
}
