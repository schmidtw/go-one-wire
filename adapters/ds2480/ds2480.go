package ds2480

import (
	//"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	serial "github.com/schmidtw/go232"
)

const (
	MODE_DATA       = 0xe1
	MODE_COMMAND    = 0xe3
	MODE_STOP_PULSE = 0xf1

	CMD_RESET            = 0xc1
	CMD_PULLUP           = 0x3b
	CMD_PULLUP_ARM       = 0xef
	CMD_PULLUP_DISARM    = 0xed
	CMD_PULSE            = 0xed
	CMD_PULSE_TERMINATE  = 0xf1
	CMD_CONFIG           = 0x01
	CMD_WRITE_BIT        = 0x81
	CMD_SEARCH           = 0xf0
	CMD_SEARCH_ACCEL_ON  = 0xb1
	CMD_SEARCH_ACCEL_OFF = 0xa1

	CFG_READ  = 0
	CFG_PDSRC = 1
	CFG_PPD   = 2
	CFG_SPUD  = 3
	CFG_W1LT  = 4
	CFG_W0RT  = 5
	CFG_LOAD  = 6
	CFG_BAUD  = 7

	CHIP_MODE__COMMAND = iota
	CHIP_MODE__DATA    = iota
)

var ErrInvalidResponse = errors.New("invalid response")
var ErrInvalidState = errors.New("file already open")

var speedMap = map[string]byte{
	"":          0, // Make the 0 value the default value
	"standard":  0,
	"flexible":  1,
	"overdrive": 2,
}

// Pull Down Slew Rate Control
// The units for these are mVolts/uSecond - defined in DS2480B.pdf page 13
var pdsrcMap = map[int]byte{
	0:     0, // Make the 0 value the default value
	15000: 0, // Default Std, Flex, Overdrive
	2200:  1,
	1650:  2,
	1370:  3,
	1100:  4,
	830:   5,
	700:   6,
	550:   7,
}

// Programming Pulse Duration - defined in DS2480B.pdf page 13
var ppdMap = map[time.Duration]byte{
	time.Hour * 0:           4, // Make the 0 value the default value
	time.Microsecond * 32:   0,
	time.Microsecond * 64:   1,
	time.Microsecond * 128:  2,
	time.Microsecond * 256:  3,
	time.Microsecond * 512:  4, // Default Std, Flex, Overdrive
	time.Microsecond * 1024: 5,
	time.Microsecond * 2048: 6,
	time.Hour * 1:           7, // Forever
}

// Strong Pull Up Duration - defined in DS2480B.pdf page 13
var spudMap = map[time.Duration]byte{
	time.Hour * 0:            4, // Make the 0 value the default value
	time.Microsecond * 16400: 0,
	time.Microsecond * 65500: 1,
	time.Millisecond * 131:   2,
	time.Millisecond * 262:   3,
	time.Millisecond * 524:   4, // Default Std, Flex, Overdrive
	time.Millisecond * 1048:  5,
	time.Hour * 1:            7, // Forever
}

// Write 1 Low Time  - defined in DS2480B.pdf page 13
var w1ltMap = map[time.Duration]byte{
	time.Hour * 0:         0, // Make the 0 value the default value
	time.Microsecond * 8:  0, // Default Std, Flex
	time.Microsecond * 9:  1, // Default Overdrive
	time.Microsecond * 10: 2,
	time.Microsecond * 11: 3,
	time.Microsecond * 12: 4,
	time.Microsecond * 13: 5,
	time.Microsecond * 14: 6,
	time.Microsecond * 15: 7,
}

// Write 0 Recovery Time / Data Sample Offset - defined in DS2480B.pdf page 13
var w0rtMap = map[time.Duration]byte{
	time.Hour * 0:         0, // Make the 0 value the default value
	time.Microsecond * 3:  0,
	time.Microsecond * 4:  1,
	time.Microsecond * 5:  2,
	time.Microsecond * 6:  3,
	time.Microsecond * 7:  4,
	time.Microsecond * 8:  5,
	time.Microsecond * 9:  6,
	time.Microsecond * 10: 7,
}

