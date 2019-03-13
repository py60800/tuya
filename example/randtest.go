package main
import (
   "github.com/py60800/tuya"
   "time"
   "fmt"
   "math/rand"
)
func randFunc(sw tuya.Switch,n string){
    for i := 0 ; i < 50 ; i++{
      delay := time.Duration ( time.Millisecond * 
                          time.Duration(500+ rand.Int31n(5000)))
      time.Sleep(delay)
      v :=  (rand.Int() & 1) == 0
      st := time.Now()
      r,err := sw.SetW(v,time.Second*5)
      after := time.Now()
      if err == nil {
         fmt.Printf ("Set %v to %v OK(%v) in %v\n",n,v, r == v, after.Sub(st))
      }else{
         fmt.Printf ("Set %v to %v Failed(Err:%v) in %v\n",n,v,err,after.Sub(st))
      }
    }
}
func main(){
   var conf = `[
    {"gwId":"1582850884f3eb30128e",
     "key":"XXXXXXXXXXX",
     "type":"Switch",
     "name":"sw1" },
    {"gwId":"86273325cc50e3c8fe2d",
     "key":"XXXXXXXXXXX",
     "type":"Switch",
     "name":"sw2" }
     ]`
     

   dm := tuya.NewDeviceManager(conf)
   b1,_ := dm.GetDevice("sw1")
   sw1  := b1.(tuya.Switch)
   b2,_ := dm.GetDevice("sw2")
   sw2  := b2.(tuya.Switch)
   for i := 0 ; i < 5 ; i++ { 
        go randFunc(sw1,"sw1")
        go randFunc(sw2,"sw2")
   }
   time.Sleep(300*time.Second)
}
