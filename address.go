package onewire

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type Address uint64

// Provides the 'family' portion of the address
func (a *Address) Family() byte {
	return byte(*a & 0xff)
}

// Provides the canonical string representation of the address
func (a *Address) String() string {
	sn := uint64(0x00ffffffffffffff&*a) >> 8
	crc := byte(0xff & (*a >> 56))
	return fmt.Sprintf("%x.%012x.%x", crc, sn, a.Family())
}

// Provides an array of bytes that represents the address
func (a *Address) Bytes() []byte {
	b := make([]byte, 8)
	tmp := *a
	//for i := len(b) - 1; i > -1; i-- {
	for i := 0; i < len(b); i++ {
		b[i] = byte(0xff & tmp)
		tmp >>= 8
	}

	return b
}

// Creates an Address from the canonical string version.
//
// Format (all characters are expected to be in hex):
//     crc.serial.family
//
// If the crc value is "--" then it will be calculated and not verified.
func ParseAddress(s string) (*Address, error) {
	parts := strings.Split(s, ".")
	if 3 != len(parts) {
		return nil, errors.New("onewire: invalid address " + s)
	}
	family, err := hex.DecodeString(parts[2])
	if nil != err || 1 != len(family) {
		return nil, errors.New("onewire: invalid family " + parts[2])
	}
	if 1 == 1&len(parts[1]) {
		parts[1] = "0" + parts[1]
	}
	sn, err := hex.DecodeString(parts[1])
	if nil != err || 6 < len(sn) {
		fmt.Printf("err: %v\n", err)
		return nil, errors.New("onewire: invalid serial number " + parts[1])
	}
	var crc []byte
	if "--" != parts[0] {
		crc, err = hex.DecodeString(parts[0])
		if nil != err || 1 != len(crc) {
			return nil, errors.New("onewire: invalid crc " + parts[0])
		}
	}

	return create(family, sn, crc)
}

// Creates an Address from the canonical byte version.
// crc[0], sn[6-0], family[0]
//
func AddressFromBytes(b []byte) (*Address, error) {
	if 8 != len(b) {
		return nil, errors.New("onewire: invalid buffer length")
	}

	return create(b[7:], b[1:7], b[:1])
}

func AddressFromUint64(a uint64) (*Address, error) {
	b := make([]byte, 8)
	for i := len(b) - 1; i > -1; i-- {
		b[i] = byte(0xff & a)
		a >>= 8
	}

	return AddressFromBytes(b)
}

func create(family, sn, crc []byte) (*Address, error) {

	expectedCrc := RevCrc8(append(sn, family...))
	if nil != crc {
		//fmt.Printf("crc: %02x =?= expected: %02x\n", crc[0], expectedCrc)
		if expectedCrc != crc[0] {
			return nil, errors.New("onewire: crc check failed")
		}
	} else {
		//fmt.Printf("crc: nil =?= expected: %02x\n", expectedCrc)
		crc = []byte{expectedCrc}
	}

	var a Address

	for i := 0; i < len(sn); i++ {
		a <<= 8
		a |= Address(sn[i])
	}
	a <<= 8
	a |= Address(crc[0]) << 56

	a |= Address(family[0])

	return &a, nil
}