// LOAD in uA - defined in DS2480B.pdf page 13
var loadMap = map[int]byte{
	0:    0, // Make the 0 value the default value
	1800: 0,
	2100: 1,
	2400: 2,
	2700: 3,
	3000: 4,
	3300: 5,
	3600: 6,
	3900: 7,
}

// Desired BAUD rate to run the 1-wire system at
var baudMap = map[int]byte{
	0:      0, // Make the 0 value the default value
	9600:   0, // Default Std, Flex, Overdrive
	19200:  1,
	57600:  2,
	115200: 3,
	// The rest are duplicates...
}

type Ds2480 struct {
	// Configuration Section
	Name  string        // The filename for the serial interface
	Speed string        // Speed: standard, flexible, overdrive
	PDSRC int           // Pull Down Slew Rate Control (Volts/uSecond)
	PPD   time.Duration // Programming Pulse Duration
	SPUD  time.Duration // Strong Pull Up Duration
	W1LT  time.Duration // Write 1 Low Time
	W0RT  time.Duration // Write 0 Recovery Time / Data Sample Offset
	LOAD  int           // LOAD on the Bus (uA)
	Baud  int           // Desired BAUD rate to run the 1-wire system at
	SPU   bool          // Strong Pull Up (if true)
	IRP   bool          // Inverse RXD Polarity - Set to inverse the RXD polarity

	// Sanitized Configuration Values
	speed byte
	pdsrc byte
	ppd   byte
	spud  byte
	w1lt  byte
	w0rt  byte
	load  byte
	baud  byte
	spu   byte
	irp   bool

	// Runtime State about the chip
	serial    *serial.Serial
	chipLevel byte
	chipBaud  byte
	chipMode  byte
	chipSpeed byte
}

func checkStringConfig(name, cfg string, m map[string]byte) error {
	if _, ok := m[cfg]; !ok {
		var comma, list string
		for k := range m {
			list += comma + k
			comma = ", "
		}
		return fmt.Errorf("%s: %s is invalid. [ %s ]", name, cfg, list)
	}
	return nil
}

func checkIntConfig(name string, cfg int, m map[int]byte) error {
	if _, ok := m[cfg]; !ok {
		var comma, list string
		for k := range m {
			list += comma + fmt.Sprintf("%d", k)
			comma = ", "
		}
		return fmt.Errorf("%s: %d is invalid. [ %s ]", name, cfg, list)
	}
	return nil
}

func checkDurationConfig(name string, cfg time.Duration, m map[time.Duration]byte) error {
	if _, ok := m[cfg]; !ok {
		var comma, list string
		for k := range m {
			list += comma + fmt.Sprintf("%s", k.String())
			comma = ", "
		}
		return fmt.Errorf("%s: %s is invalid. [ %s ]", name, cfg.String(), list)
	}
	return nil
}

func (d *Ds2480) Init() error {
	if err := checkStringConfig("Speed", d.Speed, speedMap); nil != err {
		return err
	}
	d.speed = speedMap[d.Speed]

	if err := checkIntConfig("PDSRC", d.PDSRC, pdsrcMap); nil != err {
		return err
	}
	d.pdsrc = pdsrcMap[d.PDSRC]

	if err := checkDurationConfig("PPD", d.PPD, ppdMap); nil != err {
		return err
	}
	d.ppd = ppdMap[d.PPD]

	if err := checkDurationConfig("SPUD", d.SPUD, spudMap); nil != err {
		return err
	}
	d.spud = ppdMap[d.SPUD]

	if err := checkDurationConfig("W1LT", d.W1LT, w1ltMap); nil != err {
		return err
	}
	d.w1lt = w1ltMap[d.W1LT]

	if err := checkDurationConfig("W0RT", d.W0RT, w0rtMap); nil != err {
		return err
	}
	d.w0rt = w0rtMap[d.W0RT]

	if err := checkIntConfig("LOAD", d.LOAD, loadMap); nil != err {
		return err
	}
	d.load = loadMap[d.LOAD]

	if err := checkIntConfig("Baud", d.Baud, baudMap); nil != err {
		return err
	}
	d.baud = baudMap[d.Baud]

	if true == d.SPU {
		d.spu = 1
	}
	d.irp = d.IRP

	return nil
}

