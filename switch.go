// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   //   "fmt"
   "encoding/json"
   "errors"
   "sync/atomic"
   "time"
)

type Switch interface {
   Status() bool
   Set(bool) error
   SetW(bool, time.Duration) (bool, error)
   StatusW(time.Duration) (bool, error)
}

type ISwitch struct {
   d      *Appliance
   status int32
}

const statusCmd = 10
const setCmd = 7

func (s *ISwitch) Set(on bool) error {
   m := s.d.makeBaseMsg()
   m["dps"] = map[string]bool{"1": on}
   return s.d.SendEncryptedCommand(7, m)
}
func (s *ISwitch) SetW(on bool, delay time.Duration) (bool, error) {
   k, c := s.d.getSyncChannel()
   defer s.d.deleteSyncChannel(k)
   deadLine := time.Now().Add(delay)
   err := s.Set(on)
   if err != nil {
      return s.Status(), err
   }
   for {
      select {
      case <-c:
         // Ignore Code :
         if on == (int32(atomic.LoadInt32(&s.status)) != 0) {
            return on, nil
         }
      case <-time.After(deadLine.Sub(time.Now())):
         return s.Status(), errors.New("Timeout")
      }
   }
}

func (s *ISwitch) Status() bool {
   return atomic.LoadInt32(&s.status) != int32(0)
}
func (s *ISwitch) StatusW(delay time.Duration) (bool, error) {
   k, c := s.d.getSyncChannel()
   defer s.d.deleteSyncChannel(k)
   deadLine := time.Now().Add(delay)
   err := s.d.SendCommand(statusCmd, s.d.makeStatusMsg())
   if err != nil {
      return s.Status(), err
   }
   for {
      select {
      case code := <-c:
         if code == statusCmd {
            return s.Status(), nil
         }
      case <-time.After(deadLine.Sub(time.Now())):
         return s.Status(), errors.New("Timeout")
      }
   }

}
func (s *ISwitch) ProcessResponse(code int, data []byte) {
   var r map[string]interface{}
   //fmt.Println(code, string(data))
   json.Unmarshal(data, &r)
   v, ok := r["dps"]
   if ok {
      v1, ok2 := v.(map[string]interface{})
      if ok2 {
         v2, ok3 := v1["1"]
         if ok3 {
            vs, _ := v2.(bool)
            ivs := int32(0)
            if vs {
               ivs = int32(1)
            }
            atomic.StoreInt32(&s.status, ivs)
         }
      }
   }
   s.d.broadcastSyncChannel(code)
}

// Device implementation
func (s *ISwitch) Type() string {
   return "Switch"
}
func (s *ISwitch) Attach(d *Appliance) {
   s.d = d
}
