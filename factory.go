// Copyright 2019 py60800. 
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import "log"

type Device interface {
   Type() string
   Attach(*Appliance)
}

func makeDevice(typ string) (Device, bool) {
   switch typ {
   case "Switch":
      return new(ISwitch), true
   default:
      log.Println("Unknown device:", typ)
   }
   return nil, false
}
