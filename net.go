// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

// Tuya low level communication library

package tuya

import (
   "io"
   "log"
   "net"
   "time"
)

// -------------------------------
type query struct {
   cmd  int
   data []byte
}

//
// helpers
func ui2b(v uint, n int) []byte {
   b := make([]byte, n)
   for v > 0 {
      n = n - 1
      b[n] = byte(v & 0xff)
      v = v >> 8
   }
   return b
}

func uiRd(b []byte) uint {
   r := uint(0)
   for i := 0; i < 4; i++ {
      r = r<<8 | (uint(b[i]) & 0xff)
   }
   return r
}

// -------------------------------
// receiver coroutine run continuously until communication error
// ccmd chan is used to signal a crash
func (a *Appliance) tcpReceiver(cnx net.Conn, ccmd chan int) {
   var bye = func() {
      ccmd <- 0
   }
   defer bye()
   var err error
   for {
      header := make([]byte, 4*4)
      if _, err = io.ReadFull(cnx, header); err != nil {
         log.Println("Rcv error:", err)
         return
      }
      if int(uiRd(header)) != 0x55aa {
         log.Println("Sync error:", err)
         return
      }
      code := int(uiRd(header[8:]))
      sz := int(uiRd(header[12:]))
      if sz > 10000 {
         log.Println("Dubious big response")
         return
      }
      response := make([]byte, sz)
      if _, err = io.ReadFull(cnx, response); err != nil {
         log.Println("Read failed", err)
         return
      }
      a.processResponse(code, response[:sz-8])
   }
}

// -------------------------------
// Create a connection,
//  waits for the IP broadcast by the appliance(first time)
//  connects
// retries 3 times in case of failer
func (a *Appliance) getConnection(rcvFailed chan int) (net.Conn, error) {
   var cnx net.Conn = nil
   var err error
   var addr string
   for ctrial := 0; cnx == nil && ctrial < 3; ctrial++ {
      // get IP (that can change)
      a.cnxMutex.Lock()
      for len(a.Ip) == 0 {
         a.cnxSignal.Wait()
      }
      addr = a.Ip + ":6668"
      a.cnxMutex.Unlock()
      cnx, err = net.DialTimeout("tcp", addr, time.Second*5)
      if err == nil {
         go a.tcpReceiver(cnx, rcvFailed)
         return cnx, nil
         break
      } else {
         time.Sleep(3 * time.Second)
         cnx = nil
      }
   }
   log.Printf("Connection to <%v> failed %v\n", addr, err)
   return nil, err
}


// -------------------------------
// Master TCP connection coroutine
// Receives messages to be sent from query channel
func (a *Appliance) tcpConnManager(c chan query) {
   var cnx net.Conn
   var err error
   rcvFailed := make(chan int)

   cnx, _ = a.getConnection(rcvFailed)
   q := query{CodeMsgStatus, a.initialStatusMsg()}
   for {
      // Status message is sent the first time => send it before retrieving next cmd
   sendloop:
      for trial := 0; trial < 3; trial++ {
         if cnx == nil {
            cnx, err = a.getConnection(rcvFailed)
            if cnx == nil {
               break sendloop // Connection failed
            }
         }
         err = tcpSend(cnx, q.cmd, q.data)
         if err != nil {
            cnx.Close() // => Receiver will "crash"
            <-rcvFailed // wait for receiver crash confirm
            cnx = nil
         } else {
            //Success!
            break sendloop
         }
      }
   loop:
      // wait for something to do
      for {
         select {
         case q = <-c: // New message to be sent
            break loop
         case <-rcvFailed:
            // Read error : Receive thread aborted => Need reconnection
            // No hurry => wait for another message or ping
            cnx.Close()
            cnx = nil 
         case <-time.After(15 * time.Second):
            // Send a Ping message when nothing occurs
            if cnx != nil {
               q = query{CodeMsgPing, []byte{}}
            } else {
               // Broken connection => restart with a status message
               q = query{CodeMsgStatus, a.initialStatusMsg()}
            }
            break loop
         }
      }
   }
}

// -------------------------------
// sends a message over TCP
func tcpSend(cnx net.Conn, cmd int, data []byte) error {
   // simple appliances are expected to reply quickly
   now := time.Now()
   cnx.SetWriteDeadline(now.Add(10 * time.Second))

   // tuya appliances cannot handle multiple read!!
   // => fill a buffer and write it
   b := make([]byte, 0, 16)
   b = append(b, ui2b(uint(0x55aa), 4)...)
   b = append(b, ui2b(uint(cmd), 8)...)
   b = append(b, ui2b(uint(len(data)+8), 4)...)
   b = append(b, data...)
   b = append(b, ui2b(uint(0xaa55), 8)...)
   if _, err := cnx.Write(b); err != nil {
      log.Println(err)
      return err
   }
   return nil
}
