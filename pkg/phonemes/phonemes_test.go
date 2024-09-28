package phonemes_test

import (
	"fmt"
	"testing"

	"go-azimuth/pkg/phonemes"

	"github.com/stretchr/testify/assert"
)

func TestIntToPhoneme(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		Val     uint64
		Phoneme string
	}{
		{0, "zod"},
		{1, "nec"},
		{255, "fes"},
		{256, "marzod"},
		{258, "marbud"},
		{3762, "wispem"},
		{0x01_00_00, "doznec-dozzod"},
		{246_547_318, "wispem-wantex"},
	}
	for _, c := range cases {
		result := phonemes.IntToPhoneme(c.Val)
		assert.Equal(c.Phoneme, result, fmt.Sprintf("Expected %d to become %q, but it's %q", c.Val, c.Phoneme, result))
	}
}

func TestPhonemeToInt(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		Val     uint64
		Phoneme string
	}{
		{0, "zod"},
		{1, "nec"},
		{255, "fes"},
		{256, "marzod"},
		{258, "marbud"},
		{3762, "wispem"},
		{0x01_00_00, "doznec-dozzod"},
		{246_547_318, "wispem-wantex"},
	}
	for _, c := range cases {
		result, is_ok := phonemes.PhonemeToInt(c.Phoneme)
		assert.True(is_ok, "Not OK: %s", c.Phoneme)
		assert.Equal(c.Val, result, "Expected %q to become %d, but it's %d", c.Phoneme, c.Val, result)
	}

	for _, fail_case := range []string{"nec-dozzod", "wispemj", "wispemwantex"} {
		_, is_ok := phonemes.PhonemeToInt(fail_case)
		assert.False(is_ok)
	}
}
