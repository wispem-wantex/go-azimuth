package phonemes_test

import (
	// "fmt"
	"math/rand"
	"testing"

	"go-azimuth/pkg/phonemes"

	"github.com/spaolacci/murmur3"
	"github.com/stretchr/testify/assert"
)

func TestPrf(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(uint32(3_602_081_716), murmur3.Sum32WithSeed([]byte{106, 101}, 6))
	assert.Equal(uint32(2_334_372_916), murmur3.Sum32WithSeed([]byte{178, 14}, 0xb76d_5eed))

	v, is_ok := phonemes.PhonemeQToInt("wispem")
	assert.True(is_ok)
	assert.Equal(uint32(2_334_372_916), phonemes.Prf(0, uint16(v)))
	assert.Equal(uint32(1_163_263_495), phonemes.Prf(1, uint16(v)))
}

// Feistel ciphers
// ---------------

// Test forward scramble
func TestFein(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(uint32(776_343_932), phonemes.Fein(11))

	test_cases := []struct {
		Shipname string
		FeVal    uint32
	}{
		{"wispem", 2_758_177_868},
		{"wantex", 3_433_045_696},
		{"wispem-wantex", 2_849_588_615},
	}
	for _, c := range test_cases {
		v, is_ok := phonemes.PhonemeQToInt(c.Shipname)
		assert.True(is_ok)
		assert.Equal(c.FeVal, phonemes.Fein(uint32(v)))
	}
}

// Test reverse scramble (i.e., unscramble)
func TestFynd(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(uint32(11), phonemes.Fynd(uint32(776_343_932)))
	test_cases := []struct {
		Shipname string
		FeVal    uint32
	}{
		{"wispem", 2_758_177_868},
		{"wantex", 3_433_045_696},
		{"wispem-wantex", 2_849_588_615},
	}
	for _, c := range test_cases {
		v, is_ok := phonemes.PhonemeQToInt(c.Shipname)
		assert.True(is_ok)
		assert.Equal(uint32(v), phonemes.Fynd(c.FeVal))
	}
}

func TestScrambleUnscramble(t *testing.T) {
	assert := assert.New(t)

	for i := 0; i < 10000; i++ {
		v := rand.Uint32()
		assert.Equal(v, phonemes.Unscramble(phonemes.Scramble(v)))
	}
	assert.Equal(uint32(0xffff0817), phonemes.Unscramble(phonemes.Scramble(0xffff0817)))

	test_cases := []struct {
		P string
		Q string
	}{
		{"wispem-wantex", "tastus-marpem"},
		{"dostec-risfen", "fipfes-fipfes"},
		{"winhes-doplus", "mirdyt-dasfeb"},
	}
	for _, c := range test_cases {
		p, is_ok := phonemes.PhonemeQToInt(c.P)
		assert.True(is_ok)
		q, is_ok := phonemes.PhonemeQToInt(c.Q)
		assert.True(is_ok)
		assert.Equal(uint32(p), phonemes.Scramble(uint32(q)))
		assert.Equal(uint32(q), phonemes.Unscramble(uint32(p)))
	}
}
