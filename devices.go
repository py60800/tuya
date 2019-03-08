// Copyright 2019 py60800. 
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   "encoding/json"
   "fmt"
   "log"
   "sync"
   "time"
   "net"
)

// data collected from broadcast
type pubAppliance struct {
   GwId       string
   Ip         string
   Active     int
   Ability    int
   Mode       int
   ProductKey string
//   Version    string
   Encrypt    bool
}

type Appliance struct {
   pubAppliance
   Version    string
   lastUpdate time.Time
   mutex      sync.RWMutex
   // Cnx 
   cnxStatus  int
   cnxSignal     *sync.Cond
   cnxMutex   sync.Mutex
   cnx        net.Conn
   // immutable fields
   name     string
   key      []byte
   device Device
}

func newAppliance() *Appliance {
   d := new(Appliance)
   d.cnxSignal = sync.NewCond(&d.cnxMutex)
   d.cnxStatus = 0
   d.Version = "3.1"
   return d
}
func (d *Appliance) GetName() string {
   return d.name
}
func (d *Appliance) GetDevice() Device {
   return d.device
}
func (d *Appliance) update(rd *pubAppliance) {
   d.mutex.Lock()
   defer d.mutex.Unlock()
   d.pubAppliance = *rd
   d.cnxSignal.Broadcast()
   d.lastUpdate = time.Now()
}
func (d *Appliance) String() string {
   d.mutex.RLock()
   defer d.mutex.RUnlock()
   return fmt.Sprintf("Dev[Name:%v Id:%v IP:%v]", d.name, d.GwId, d.Ip)
}

// configuration data
type confItem struct {
   Name string
   GwId string
   Type string
   Key  string
   Ip   string //optionnel
}

func (dm *DeviceManager) getAppliance(id string) *Appliance {
   d, ok := dm.collection[id]
   if !ok {
      d = newAppliance()
      dm.collection[id] = d
   }
   return d
}
func (dm *DeviceManager) configure(jdata string) {
   conf := make([]confItem, 0)
   err := json.Unmarshal([]byte(jdata), &conf)
   if err != nil {
      log.Fatal("Conf error:", err)
   }
   dm.mutex.Lock()
   defer dm.mutex.Unlock()
   for _, c := range conf {
      if len(c.GwId) == 0 {
         log.Fatal("Conf Id missing")
      }
      d := dm.getAppliance(c.GwId)
      d.GwId = c.GwId
      d.name = c.Name
      d.key = []byte(c.Key)
      if len(c.Ip) > 0 {
        d.Ip = c.Ip
      }
      b, ok := makeDevice(c.Type)
      if ok {
         b.Attach(d)
         d.device = b
         dm.namedColl[d.name] = b
      }
   }
}

type DeviceManager struct {
   collection map[string]*Appliance
   namedColl  map[string]Device
   mutex      sync.Mutex
}

func newDeviceManager() *DeviceManager {
   dm := new(DeviceManager)
   dm.collection = make(map[string]*Appliance)
   dm.namedColl = make(map[string]Device)
   go udpListener(dm)
   return dm
}
func NewDeviceManager(jdata string) *DeviceManager {
   dm := newDeviceManager()
   dm.configure(jdata)
   return dm
}
func (dm *DeviceManager) applianceUpdate(data []byte) {
   var rd pubAppliance
   je := json.Unmarshal(data, &rd)
   if je != nil {
      log.Print("JSON decode error:", je)
      return
   }
   dm.mutex.Lock()
   defer dm.mutex.Unlock()
   d := dm.getAppliance(rd.GwId)
   d.update(&rd)
}
func (dm *DeviceManager) Count() int {
   dm.mutex.Lock()
   defer dm.mutex.Unlock()
   return len(dm.collection)
}
func (dm *DeviceManager) ApplianceKeys() []string {
   dm.mutex.Lock()
   defer dm.mutex.Unlock()
   keys := make([]string, 0, len(dm.collection))
   for k := range dm.collection {
      keys = append(keys, k)
   }
   return keys
}
func (dm *DeviceManager) DeviceKeys() []string {
   dm.mutex.Lock()
   defer dm.mutex.Unlock()
   keys := make([]string, 0, len(dm.namedColl))
   for k := range dm.namedColl {
      keys = append(keys, k)
   }
   return keys
}
func (dm *DeviceManager) GetAppliance(key string) (*Appliance, bool) {
   dm.mutex.Lock()
   defer dm.mutex.Unlock()
   d, ok := dm.collection[key]
   return d, ok
}
func (dm *DeviceManager) GetDevice(key string) (Device, bool) {
   dm.mutex.Lock()
   defer dm.mutex.Unlock()
   b, ok := dm.namedColl[key]
   return b, ok
}
