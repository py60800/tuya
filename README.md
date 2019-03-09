# tuya
Golang API for tuya devices (or the like)

Experimental code subject to major changes

## Limits
Support limited to protocol version 3.1

Tested only with Neo power plugs

## Acknowledgements
@codetheweb for reverse engineering the protocol

## Prerequisites

Collect the keys and the id of tuya devices according to [@Codetheweb method](https://github.com/codetheweb/tuyapi/blob/master/docs/SETUP.md)

## Reliability

I have done my best to get the Neo devices that I have, work properly but it happens that a set command does not succeed. 

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

`sw1.Set(true)`

`st,_ := sw1.Status()`

`fmt.Println("sw1 status:", st)` 

# Design considerations
IP addresses are collected automatically from UDP messages broadcast on port 6666

API is supposed to be thread safe (I hope). A device can be used by concurrent go coroutines however communication with each device are serialized (no more than one TCP connection)

the API is synchroneous however it is safe to make it asynchroneous (i.e `go sw1.Set(true)` for a fire and forget usage). 

# Extension
Looking at switch.go source code, it should be easy to create new devices using the same protocol.

Just define appropriate interface, code what is specific in a dedicated file and update the factory to make it usable.

# Complains

I have found many oddities in tuya protocol:

- Actual 64 bits encryption instead of 128 bits encryption (incorrect string to byte conversion)

- ECB encryption is weak

- Half of MD5 signing is used

- Useless prefixes and suffixes

- Worthless base64 encoding

- Randomly encrypted responses

- Protocol not properly layered (command outside payload)

By many aspects, it seems that the protocol was designed for serial line communication.
Tuya guys should seriously consider refactoring their protocol (Perhaps is it on the way)
