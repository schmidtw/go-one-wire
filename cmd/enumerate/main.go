package main

import (
	"fmt"
	"os"
	"time"

	"github.com/schmidtw/go1wire"
	"github.com/schmidtw/go1wire/adapters/ds2480"
	"github.com/schmidtw/go1wire/devices/ds18x20"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "path_to_ds2480 (usually /dev/ttyUSB0)")
		return
	}

	adapter := &ds2480.Ds2480{Name: os.Args[1],
		Speed: "standard",
		PDSRC: 1370,
		//PPD:   time.Microsecond * 512,
		//SPUD:  time.Microsecond * 1,
		W1LT: time.Microsecond * 10,
		W0RT: time.Microsecond * 8,
		LOAD: 0,
		Baud: 9600,
		SPU:  false,
		IRP:  false,
	}
	err := adapter.Init()
	if nil != err {
		fmt.Printf("Init Error: %v\n", err)
		return
	}
	adapter.Open()
	defer adapter.Close()
	adapter.Detect()

	tempSensors := []*ds18x20.Ds18x20{}
	n, _ := go1wire.NewNetwork(adapter, nil)
	list, err := n.Search()
	if nil != err {
		fmt.Printf("Err: %s\n", err)
	} else {
		fmt.Printf("== Found =========================\n")
		for i := 0; i < len(list); i++ {
			t, _ := ds18x20.New(adapter, list[i])
			if nil != t {
				fmt.Println(t.String())
				tempSensors = append(tempSensors, t)
			} else {
				fmt.Printf("%s - Unknown\n", list[i].String())
			}
		}
	}

	ds18x20.ConvertAll(adapter)

	fmt.Printf("==================================\n")
	for _, t := range tempSensors {
		temp, _ := t.LastTemp()
		fmt.Printf("%s - Temp: %f (C) %f (F)\n", t.String(), temp, temp*9/5+32.0)
	}
}
