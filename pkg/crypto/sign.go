package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
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
	ret.IsPublic = false
	return
}

type HexField [32]byte

// MarshalJSON converts [32]byte to a hex-encoded string for JSON
func (h HexField) MarshalJSON() ([]byte, error) {
	ret, err := json.Marshal(fmt.Sprintf("%x", h[:])) // Convert to slice to hex-encode
	if err != nil {
		return nil, fmt.Errorf("can't decode hex field %x: %w", h, err)
	}
	return ret, nil
}

// UnmarshalJSON parses a hex-encoded string back into [32]byte
func (h *HexField) UnmarshalJSON(data []byte) error {
	var hexStr string
	if err := json.Unmarshal(data, &hexStr); err != nil {
		return fmt.Errorf("invalid json %s: %w", string(data), err)
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return fmt.Errorf("can't decode %q: %w", hexStr, err)
	}
	if len(decoded) != 32 {
		return fmt.Errorf("expected 32 bytes, got %d", len(decoded))
	}
	copy(h[:], decoded)
	return nil
}

type UrbitCrub struct {
	IsPublic bool
	SignKeys struct {
		Pub    HexField `json:"pub"`
		Secret HexField `json:"priv"`
	} `json:"signature"`
	EncryptKeys struct {
		Pub    HexField `json:"pub"`
		Secret HexField `json:"priv"`
	} `json:"encryption"`
}

func (c UrbitCrub) Sign(msg []byte) []byte {
	privkey := ed25519.NewKeyFromSeed(reverse(c.SignKeys.Secret[:]))
	signature := ed25519.Sign(privkey, msg)
	return reverse(signature)
}

func (c UrbitCrub) Verify(signature []byte, msg []byte) bool {
	return ed25519.Verify(reverse(c.SignKeys.Pub[:]), msg, reverse(signature))
}
