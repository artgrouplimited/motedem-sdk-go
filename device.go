package motedem

import (
	"context"
	"encoding/hex"
	"math"
	"time"

	"github.com/go-ble/ble"
	"github.com/pkg/errors"
)

// Device presenting device after scan
type Device struct {
	MAC                 string
	client              ble.Client
	addr                ble.Addr
	profile             *ble.Profile
	Connected           bool
	dataNotification    []*notificationCallback
	controlNotification []*notificationCallback
	Timeout             time.Duration
}

func (d *Device) isConnected() bool {
	return d.client != nil && d.Connected
}

func (d *Device) disconnectCleanUp() {
	d.Connected = false
	d.client = nil
	d.profile = nil
}

// Connect a BLE device
func (d *Device) Connect() error {
	if d.isConnected() {
		return nil
	}

	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), d.Timeout))
	cln, err := ble.Dial(ctx, d.addr)

	if err != nil {
		return err
	}

	d.client = cln
	d.Connected = true
	go func() {
		<-cln.Disconnected()
		d.disconnectCleanUp()
	}()

	if err := d.sub(); err != nil {
		return err
	}

	return nil
}

// DisconnectSync function
func (d *Device) DisconnectSync() error {
	if !d.isConnected() {
		return nil
	}

	if err := d.client.CancelConnection(); err != nil {
		return err
	}
	<-d.client.Disconnected()
	d.disconnectCleanUp()
	return nil
}

// Disconnect a device
func (d *Device) Disconnect() error {
	if !d.isConnected() {
		return nil
	}

	return d.client.CancelConnection()
}

// Discover profile
func (d *Device) Discover() error {
	if err := d.Connect(); err != nil {
		return errors.Wrap(err, "can't discover")
	}

	if d.profile != nil {
		return nil
	}

	p, err := d.client.DiscoverProfile(true)
	if err != nil {
		return errors.Wrap(err, "can't discover profile")
	}

	d.profile = p
	return nil
}

// AddDataNotification function
func (d *Device) AddDataNotification(c *notificationCallback) {
	d.dataNotification = append(d.dataNotification, c)
}

// AddControlNotification function
func (d *Device) AddControlNotification(c *notificationCallback) {
	d.controlNotification = append(d.controlNotification, c)
}

func (d *Device) sub() error {
	if err := d.Discover(); err != nil {
		return errors.Wrap(err, "can't sub")
	}

	dataN := ble.NewCharacteristic(ble.MustParse("d6c12805-95e7-11e6-ae22-56b6b6499611"))
	controlN := ble.NewCharacteristic(ble.MustParse("d6c12807-95e7-11e6-ae22-56b6b6499611"))

	indication := false
	if u := d.profile.Find(controlN); u != nil {
		err := d.client.Subscribe(u.(*ble.Characteristic), indication, func(req []byte) {
			d.controlNotification = handleNotification(d.controlNotification, req)
		})
		if err != nil {
			return errors.Wrap(err, "can't subscribe to characteristic")
		}
	}

	if u := d.profile.Find(dataN); u != nil {
		err := d.client.Subscribe(u.(*ble.Characteristic), indication, func(req []byte) {
			d.dataNotification = handleNotification(d.dataNotification, req)
		})
		if err != nil {
			return errors.Wrap(err, "can't subscribe to characteristic")
		}
	}

	return nil
}

func handleNotification(notifications []*notificationCallback, req []byte) []*notificationCallback {
	dNs := []*notificationCallback{}
	for _, n := range notifications {
		if !n.Active {
			continue
		}
		if n.Filter == nil || n.Filter(req) {
			select {
			case n.Channel <- req:
				n.Life--
				if n.Life == 0 {
					n.Active = false
				}
			default:
				n.Active = false
			}
		}
		if n.Active {
			dNs = append(dNs, n)
		}
	}
	return dNs
}

// GetTemperature function
func (d *Device) GetTemperature() (chan SensorData, error) {
	if err := d.Discover(); err != nil {
		return nil, err
	}

	c := make(chan []byte)
	nC := &notificationCallback{
		Filter:  func(b []byte) bool { return b[0] == 0x71 },
		Active:  true,
		Channel: c,
		Life:    1,
	}
	d.AddDataNotification(nC)

	cc := make(chan SensorData)

	go func() {

		control := ble.NewCharacteristic(ble.MustParse("d6c12806-95e7-11e6-ae22-56b6b6499611"))
		data := ble.NewCharacteristic(ble.MustParse("d6c12804-95e7-11e6-ae22-56b6b6499611"))

		if u := d.profile.Find(control); u != nil {
			_ = d.client.WriteCharacteristic(u.(*ble.Characteristic), []byte{0x01}, false)
		}

		if u := d.profile.Find(data); u != nil {
			src := []byte{0x45, 0x34, 0x71, 0x04, 0xEE}
			_ = d.client.WriteCharacteristic(u.(*ble.Characteristic), src, false)
		}

		select {
		case <-time.After(d.Timeout):
			return
		case res := <-c:
			data := SensorData{
				Success:     false,
				HaveData:    false,
				Humidity:    0,
				Temperature: 0,
			}
			if int(res[2]) > 0x30 {
				cc <- data
				return
			}
			data.Success = true
			if int(res[1]) > 0x03 {
				data.HaveData = true
				data.Temperature = float64(int(res[3]&0x0F)<<8+int(res[4])) * 0.0625
			}
			cc <- data
		}
	}()

	return cc, nil
}

