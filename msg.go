// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

// Tuya high level communication library

package tuya

import (
   "encoding/json"
   "fmt"
   "log"
   "time"
)

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
func (d *Appliance) initialStatusMsg() []byte {
   m := map[string]interface{}{"gwId": d.GwId, "devId": d.GwId}
   data, _ := json.Marshal(m)
   return data
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
   if len(b) == 0 { // can be an ack
      d.device.processResponse(code, b)
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
   d.device.processResponse(code, data)
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
