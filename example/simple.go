package main
import (
   "github.com/py60800/tuya"
   "fmt"
   "os"
)
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
  
   toSet := true
   if len(os.Args) > 1{
      toSet = os.Args[1] == "on"
   }

   dm := tuya.NewDeviceManager(conf)
   b1,_ := dm.GetDevice("sw1")
   sw1  := b1.(tuya.Switch)

   r, err := sw1.Set(toSet)
   
   if err != nil {
      fmt.Println("exec error:",err)
   }else{
      fmt.Println("Success:",r)
   }
}
