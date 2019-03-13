package main

import (
	"fmt"
	"github.com/py60800/tuya"
	"time"
)

var conf2 = `[
    {"gwId":"1582850884f3eb30128e",
     "key":"XXXXXXXXXXX",
     "type":"Switch",
     "name":"sw1" },
    {"gwId":"86273325cc50e3c8fe2d",
     "key":"XXXXXXXXXXX",
     "type":"Switch",
     "name":"sw2" }
     ]`
var conf1 = `[
    {"gwId":"1582850884f3eb30128e",
     "key":"XXXXXXXXXXX",
     "type":"Switch",
     "name":"sw1" }
     ]`

func main() {
	dm := tuya.NewDeviceManager(conf1)
	b1, _ := dm.GetDevice("sw1")
	sw1 := b1.(tuya.Switch)
	b, err := sw1.SetW(true, 10*time.Second)
	if err != nil {
		fmt.Println("Set error:", err)
	} else {
		fmt.Println("Set OK", b)
	}
	time.Sleep(2 * time.Second)
	b, err = sw1.SetW(false, 10*time.Second)
	if err != nil {
		fmt.Println("Set error:", err)
	} else {
		fmt.Println("Set OK", b)
	}
}
