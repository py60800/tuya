# tuya
Golang API for tuya devices (or the like)

Experimental code subject to major changes

## Limits
Support limited to protocol version 3.1

Tested only with Neo power plugs

## Standalone application

Primary as an example and for my one use I have released a small web application that uses this API.

The application can be installed on a Raspberry PI (for example).

You will find it there : [https://github.com/py60800/tinytuya](https://github.com/py60800/tinytuya)

## Acknowledgements
@codetheweb for reverse engineering the protocol

## Prerequisites

Collect the keys and the id of tuya devices according to [@Codetheweb method](https://github.com/codetheweb/tuyapi/blob/master/docs/SETUP.md)

## Reliability

The Neo devices I use (Tuya clone) behave as expected most of the time but I have experienced some random crash during development.

## Usage
Get the API `go get "github.com/py60800/tuya"`

Import 'github.com/py60800/tuya`

Create json configuration, thanks to keys and id collected previously (Use backquotes for multiline conf data or get it from a file)

`   conf := `[`
`    {"gwId":"1582850884f3eb30128e", 
`     "key":"XXXXXXXXXXX", `
`     "type":"Switch",`
`     "name":"sw1" }, `
`    {"gwId":"86273325cc50e3c8fe2d", `
`     "key":"XXXXXXXXXXX", `
`     "type":"Switch", `
`     "name":"sw2" } `
`     ]`

Create a device manager:
` dm := tuya.NewDeviceManager(conf)`

Get configured devices by their name `b1,ok := dm.GetDevice("sw1")`

Check type and cast to get active interface ` sw1  := b1.(tuya.Switch)`

Play with the device :

`sw1.Set(true)`  // doesn't wait for the result of the command

`sw1.SetW(5*time.Second)`  // ensure the command is properly done


`st,_ := sw1.Status()`

`fmt.Println("sw1 status:", st)` 

# Design considerations
IP addresses are collected automatically from UDP messages broadcast on port 6666

API is supposed to be thread safe (I hope). A device can be used by concurrent go coroutines however communication with each device are serialized (no more than one TCP connection)

Communication with Tuya device is asynchronous. This means that tuya device can notify a change if someone plays with the hardware switch. Naive implement may encounters issues while a expecting one request for each response.


# Extension
Looking at switch.go source code, it should be easy to create new devices using the same protocol.

Just define appropriate interface, code what is specific in a dedicated file and update the factory to make it usable.

# Notes


I have found many oddities in tuya protocol:

- Actual 64 bits encryption instead of 128 bits encryption (incorrect string to byte conversion)

- ECB encryption is weak

- Half of MD5 signing is used

- Useless prefixes and suffixes

- Worthless base64 encoding

- Protocol not properly layered (command outside payload)

By many aspects, it seems that the protocol was designed for serial line communication.

## Need help ?

Open an issue!
