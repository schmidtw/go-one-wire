package onewire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidAddresses(t *testing.T) {
	assert := assert.New(t)

	a, err := ParseAddress("e7.000803360745.10")
	if assert.NoError(err) {
		assert.Equal(byte(0x10), a.Family())
		expected := []byte{0xe7, 0x00, 0x08, 0x03, 0x36, 0x07, 0x45, 0x10}
		assert.Equal(expected, a.Bytes())
		assert.Equal("e7.000803360745.10", a.String())
	}

	b, err := AddressFromBytes([]byte{0xe7, 0x00, 0x08, 0x03, 0x36, 0x07, 0x45, 0x10})
	if assert.NoError(err) {
		assert.Equal("e7.000803360745.10", b.String())
	}

	c, err := ParseAddress("--.000803360745.10")
	if assert.NoError(err) {
		assert.Equal("e7.000803360745.10", c.String())
	}

	c, err = ParseAddress("--.803360745.10")
	if assert.NoError(err) {
		assert.Equal("e7.000803360745.10", c.String())
	}
}

func TestInvalidAddresses(t *testing.T) {
	assert := assert.New(t)

	// CRC does not match
	a, err := ParseAddress("09.000803360745.10")
	assert.Error(err)
	assert.Nil(a)

	// Invalid CRC character
	a, err = ParseAddress("rr.000803360745.10")
	assert.Error(err)
	assert.Nil(a)

	// Invalid sn character
	//a, err = ParseAddress("--.rrr.10")
	//assert.Error(err)
	//assert.Nil(a)
}
