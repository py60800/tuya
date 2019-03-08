// Copyright 2019 py60800. 
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   "encoding/json"
   "fmt"
   "io"
   "net"
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
func (a *Appliance) getCnx() (net.Conn, error){
   // Get an IP
   a.cnxMutex.Lock()
   for len(a.Ip) == 0 {
      a.cnxSignal.Wait()
   }
   addr := a.Ip+":6668"
   if a.cnxStatus == 4 {
      a.cnx.Close()
      a.cnxStatus = 0
   }
   switch a.cnxStatus {
      case 0:
         cnx, err := net.DialTimeout("tcp", addr, time.Second*5)
         if err != nil {
             a.cnxMutex.Unlock()
             return cnx,err
         }
         a.cnxStatus = 1 // dirty
         a.cnx = cnx
         return a.cnx, nil // mutex still locked
     case 2: // connexion clean
         a.cnxStatus = 1 // dirty
         return a.cnx, nil
     default:
         panic("Cnx Handling error")
   }
   panic("Cnx Error")
   return nil,nil
}
func (a *Appliance) releaseCnx(){
   if a.cnxStatus != 2 {
      a.cnx.Close()
      a.cnxStatus = 0
   }
  a.cnxMutex.Unlock()
}
func (a *Appliance) setCnxClean(){
   a.cnxStatus = 2
}
func (a *Appliance) resetCnx(){
   a.cnxStatus = 4
}
// sends a message over TCP and waits for the response
func (a *Appliance)tcpSendRcv(cmd int, data []byte) ([]byte, error) {
  // No more than one connection at a time
   cnx,err := a.getCnx();
   if err != nil {
       return []byte{},err
   }
   defer a.releaseCnx()
   
   // simple appliances are expected to reply quickly
   now := time.Now()
   cnx.SetWriteDeadline(now.Add(5 * time.Second))
   cnx.SetReadDeadline(now.Add(5 * time.Second))

   // tuya appliances cannot handle multiple read!!
   // => fill a buffer and write it
   b := make([]byte, 0, 16)
   b = append(b, ui2b(uint(0x55aa), 4)...)
   b = append(b, ui2b(uint(cmd), 8)...)
   b = append(b, ui2b(uint(len(data)+8), 4)...)
   b = append(b, data...)
   b = append(b, ui2b(uint(0xaa55), 8)...)
   if _, err := cnx.Write(b); err != nil {
      return []byte{}, err
   }

   // Message has been sent, try to get a response
   header := make([]byte, 4*4)
   if _, err := io.ReadFull(cnx, header); err != nil {
      return []byte{}, err
   }
   // who cares of the header ?
   sz := int(uiRd(header[12:]))
   if sz > 10000 { 
      return []byte{}, fmt.Errorf("Dubious big response")
   }
   response := make([]byte, sz)
   if _, err := io.ReadFull(cnx, response); err != nil {
      return []byte{}, err
   }
   //ignore crc and end marker
   a.setCnxClean()
   return response[:sz-8], nil
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
func (d *Appliance) SendEncryptedCommand(cmd int, jdata interface{}) (map[string]interface{}, error) {
   resp := make(map[string]interface{})
   d.mutex.RLock()
   data, er1 := json.Marshal(jdata)
   if er1 != nil {
      d.mutex.RUnlock()
      return resp, fmt.Errorf("Json Marshal (%v)", er1)
   }
   cipherText, er2 := aesEncrypt(data, d.key)
   if er2 != nil {
      d.mutex.RUnlock()
      return resp, fmt.Errorf("Encrypt error(%v)", er2)
   }
   sig := md5Sign(cipherText, d.key, d.Version)
   b := make([]byte, 0, len(sig)+len(d.Version)+len(cipherText))
   b = append(b, []byte(d.Version)...)
   b = append(b, sig...)
   b = append(b, cipherText...)
   d.mutex.RUnlock() 
   r, err := d.tcpSendRcv(cmd, b)
   d.mutex.RLock()
   defer d.mutex.RUnlock()
   if err != nil {
      return resp, err
   }
   resp, erf := d.processResponse(r,cmd)
   return resp, erf
}

func (d *Appliance) processResponse(b []byte,cmd int) (map[string]interface{}, error) {
   //dump("Response:",b)
   resp := make(map[string]interface{})
   // discard leading 0
   var i int
   for i = 0; i < len(b) && b[i] == byte(0); i++ {
   }
   b = b[i:]
   if len(b) == 0 {
      //empty response
      return resp, nil
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
         fmt.Println("Resp:",cmd, string(b))
         return resp, fmt.Errorf("Unexpected response(%v)", string(b))
      }
      var err error
      data, err = aesDecrypt(b[len(d.Version)+16:], d.key) // ignore signature
      if err != nil {
         return resp, err
      }
   }
   //fmt.Println("Data:",string(data))   
   erf := json.Unmarshal(data, &resp)
   //fmt.Println(erf,resp)
   return resp, erf
}
// Send message unencrypted
func (d *Appliance) SendCommand(cmd int, jdata interface{}) (map[string]interface{}, error) {
   resp := make(map[string]interface{})// default response
   data, er1 := json.Marshal(jdata)
   if er1 != nil {
      return resp, fmt.Errorf("Json Marshal(%v)", er1)
   }
   r, er2 := d.tcpSendRcv(cmd, data)
   d.mutex.RLock()
   defer d.mutex.RUnlock()
   if er2 != nil {
      return resp, er2
   }
   resp, er3 := d.processResponse(r,cmd)
   return resp, er3
}
