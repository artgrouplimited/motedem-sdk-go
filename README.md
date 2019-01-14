This SDK is for a BLE device "MOTEDEM" and target to work on Raspberry Pi

- [Official Site](https://motedem.com)
- [Kickstarter Campaign](https://www.kickstarter.com/projects/digitalcreations/motedem-infrared-blaster-with-sdk-for-raspberry-pi)

# Install
```bash
$ go get github.com/pkg/errors
$ go get github.com/go-ble/ble
```
```go
import "github.com/artgrouplimited/motedem-sdk-go"
```

# Usage
## Basic setup
```go
// Store all device connection
var devices map[string]*motedem.Device

func main() {
    // Setup on the Bluetooth LE
    // It must be call before anything BLE action
    motedem.BLESetup()

    devices = make(map[string]*motedem.Device, 0)

    // do your magic here

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)

    // Waiting system signal such like Ctrl + C
    <-c
    for _, d := range devices {
        // Disconnect all devices to have a graceful exit
        d.Disconnect()
    }
}
```

## Declare device
```go
device := motedem.NewDevice("8c:14:7d:00:00:00")
```
|Field Name|Type|Description|
|-|-|-|
|MAC|string|MAC address of motedem
|Connected|bool|Is it connecting
|Timeout|time.Duration|operation timeout

## Connect and Disconnect
```go
if err := device.Connect(); err != nil {
    // connect fail
}
if err := device.DisConnect(); err != nil {
    // disconnect fail
}
```
## Get temperature
```go
c, err := device.GetTemperature()
if err != nil {
    // err.Error()
    return
}

select {
case <-time.After(device.Timeout):
    // "Request timeout"
    return
case sensorData := <-c:
    fmt.Printf("Temperature: %f\n", sensorData.Temperature)
}
```
### Temperature Result
|Field Name|Type|Description|
|-|-|-|
|Temperature|float64|current temperature data of the sensor
## Learn IR Code
TODO: should we change SDK to long or short IR instead of AC / AV?
```go
c, err := device.LearnAV()
// or 
c, err := device.LearnAC()
if err != nil {
    // err.Error()
    return
}

select {
case <-time.After(device.Timeout):
    // "Request timeout"
    return
case learnData := <-c:
    if learnData.Success {
        if learnData.HaveData {
            fmt.Printf("Learn Data: %s\n", learnData.Data)
        } else {

        }
    } else {
        fmt.Printf("Learn failed\n")
    }
}
```
### LearnData
|Field Name|Type|Description|
|-|-|-|
|Success|bool|is learn operation success
|HaveData|bool|is there any data learnt
|Data|string|learn data in hex string