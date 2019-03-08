// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   "fmt"
   "time"
)

type Switch interface {
   On() error
   Off() error
   Status() (bool, error)
   Set(bool) (bool, error)
}

type ISwitch struct {
   d      *Appliance
   status bool
}

const statusCmd = 10
const setCmd = 7

func (s *ISwitch) Set(on bool) (bool, error) {
   var er1 error
   var er2 error
   delay := time.Millisecond * 500
   for i := 0; i < 3; i++ {
      m := s.d.makeBaseMsg()
      m["dps"] = map[string]bool{"1": on}
      _, er1 = s.d.SendEncryptedCommand(7, m)
      var st bool
      time.Sleep(100*time.Millisecond)
      st, er2 = s.Status()
      switch {
         case er2 != nil:
            time.Sleep(delay)
            delay = delay * 2
         case st == on:
            return s.status, nil
         case st != on:
            fmt.Println("Invalid value :", on, st )
            return s.status, nil // fmt.Errorf("Invalid value")
      }
   }
   return s.status, fmt.Errorf("Set failed(%v/%v)", er1,er2)
}
func (s *ISwitch) On() error {
   _, err := s.Set(true)
   return err
}
func (s *ISwitch) Off() error {
   _, err := s.Set(false)
   return err
}
func (s *ISwitch) Status() (bool, error) {
   var err error
      var r map[string]interface{}
   for i := 0; i < 3; i++ {
      r, err = s.d.SendCommand(statusCmd, s.d.makeStatusMsg())
      if err == nil {
         v, ok := r["dps"]
         if ok {
            v1, ok2 := v.(map[string]interface{})
            if ok2 {
               v2, ok3 := v1["1"]
               if ok3 {
                  s.status, _ = v2.(bool) // supposed to be thread safe
                  return s.status, nil
               }
            }
         }else{
            fmt.Println(s.d.name,"Empty resp")
         }
         fmt.Println("Retry status")
      }
      s.d.resetCnx()
      time.Sleep(500 * time.Millisecond)
   } //for
   return false, fmt.Errorf("Bad response(%v/%v)", err,r)
}

// Device implementation
func (s *ISwitch) Type() string {
   return "Switch"
}
func (s *ISwitch) Attach(d *Appliance) {
   s.d = d
}