func (d *Ds2480) Open() error {
	if nil != d.serial {
		return ErrInvalidState
	}
	d.serial = &serial.Serial{Name: d.Name}
	return d.serial.Open()
}

func (d *Ds2480) Close() error {
	if nil != d.serial {
		return d.serial.Close()
	}
	return nil
}

func (d *Ds2480) Detect() (bool, error) {
	d.chipMode = CHIP_MODE__COMMAND
	d.chipBaud = baudMap[9600]
	d.chipSpeed = speedMap["flexible"]

	d.serial.Baud = 9600
	d.serial.Config = "8N1"
	if err := d.serial.UpdateCfg(); nil != err {
		return false, err
	}
	if err := d.serial.SendBreak(); nil != err {
		return false, err
	}

	time.Sleep(time.Millisecond * 2)

	if err := d.serial.Flush(); nil != err {
		return false, err
	}

	reset := make([]byte, 1)
	reset[0] = CMD_RESET | (d.speed << 2)
	if n, err := d.serial.Write(reset); n != 1 || nil != err {
		return false, err
	}

	time.Sleep(time.Millisecond * 2)
	send := make([]byte, 5)
	send[0] = CMD_CONFIG | (CFG_PDSRC << 4) | (d.pdsrc << 1)
	send[1] = CMD_CONFIG | (CFG_W1LT << 4) | (d.w1lt << 1)
	send[2] = CMD_CONFIG | (CFG_W0RT << 4) | (d.w0rt << 1)
	send[3] = CMD_CONFIG | (CFG_READ << 4) | (CFG_BAUD << 1)
	send[4] = CMD_WRITE_BIT | (1 << 4) | (d.speed << 2) | (d.spu << 1)

	//fmt.Printf("Sending:\n%s", hex.Dump(send))
	if n, err := d.serial.Write(send); len(send) != n || nil != err {
		return false, err
	}

	got := make([]byte, 5)
	if _, err := io.ReadFull(d.serial, got); nil != err {
		return false, err
	}

	//fmt.Printf("Got:\n%s", hex.Dump(got))

	if got[3] == d.baud && (got[4]&0xFC) == (send[4]&0xfc) {
		return true, nil
	}

	return false, nil
}

func (d *Ds2480) Reset() (version string, result byte, err error) {

	tx := []byte{CMD_RESET | (d.speed << 2)}
	rx := make([]byte, 1)

	err = d.txrx(CHIP_MODE__COMMAND, tx, rx)
	if nil != err {
		return "", 0, err
	}

	if 0xc != 0xc&rx[0] {
		d.Detect()
		return "", 0, ErrInvalidResponse
	}

	switch (0x1c & rx[0]) >> 2 {
	case 2:
		version = "ds2480"
	case 3:
		version = "ds2480b"
	default:
		d.Detect()
		return "", 0, ErrInvalidResponse
	}

	result = 0x3 & rx[0]

	return version, result, nil
}

/*
func (d *Ds2480) ReadBit(b []byte) error {
	for i := 0; i < len(b); i++ {
		tmp, err := d.writeBit(1)
		if nil != err {
			return err
		}
		b[i] = tmp
	}
	return nil
}

func (d *Ds2480) WriteBit(b []byte) error {
	for i := 0; i < len(b); i++ {
		_, err := d.writeBit(b[i])
		if nil != err {
			return err
		}
	}
	return nil
}

func (d *Ds2480) Read(b []byte) error {
	buf := make([]byte, len(b))
	for i := 0; i < len(buf); i++ {
		buf[i] = 0xff
	}

	got, err := d.txrx(CHIP_MODE__DATA, buf, len(buf))
	if nil != err {
		return err
	}

	for i := 0; i < len(got); i++ {
		b[i] = got[i]
	}

	return nil
}

func (d *Ds2480) Write(b []byte) error {
	got, err := d.txrx(CHIP_MODE__DATA, b, len(b))
	if nil != err {
		return err
	}
	for i := 0; i < len(got); i++ {
		if b[i] != got[i] {
			return ErrInvalidResponse
		}
	}

	return nil
}
*/

