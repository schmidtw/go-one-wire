package go1wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAddress(t *testing.T) {
	assert := assert.New(t)

	a, err := ParseAddress("10.450736030800.e7")
	if assert.NoError(err) {
		assert.Equal(byte(0x10), a.Family())
		expected := []byte{0x10, 0x45, 0x07, 0x36, 0x03, 0x08, 0x00, 0xe7}
		assert.Equal("10.450736030800.e7", a.String())
		assert.Equal(expected, a.Bytes())
	}

	a, err = ParseAddress("1.4507360308.--")
	if assert.NoError(err) {
		assert.Equal("01.004507360308.18", a.String())
	}

	// CRC does not match
	a, err = ParseAddress("10.450736030800.09")
	assert.Error(err)
	assert.Equal(Address(0), a)

	// Invalid CRC character
	a, err = ParseAddress("rr.000803360745.10")
	assert.Error(err)
	assert.Equal(Address(0), a)

	// Invalid sn character
	a, err = ParseAddress("--.rrr.10")
	assert.Error(err)
	assert.Equal(Address(0), a)

	// Invalid family character
	a, err = ParseAddress("--.1.rr")
	assert.Error(err)
	assert.Equal(Address(0), a)

	// Invalid format
	a, err = ParseAddress("--.10")
	assert.Error(err)
	assert.Equal(Address(0), a)

	a, err = ParseAddress("-.803360745.1")
	assert.Error(err)
	assert.Equal(Address(0), a)
}

func TestAddressFromBytes(t *testing.T) {
	assert := assert.New(t)

	a, err := AddressFromBytes([]byte{0x10, 0x45, 0x07, 0x36, 0x03, 0x08, 0x00, 0xe7})
	assert.Equal("10.450736030800.e7", a.String())
	if assert.NoError(err) {
		assert.Equal("10.450736030800.e7", a.String())
	}

	// Invalid number of bytes
	a, err = AddressFromBytes([]byte{0x00})
	assert.Error(err)
	assert.Equal(Address(0), a)
}

func TestAddressFromUint64(t *testing.T) {
	assert := assert.New(t)

	a, err := AddressFromUint64(0x10450736030800e7)
	//a, err := AddressFromUint64(0xe700080336074510)
	if assert.NoError(err) {
		assert.Equal("10.450736030800.e7", a.String())
	}
}
