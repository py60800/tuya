// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   "log"
   "net"
)

func udpListener(dm *DeviceManager) {
   cnx, err := net.ListenPacket("udp", ":6666")
   if err != nil {
      log.Fatal("UDP Listener failed:", err)
   }
   for {
      buffer := make([]byte, 1024)
      n, _, err := cnx.ReadFrom(buffer)
      buffer = buffer[:n]
      if err == nil && len(buffer) > 16 {
         if uiRd(buffer) == uint(0x55aa) {
            sz := uiRd(buffer[12:])
            if sz <= uint(len(buffer)-16) {
               //discard potential leading 0
               sz = sz - 8 // discard CRC and end marker
               is := 16
               for ; buffer[is] == byte(0) && is < (int(sz)+16); is++ {
               }
               //log.Print(string(buffer[is:16+sz]))
               dm.applianceUpdate(buffer[is : 16+sz])
            }
         }
      }
   }
}
