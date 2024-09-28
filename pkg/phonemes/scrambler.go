package phonemes

import (
	"github.com/spaolacci/murmur3"
)

func Prf(j int, r uint16) uint32 {
	if j > 3 {
		panic(j)
	}
	seed := []uint32{
		0xb76d_5eed,
		0xee28_1300,
		0x85bc_ae01,
		0x4b38_7af7,
	}[j]
	return murmur3.Sum32WithSeed([]byte{byte(r % 0x100), byte(r >> 8)}, seed)
}

// Used to produce a `@p` ship name
func Scramble(m uint32) uint32 {
	if m < 0x10000 {
		// Not a planet-- no scrambling needed
		return m
	}
	return Fein(m-0x10000) + 0x10000
}

// Implementation of `fein`, the scrambler function
func Fein(m uint32) uint32 {
	l := m % 0xffff
	r := m / 0xffff

	for j := 0; j < 4; j++ {
		// Temporarily use a uint64 to avoid overflows if PRF is close to 0xffff_ffff (we mod `%`
		// it back down again below)
		f := uint64(Prf(j, uint16(r))) + uint64(l)

		l = r
		if j%2 == 0 {
			r = uint32(f % 0xffff)
		} else {
			r = uint32(f % 0x10000)
		}
	}
	if r == 0xffff {
		return r*0xffff + l
	}
	return l*0xffff + r
}

// Decode a `@p` to get its actual Azimuth point number
func Unscramble(m uint32) uint32 {
	if m < 0x10000 {
		// Not a planet-- no scrambling needed
		return m
	}
	return Fynd(m-0x10000) + 0x10000
}

// Implementation of `fynd`, the unscrambler
func Fynd(m uint32) uint32 {
	l := m % 0xffff
	r := m / 0xffff
	if r != 0xffff {
		l, r = r, l
	}

	for j := (4); j > 0; j-- {
		f := Prf(j-1, uint16(l))
		var tmp uint32
		if j%2 != 0 {
			tmp = (r + 0xffff - (f % 0xffff)) % 0xffff
		} else {
			tmp = (r + 0x10000 - (f % 0x10000)) % 0x10000
		}
		r = l
		l = tmp
	}
	return r*0xffff + l
}