func searchToBytes(tree uint64, conflict int) []byte {
	data := make([]byte, 16)

	for i := uint(0); i < uint(conflict); i++ {
		idx := i*2 + 1
		byte_offset := idx / 8
		bit_offset := idx - (8 * byte_offset)

		data[byte_offset] |= byte(1&(tree>>i)) << bit_offset
	}

	if conflict < 64 {
		byte_offset := (conflict*2 + 1) / 8
		bit_offset := (conflict*2 + 1) - (8 * byte_offset)
		data[byte_offset] |= 1 << uint(bit_offset)
	}
	return data
}

func searchFromBytes(data []byte) (out uint64, conflict int) {
	//fmt.Printf("search data:\n%s", hex.Dump(data))
	conflict = 64
	for i := uint(0); i < 64; i++ {
		idx := i * 2
		byte_offset := idx / 8
		bit_offset := idx - (8 * byte_offset)

		rom_bit := 1 & uint64(data[byte_offset]>>(bit_offset+1))
		conflict_bit := 1 & uint64(data[byte_offset]>>bit_offset)

		out |= rom_bit << i

		if 0 != conflict_bit && 0 == rom_bit {
			if i < uint(conflict) {
				conflict = int(i)
			}
		}
	}

	return out, conflict
}

// Takes the uint64 that describes the tree to explore with the last byte
// indicating the MSB to explore from
//
// Note: The uint64 that is returned is reversed endian to how the addresses
//       are defined and used everywhere else.
//
// Returns the discovered tree and the index of the LSB conflict (or 64 if none)
func (d *Ds2480) Search(tree uint64, last int) (uint64, int, error) {

	if _, _, err := d.Reset(); nil != err {
		return 0, 0, err
	}

	preamble := []byte{
		CMD_SEARCH,
		MODE_COMMAND, CMD_SEARCH_ACCEL_ON | (d.speed << 2),
		MODE_DATA}

	suffix := []byte{MODE_COMMAND, CMD_SEARCH_ACCEL_OFF}

	data := searchToBytes(tree, last)

	tx := append(preamble, data...)
	tx = append(tx, suffix...)

	rx := make([]byte, 17)
	//fmt.Printf("tx:\n%s", hex.Dump(tx))
	err := d.txrx(MODE_DATA, tx, rx)
	if err != nil {
		return 0, 0, err
	}

	if CMD_SEARCH != CMD_SEARCH&rx[0] {
		d.Detect()
		return 0, 0, ErrInvalidResponse
	}

	//fmt.Printf("rx:\n%s", hex.Dump(rx))
	rx = rx[1:]
	out, last := searchFromBytes(rx)

	return out, last, nil
}

func (d *Ds2480) TxRx(tx, rx []byte) error {
	return d.txrx(MODE_DATA, tx, rx)
}

func (d *Ds2480) txrx(mode byte, tx, rx []byte) error {
	// Prepend the mode select byte
	if mode != d.chipMode {
		tmp := make([]byte, 1)
		tmp[0] = MODE_DATA
		d.chipMode = mode
		if CHIP_MODE__COMMAND == mode {
			tmp[0] = MODE_COMMAND
		}
		tx = append(tmp, tx...)
	}

	if err := d.serial.Flush(); nil != err {
		return err
	}

	//fmt.Printf("Sending:\n%s", hex.Dump(tx))
	if n, err := d.serial.Write(tx); len(tx) != n || nil != err {
		return err
	}

	if _, err := io.ReadFull(d.serial, rx); nil != err {
		d.Detect()
		return err
	}

	//fmt.Printf("Got:\n%s", hex.Dump(rx))

	return nil
}

/*
func (d *Ds2480) writeBit(bit byte) (byte, error) {
	send := []byte{CMD_WRITE_BIT | ((1 & bit) << 4) | (d.speed << 2)}

	got, err := d.txrx(CHIP_MODE__COMMAND, send, len(send))
	if nil != err {
		return 0, err
	}

	if (0xfc&send[0]) == (0xfc&got[0]) &&
		((0 == got[0]) || (3 == got[0])) {
		return 0, ErrInvalidResponse
	}

	return 1 & got[0], nil
}
*/
