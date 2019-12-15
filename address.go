// Package go1wire provides a small framework for supporting different 1-wire
// based devices through different types of adapters.

package go1wire

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// An Address represents the unique 64-bit ROM code of a 1-wire device.
//
// Note: This is work in progress as it's unclear which form the device
//       needs and uses most of the time...
//
//	 MSB       LSB MSB                  LSB MSB               LSB
//	+-------------+------------------------+---------------------+
//	|  8-bit crc  |  48-bit serial number  |  8-bit family code  |
//	+-------------+------------------------+---------------------+
type Address uint64

// Provides the 'family' portion of the address
func (a Address) Family() byte {
	return byte(0xff & (a >> 56))
}

// Provides the canonical string representation of the address
func (a Address) String() string {
	sn := uint64(0x00ffffffffffff00&a) >> 8
	crc := uint64(a & 0xff)
	return fmt.Sprintf("%02x.%012x.%02x", a.Family(), sn, crc)
}

// Provides an array of bytes that represents the address.
func (a Address) Bytes() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(a))
	return buf
}

// Creates an Address from the canonical string version.
//
// Format (all characters are expected to be in hex):
//     crc.serial.family
//
// If the crc value is "--" then it will be calculated and not verified.
func ParseAddress(s string) (Address, error) {

	var family uint8
	var sn uint64
	var crcStr string
	cnt, err := fmt.Sscanf(s, "%x.%x.%s", &family, &sn, &crcStr)

	if (nil != err) || (3 != cnt) || (sn != (0xffffffffffff & sn)) {
		return 0, errors.New("onewire: invalid address " + s)
	}
	a := sn<<8 | (uint64(family) << 56)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, sn<<8|(uint64(family)<<56))

	crc := RevCrc8(buf[1:])

	if "--" != crcStr {
		var c uint8
		cnt, err = fmt.Sscanf(crcStr, "%x", &c)
		if c != crc {
			return 0, errors.New("onewire: invalid crc " + s)
		}
	}

	a |= 0xff & uint64(crc)

	return Address(a), nil
}

// Creates an Address from the canonical byte version.
// crc[0], sn[6-0], family[0]
//
func AddressFromBytes(buf []byte) (Address, error) {
	if 8 != len(buf) {
		return Address(0), errors.New("onewire: invalid buffer length")
	}
	crc := Crc8(buf[:7])
	if buf[7] != crc {
		return 0, errors.New("onewire: invalid crc")
	}

	return Address(binary.BigEndian.Uint64(buf)), nil
}

func AddressFromUint64(a uint64) (Address, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, a)

	return AddressFromBytes(buf)
}

func AddressFromSearch(a uint64) (Address, error) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, a)

	return AddressFromBytes(buf)
}
