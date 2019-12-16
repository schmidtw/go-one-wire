package go1wire

import ()

type Adapter interface {
	Detect() (bool, error)
	Reset() (string, byte, error)
	Search() ([]Address, error)
	TxRx(tx, rx []byte) error
}
