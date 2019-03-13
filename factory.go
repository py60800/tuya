// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

// To be updated if new logical devices are created
package tuya

import "log"

func makeDevice(typ string) (Device, bool) {
   switch typ {
   case "Switch":
      return new(ISwitch), true
   default:
      log.Println("Unknown device:", typ)
   }
   return nil, false
}
