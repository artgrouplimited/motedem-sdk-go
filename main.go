package motedem

import (
	"context"

	"strings"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
	"github.com/pkg/errors"
)

// SensorData struct
type SensorData struct {
	Success     bool
	HaveData    bool
	Temperature float64
	Humidity    float64
}

// LearnData struct
type LearnData struct {
	Success  bool
	HaveData bool
	Data     string
}

// ScanResult struct
type ScanResult struct {
	MAC  string `json:"MAC"`
	RSSI int    `json:"RSSI"`
	Name string `json:"name"`
}

// BLESetup must have to call at start of program
func BLESetup() error {
	d, err := dev.NewDevice("default")
	if err != nil {
		return errors.Wrap(err, "can't new device")
	}
	ble.SetDefaultDevice(d)

	return nil
}

// ScanDevice BLE scan for a duration in Millisecond
func ScanDevice(duration int) ([]ScanResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(duration)*time.Millisecond)
	ctx = ble.WithSigHandler(ctx, cancel)
	allowDup := true
	advsMap := map[string]ble.Advertisement{}
	aH := func(a ble.Advertisement) {
		advsMap[a.Addr().String()] = a
		// cancel()
	}
	filter := func(a ble.Advertisement) bool {
		name := a.LocalName()
		if len(name) > 0 && strings.HasPrefix(name, "IR MOTEDEM") {
			return true
		}
		return false
	}

	err := ble.Scan(ctx, allowDup, aH, filter)
	if err == nil {
		<-ctx.Done()
	}

	advs := []ScanResult{}
	for _, adv := range advsMap {
		advs = append(advs, ScanResult{
			MAC:  adv.Addr().String(),
			Name: adv.LocalName(),
			RSSI: adv.RSSI(),
		})
	}

	return advs, err
}

// NewDevice function
func NewDevice(MAC string) *Device {
	return &Device{
		MAC:       MAC,
		addr:      ble.NewAddr(MAC),
		Connected: false,
	}
}

// notificationCallback generic callback of notification
type notificationCallback struct {
	Channel chan []byte
	Active  bool
	Filter  func([]byte) bool
	Life    int
}
