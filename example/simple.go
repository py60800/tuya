package main

import (
	"fmt"
	"time"
	"github.com/py60800/tuya"
)

func main() {
	var conf = `[
    {"gwId":"1582850884f3eb30128e",
     "key":"XXXXXXXXXXX",
     "type":"Switch",
     "name":"sw1" },
    {"gwId":"86273325cc50e3c8fe2d",
     "key":"XXXXXXXXXXX",
     "type":"Switch",
     "name":"sw2" }
     ]`

	dm := tuya.NewDeviceManager(conf)
	b1, _ := dm.GetDevice("sw1")
	sw1 := b1.(tuya.Switch)
	for i := 0; ; i++ {
		toSet := (i % 2) != 0
		t := time.Now()
		b, err := sw1.SetW(toSet, 5*time.Second)
		if err != nil {
			fmt.Println("Set error:", err)
		} else {
			fmt.Printf("Set: %v/%v [%v]\n", b, toSet, time.Now().Sub(t))
		}
		time.Sleep(2 * time.Second)
	}

}
