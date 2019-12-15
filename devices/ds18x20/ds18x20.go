package ds18x20

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/schmidtw/go1wire"
)

const (
	CMD_CONVERT_T         = 0x44
	CMD_READ_SCRATCHPAD   = 0xbe
	CMD_WRITE_SCRATCHPAD  = 0x4e
	CMD_COPY_SCRATCHPAD   = 0x48
	CMD_READ_POWER_SUPPLY = 0xb4

	FAMILY_DS18S20 = 0x10
	FAMILY_DS18B20 = 0x28
)

type Ds18x20 struct {
	address go1wire.Address
	net     go1wire.Adapter
}

func New(adapter go1wire.Adapter, addr go1wire.Address) (*Ds18x20, error) {
	d := &Ds18x20{
		net:     adapter,
		address: addr,
	}

	return d, nil
}

func (d *Ds18x20) ConvertAll() {
	d.net.Reset()
	d.net.TxRx([]byte{0xcc, 0x44}, nil)
	time.Sleep(time.Millisecond * 750)
	//d.net.Reset()

	return
}

func (d *Ds18x20) readScratchPad() ([]byte, error) {
	addr := append([]byte{0x55}, d.address.Bytes()...)
	cmd := []byte{CMD_READ_SCRATCHPAD,
		0xff, 0xff, 0xff,
		0xff, 0xff, 0xff,
		0xff, 0xff, 0xff}
	tx := append(addr, cmd...)
	rx := make([]byte, len(tx))

	d.net.Reset()
	fmt.Printf("tx:\n%s\n", hex.Dump(tx))
	if err := d.net.TxRx(tx, rx); nil != err {
		return nil, err
	}
	fmt.Printf("rx:\n%s\n", hex.Dump(rx))

	data := rx[len(rx)-9:]
	if data[8] != go1wire.Crc8(data[:8]) {
		return nil, fmt.Errorf("CRC didn't match")
	}

	return data, nil
}

// Returns the last measured temperature in degrees C
func (d *Ds18x20) LastTemp() (float64, error) {
	buf, err := d.readScratchPad()
	if nil != err {
		return 0.0, err
	}

	lsb := 0.5
	if FAMILY_DS18B20 == d.address.Family() {
		switch 0x03 & (buf[4] >> 5) {
		case 1:
			lsb = 0.25
		case 2:
			lsb = 0.125
		case 3:
			lsb = 0.0625
		}
	}
	rv := float64(int(int8(buf[1]))<<8+int(buf[0])) * lsb

	return rv, nil
}