// LearnAV function
func (d *Device) LearnAV() (chan LearnData, error) {
	return d.learn([]byte{0x45, 0x34, 0x24, 0x04, 0xA1})
}

// LearnAC function
func (d *Device) LearnAC() (chan LearnData, error) {
	return d.learn([]byte{0x45, 0x34, 0x27, 0x04, 0xA4})
}

func (d *Device) learn(cmd []byte) (chan LearnData, error) {
	if err := d.Discover(); err != nil {
		return nil, err
	}

	cN := make(chan []byte)
	nC := &notificationCallback{
		Filter:  nil,
		Active:  true,
		Channel: cN,
		Life:    2,
	}
	d.AddControlNotification(nC)

	c := make(chan []byte)
	nC = &notificationCallback{
		Active:  true,
		Channel: c,
		Life:    2,
	}
	d.AddDataNotification(nC)

	cc := make(chan LearnData)

	go func() {

		control := ble.NewCharacteristic(ble.MustParse("d6c12806-95e7-11e6-ae22-56b6b6499611"))
		data := ble.NewCharacteristic(ble.MustParse("d6c12804-95e7-11e6-ae22-56b6b6499611"))

		if u := d.profile.Find(control); u != nil {
			_ = d.client.WriteCharacteristic(u.(*ble.Characteristic), []byte{0x01}, false)
		}

		if u := d.profile.Find(data); u != nil {
			_ = d.client.WriteCharacteristic(u.(*ble.Characteristic), cmd, false)
		}

		controlByte := <-cN
		select {
		case <-time.After(d.Timeout):
			return
		case res := <-c:
			if int(res[2]) > 0x30 {
				cc <- LearnData{
					Success:  false,
					HaveData: false,
					Data:     "",
				}
				return
			}
		}
		controlByte = <-cN
		nC.Life = int(controlByte[0])

		select {
		case <-time.After(d.Timeout):
			return
		case res := <-c:
			data := LearnData{
				Success:  false,
				HaveData: false,
				Data:     "",
			}
			if int(res[2]) > 0x30 {
				cc <- data
				return
			}
			data.Success = true
			if int(res[1]) > 0x03 {
				data.HaveData = true
				data.Data = hex.EncodeToString(res[3:])
				for nC.Life > 0 {
					select {
					case <-time.After(d.Timeout):
						return
					case res := <-c:
						data.Data = data.Data + hex.EncodeToString(res)
					}
				}
				data.Data = data.Data[:len(data.Data)-2]
			}
			cc <- data
		}
	}()

	return cc, nil
}

// EmitData function
func (d *Device) EmitData(irData string) error {
	if err := d.Discover(); err != nil {
		return err
	}

	control := ble.NewCharacteristic(ble.MustParse("d6c12806-95e7-11e6-ae22-56b6b6499611"))
	data := ble.NewCharacteristic(ble.MustParse("d6c12804-95e7-11e6-ae22-56b6b6499611"))

	irDataByte, err := hex.DecodeString(irData)

	if err != nil {
		return errors.Wrap(err, "Invalid IR Data")
	}

	size := byte(len(irDataByte) + 5)

	if u := d.profile.Find(control); u != nil {
		_ = d.client.WriteCharacteristic(u.(*ble.Characteristic), []byte{byte(math.Ceil(float64(size+1) / 20))}, false)
	}

	if u := d.profile.Find(data); u != nil {
		i := byte(0)
		j := byte(0)
		k := byte(0)
		m := byte(5)
		src := make([]byte, 20)
		for _, element := range []byte{0x45, 0x34, 0x25, size, 0x81} {
			src[i] = element
			j += element
			i++
		}
		for i < size {
			src[m] = irDataByte[k]
			j += irDataByte[k]
			k++
			i++
			m++
			if m == 20 {
				m = 0
				_ = d.client.WriteCharacteristic(u.(*ble.Characteristic), src, false)
			}
		}
		src[m] = j
		m++
		_ = d.client.WriteCharacteristic(u.(*ble.Characteristic), src[:m], false)
	}

	return nil
}
