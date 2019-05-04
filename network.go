package onewire

import (
	//"encoding/hex"
	//"fmt"
	"sync"
)

type Adapter interface {
	Detect() (bool, error)
	Reset() (string, byte, error)
	Search(tree uint64, last int) (uint64, int, error)
	TxRx(tx, rx []byte) error
}

type Device interface {
	Address() Address
	Self() *interface{}
}

type Network struct {
	AdapterPresent bool
	AdapterType    string
	Functional     map[Address]*Device
	NonFunctional  map[Address]*Device

	adapter Adapter
	devices *[]Device
	mutex   sync.Mutex
}

func NewOneWireNetwork(a Adapter, d ...*Device) (*Network, error) {
	rv := &Network{
		adapter: a,
	}

	return rv, nil
}

func (n *Network) Search() ([]Address, error) {

	var last uint64
	var list []Address

	for i := 0; i < 64; {
		var err error
		var next uint64

		next, i, err = n.adapter.Search(last, i)
		if nil != err {
			return nil, err
		}
		last = next

		//fmt.Printf("next: 0x%0x\n", next)
		a, err := AddressFromUint64(next)
		if nil == err {
			list = append(list, *a)
		}
	}

	return list, nil
}

func (n *Network) Validate() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.AdapterPresent = false
	n.AdapterType = ""
	n.Functional = make(map[Address]*Device)
	n.NonFunctional = make(map[Address]*Device)

	present, err := n.adapter.Detect()
	if nil != err {
		return err
	}
	n.AdapterPresent = present
	version, _, err := n.adapter.Reset()
	if nil != err {
		return err
	}
	n.AdapterType = version

	return nil
}

func (n *Network) Init() error {
	return nil
}
