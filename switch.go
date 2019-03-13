// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   //   "fmt"
   "encoding/json"
   "errors"
   "log"
   "sync/atomic"
   "time"
)

type Switch interface {
   Set(bool) error
   SetW(bool, time.Duration) (bool, error)
   Status() (bool, error)
   StatusW(time.Duration) (bool, error)
}

const (
   SwitchOff          = 0
   SwitchOn           = 1
   SwitchUndetermined = 2
)

type ISwitch struct {
   baseDevice
   status int32
}

func (s *ISwitch) Set(on bool) error {
   m := s.app.makeBaseMsg()
   m["dps"] = map[string]bool{"1": on}
   return s.app.SendEncryptedCommand(CodeMsgSet, m)
}
func (s *ISwitch) SetW(on bool, delay time.Duration) (bool, error) {
   c := MakeSyncChannel()
   k := s.Subscribe(c)
   defer s.Unsubscribe(k)
   deadLine := time.Now().Add(delay)
   err := s.Set(on)
   if err != nil {
      return s._status(), err
   }
   for {
      select {
      case <-c:
         // Ignore Code :
         if on == (int32(atomic.LoadInt32(&s.status)) != 0) {
            return on, nil
         }
      case <-time.After(deadLine.Sub(time.Now())):
         return s._status(), errors.New("Timeout")
      }
   }
}

func (s *ISwitch) Status() (bool, error) {
   switch int(atomic.LoadInt32(&s.status)) {
   case 0:
      return false, nil
   case 1:
      return true, nil
   default:
      return false, errors.New("Undetermined")
   }
}
func (s *ISwitch) _status() bool {
   return atomic.LoadInt32(&s.status) != 0
}
func (s *ISwitch) StatusW(delay time.Duration) (bool, error) {
   c := MakeSyncChannel()
   k := s.Subscribe(c)
   defer s.Unsubscribe(k)

   deadLine := time.Now().Add(delay)
   err := s.app.SendCommand(CodeMsgStatus, s.app.makeStatusMsg())
   if err != nil {
      return s._status(), err
   }
   for {
      select {
      case synMsg := <-c:
         if synMsg.Code == CodeMsgStatus ||
            synMsg.Code == CodeMsgAutoStatus {
            s, e := s.Status()
            return s, e
         }
      case <-time.After(deadLine.Sub(time.Now())):
         return s._status(), errors.New("Timeout")
      }
   }

}
func (s *ISwitch) processResponse(code int, data []byte) {
   switch {
   case len(data) == 0:
      return
   case code == 7:
      return
   case code == 9:
      return
   }
   var r map[string]interface{}
   //fmt.Println(code, string(data))
   err := json.Unmarshal(data, &r)
   if err != nil {
      log.Println("JSON decode error")
      return
   }
   atomic.StoreInt32(&s.status, SwitchUndetermined)
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
   s.Notify(code, s)
}

// Device implementation
func (s *ISwitch) configure(a *Appliance, c *configurationData) {
   s.status = SwitchUndetermined
   s._configure("Switch", a, c)
}
