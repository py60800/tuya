package main

import (
	"log"
	"net/http"
	"text/template"
	//"github.com/py60800/tuya"
	"encoding/json"
	"time"
	"tuya"
        "flag"
"io/ioutil"
"strconv"
)

var tuyaConfig = `[
    {"gwId":"1582850884f3eb30128e",
     "key":"2869ef9b8c637e67",
     "type":"Switch",
     "name":"sw1" },
    {"gwId":"86273325cc50e3c8fe2d",
     "key":"aedd60a28f380b6a",
     "type":"Switch",
     "name":"sw2" }
     ]`

var tmpl *template.Template
var dm *tuya.DeviceManager

type button struct {
	Name    string
	Checked string
}

func setHandler(w http.ResponseWriter, r *http.Request) {
	sw := r.FormValue("switch")
	val := r.FormValue("set")
	log.Println("Set:", sw, val)
	s, ok := dm.GetDevice(sw)
	if ok {
		p := s.(tuya.Switch)
		if val == "on" || val == "true" {
			p.Set(true)
		} else {
			p.Set(false)
		}
	} else {
		log.Println("Unknown device :", sw)
	}
}
func getHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Get")

	// The code hereafter is not optimized
	// if there were many devices, an intermediate coroutine
	// should be used to avoid the burden and the load of
	// subscription/subscription

	keys := dm.DeviceKeys()
	devs := make([]tuya.Device, 0)
	skeys := make([]int64, 0)

	// Get the list of configured devices
	for _, v := range keys {
		b, _ := dm.GetDevice(v)
		devs = append(devs, b)
	}
	// Subscribe for events from these devices
	syncChannel := tuya.MakeSyncChannel()
	for _, b := range devs {
		skeys = append(skeys, b.Subscribe(syncChannel))
	}

	// Wait until data update or timeout
	select {
	case <-syncChannel:
	case <-time.After(time.Second * 15):
	}

	// cancel subscriptions
	for i := range skeys {
		devs[i].Unsubscribe(skeys[i])
	}

	// build the response
	result := make(map[string]interface{})
	for _, b := range devs {
		s, ok := b.(tuya.Switch)
		if ok {
			st, err := s.Status()
			t := make(map[string]interface{})
			t["Value"] = st
			if err == nil {
				t["Status"] = "OK"
			} else {
				t["Status"] = err.Error()
			}
			result[b.Name()] = t
		}
	}

	// send the response
	json, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}
func homeHandler(w http.ResponseWriter, r *http.Request) {
        
	tmpl.ExecuteTemplate(w, "header.tmpl", nil)
	keys := dm.DeviceKeys()
	for _, v := range keys {
		d := button{v, ""}
		b, _ := dm.GetDevice(v)
		if b.Type() == "Switch" {
			sw, _ := b.(tuya.Switch)

			if st, e := sw.Status(); e == nil {
				if st {
					d.Checked = "checked"
				} else {
					d.Checked = "unchecked"
				}
				log.Println(v, st)
			} else {
				log.Println(v, e)
			}
			tmpl.ExecuteTemplate(w, "button.tmpl", d)
		}
	}
	tmpl.ExecuteTemplate(w, "footer.tmpl", nil)

}
func main() {
        pconfig := flag.String("c","","configuration file")
        port    := flag.Int("p",8080,"Port number")
        flag.Parse()
        if len(*pconfig) == 0 {
		flag.PrintDefaults()
    		return
        }
	dm = tuya.NewDeviceManager(getConfig(*pconfig))
	var err error
	tmpl, err = template.ParseGlob("tmpl/*.tmpl")
	if err != nil {
		log.Fatal("Template error:", err)
	}
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/set", setHandler)
	http.HandleFunc("/get", getHandler)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static", fs))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func getConfig(tuyaconf string) string{
	b, err := ioutil.ReadFile(tuyaconf)
	if err != nil {
		log.Fatal("Cannot read:", tuyaconf)
	}
    return string(b)
}
