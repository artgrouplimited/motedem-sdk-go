It is a simple web server which demo the usage of the SDK

# Usage
```bash
# Install
$ go get github.com/gorilla/mux
# Compile
$ go build -ldflags "-s -w"
# it require permission to access Bluetooth LE
$ sudo ./example
```

# API

#### GET /scan
```bash
$ curl localhost:8080/scan
[{"MAC":"8c:14:7d:00:00:00","RSSI":-24,"name":"IR MOTEDEM"}]
```
#### GET /device
```bash
$ curl localhost:8080/device
[{"MAC":"8c:14:7d:00:00:00","connected":true}]
```
#### GET /device/{mac}
```bash
$ curl localhost:8080/device/8c:14:7d:00:00:00
{"MAC":"8c:14:7d:00:00:00","connected":true}
```
#### GET /device/{mac}/connect
```bash
$ curl localhost:8080/device/8c:14:7d:00:00:00/connect
{"MAC":"8c:14:7d:00:00:00","connected":true}
```
#### GET /device/{mac}/disconnect
```bash
$ curl localhost:8080/device/8c:14:7d:00:00:00/disconnect
{"MAC":"8c:14:7d:00:00:00","connected":false}
```
#### GET /device/{mac}/emit/{irData}
```bash
$ curl localhost:8080/device/8c:14:7d:00:00:00/emit/93181133130001e4004cdf4edf300052004ce6575c41015301014d010100414f001203505310014c3000524c4e004e524c30523000000000000000000000000000000000000000000000000000000000
{"code":200,"message":"Emit"}
```
#### GET /device/{mac}/temperature
```bash
$ curl localhost:8080/device/8c:14:7d:00:00:00/temperature
{"MAC":"8c:14:7d:00:00:00","temperature":24.9375}
```
#### GET /device/{mac}/learnAV
```bash
$ curl localhost:8080/device/8c:14:7d:00:00:00/learnAV
{"MAC":"8c:14:7d:00:00:00","success":true,"data":"93181133130001e4004cdf4edf300052004ce6575c41015301014d010100414f001203505310014c3000524c4e004e524c30523000000000000000000000000000000000000000000000000000000000"}
# if fail or timeout
{"MAC":"8c:14:7d:00:00:00","success":false,"data":""}
```
#### GET /device/{mac}/learnAC
```bash
$ curl localhost:8080/device/8c:14:7d:00:00:00/learnAC
{"MAC":"8c:14:7d:00:00:00","success":true,"data":"922872128220a26d4c50975670302054704c5a01c9302d547000e9057d003c4f7000447f5321126d5212735e60125f635d51745222212221122222223401211221221221151212111111212222222122211222222216"}
# if fail or timeout
{"MAC":"8c:14:7d:00:00:00","success":false,"data":""}
```