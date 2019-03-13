// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   "sync"
   "sync/atomic"
)

type SyncMsg struct {
   Code int
   Dev  Device
}
type SyncChannel chan SyncMsg

type Device interface {
   Type() string
   Name() string
   Subscribe(SyncChannel) int64
   Unsubscribe(int64)
   // private 
   configure(*Appliance, *configurationData)
   processResponse(int, []byte)
}

// Code for tuya messages
const (
   CodeMsgSet        = 7
   CodeMsgStatus     = 10
   CodeMsgPing       = 9
   CodeMsgAutoStatus = 8
)

// to be embedded in Device
type baseDevice struct {
   sync.Mutex
   typ  string
   name string
   app  *Appliance
   // Publish subscribe
   subscribers map[int64]SyncChannel
}

// baseDevice initialization to be invoked during configation
func (b *baseDevice) _configure(typ string, a *Appliance, c *configurationData) {
   b.typ = typ
   b.app = a
   b.name = c.Name
   b.subscribers = make(map[int64]SyncChannel)
}

// Implementation of Device interface provided by baseDevice
func (b *baseDevice) Type() string {
   return b.typ
}
func (b *baseDevice) Name() string {
   return b.name
}
// Publish subscribe 
var keyIndexCpt int64

func MakeSyncChannel() SyncChannel {
   return SyncChannel(make(chan SyncMsg, 1))
}

func (b *baseDevice) Subscribe(c SyncChannel) int64 {
   b.Lock()
   defer b.Unlock()
   key := atomic.AddInt64(&keyIndexCpt, 1) // ignore overflow
   b.subscribers[key] = c
   return key
}
func (b *baseDevice) Unsubscribe(key int64) {
   b.Lock()
   defer b.Unlock()
   delete(b.subscribers, key)
}
func (b *baseDevice) Notify(code int, d Device) {
   b.Lock()
   defer b.Unlock()
   syncMsg := SyncMsg{code, d}
   for _, c := range b.subscribers {
      select {
      case c <- syncMsg:
      default:
      }
   }
}
