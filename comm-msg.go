// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net"
   "sync/atomic"
   "time"
)

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

type response struct {
   err  error
   data []byte
}
type query struct {
   cmd  int
   data []byte
}

const (
   cnxClean = iota
   cnxDirty
)

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

func (a *Appliance) tcpConnManager(c chan query) {
   var cnx net.Conn
   var err error
   var addr string
   rcvFailed := make(chan int)
   for {
      // Wait for some order
      var q query
   loop:
      for {
         select {
         case <-rcvFailed:
            cnx.Close()
            cnx = nil
         case q = <-c:
            break loop
         }
      }
   sendloop:
      for trial := 0; trial < 3; trial++ {
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
               break
            } else {
               time.Sleep(3 * time.Second)
               cnx = nil
            }
         }
         if cnx == nil {
            log.Println("Connection to <%v> failed ", err)
            break sendloop
         }
         err = tcpSend(cnx, q.cmd, q.data)
         if err != nil {
            cnx.Close()
            <-rcvFailed
            cnx = nil
         } else {
            //Success!
            break sendloop
         }
      }
   }
}

// sends a message over TCP

func tcpSend(cnx net.Conn, cmd int, data []byte) error {
   // simple appliances are expected to reply quickly
   now := time.Now()
   cnx.SetWriteDeadline(now.Add(5 * time.Second))
   //   cnx.SetReadDeadline(now.Add(5 * time.Second))

   // tuya appliances cannot handle multiple read!!
   // => fill a buffer and write it
   b := make([]byte, 0, 16)
   b = append(b, ui2b(uint(0x55aa), 4)...)
   b = append(b, ui2b(uint(cmd), 8)...)
   b = append(b, ui2b(uint(len(data)+8), 4)...)
   b = append(b, data...)
   b = append(b, ui2b(uint(0xaa55), 8)...)
   if _, err := cnx.Write(b); err != nil {
      return err
   }
   return nil
}

// create base messages
func (d *Appliance) makeBaseMsg() map[string]interface{} {
   d.mutex.RLock()
   defer d.mutex.RUnlock()
   m := make(map[string]interface{})
   m["devId"] = d.GwId
   m["uid"] = d.GwId
   m["t"] = time.Now().Unix()
   return m
}
func (d *Appliance) makeStatusMsg() map[string]interface{} {
   d.mutex.RLock()
   defer d.mutex.RUnlock()
   return map[string]interface{}{"gwId": d.GwId, "devId": d.GwId}
}

// -------------------------------
func (d *Appliance) SendEncryptedCommand(cmd int, jdata interface{}) error {
   d.mutex.RLock()
   data, er1 := json.Marshal(jdata)
   if er1 != nil {
      d.mutex.RUnlock()
      return fmt.Errorf("Json Marshal (%v)", er1)
   }
   cipherText, er2 := aesEncrypt(data, d.key)
   if er2 != nil {
      d.mutex.RUnlock()
      return fmt.Errorf("Encrypt error(%v)", er2)
   }
   sig := md5Sign(cipherText, d.key, d.Version)
   b := make([]byte, 0, len(sig)+len(d.Version)+len(cipherText))
   b = append(b, []byte(d.Version)...)
   b = append(b, sig...)
   b = append(b, cipherText...)
   d.mutex.RUnlock()

   d.tcpChan <- query{cmd, b}
   return nil
}

func (d *Appliance) processResponse(code int, b []byte) {
   var i int
   for i = 0; i < len(b) && b[i] == byte(0); i++ {
   }
   b = b[i:]
   if len(b) == 0 {
      // can be an Ack
      d.device.ProcessResponse(code, b)
      return
   } // empty
   var data []byte
   if b[0] == byte('{') {
      //  Message in clear text
      data = b
   } else {
      encrypted := false
      if len(b) > (len(d.Version) + 16) {
         // Check if message starts with version number
         encrypted = true
         for i, vb := range d.Version {
            encrypted = encrypted && b[i] == byte(vb)
         }
      }
      if !encrypted {
         // can be an error message
         log.Println("Resp:", code, string(b))
         return
      }
      var err error
      data, err = aesDecrypt(b[len(d.Version)+16:], d.key) // ignore signature
      if err != nil {
         log.Println("Decrypt error")
         return
      }
   }
   d.device.ProcessResponse(code, data)
}

// Send message unencrypted
func (d *Appliance) SendCommand(cmd int, jdata interface{}) error {
   data, er1 := json.Marshal(jdata)
   if er1 != nil {
      return fmt.Errorf("Json Marshal(%v)", er1)
   }
   d.tcpChan <- query{cmd, data}
   return nil
}

// Sync mehton
var chanRef = int64(0)

func (a *Appliance) getSyncChannel() (int64, chan int) {
   a.cnxMutex.Lock()
   c := make(chan int, 1)
   k := atomic.AddInt64(&chanRef, 1)
   a.syncChannel[k] = c
   a.cnxMutex.Unlock()
   return k, c
}
func (a *Appliance) deleteSyncChannel(k int64) {
   a.cnxMutex.Lock()
   close(a.syncChannel[k])
   delete(a.syncChannel, k)
   a.cnxMutex.Unlock()
}
func (a *Appliance) broadcastSyncChannel(code int) {
   a.cnxMutex.Lock()
   for _, c := range a.syncChannel {
      c <- code
   }
   a.cnxMutex.Unlock()
}
