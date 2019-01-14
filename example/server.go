package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	".."
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// main function
func main() {
	motedem.BLESetup()

	devices = make(map[string]*motedem.Device, 0)

	r := mux.NewRouter()

	r.HandleFunc("/scan", scan).Methods("GET")
	r.HandleFunc("/device", allStatus).Methods("GET")
	r.HandleFunc("/device/{mac}", status).Methods("GET")
	r.HandleFunc("/device/{mac}/connect", connect).Methods("GET")
	r.HandleFunc("/device/{mac}/disconnect", disconnect).Methods("GET")
	r.HandleFunc("/device/{mac}/emit/{irData}", emit).Methods("GET")
	r.HandleFunc("/device/{mac}/temperature", temperature).Methods("GET")
	r.HandleFunc("/device/{mac}/learnAV", learnAV).Methods("GET")
	r.HandleFunc("/device/{mac}/learnAC", learnAC).Methods("GET")

	log.SetFlags(log.Lmicroseconds)

	srv := &http.Server{
		Addr: "0.0.0.0:8080",
		// Good practice to set timeouts to avoid Slowloris attacks.
		// WriteTimeout: time.Second * 15,
		// ReadTimeout:  time.Second * 15,
		// IdleTimeout:  time.Second * 60,
		Handler: r, // Pass our instance of gorilla/mux in.
	}
	go func() {
		log.Println("Start HTTP service")
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)

	<-c
	for _, d := range devices {
		d.Disconnect()
	}
	log.Println("Disconnect all BLE connections")
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}

// ErrorResult struct
type ErrorResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// StatusResult struct
type StatusResult struct {
	MAC       string `json:"MAC"`
	Connected bool   `json:"connected"`
	// RSSI      int    `json:"RSSI"` // go-ble not implement it for linux yet
}

// ScanResult struct
type ScanResult struct {
	MAC  string `json:"MAC"`
	RSSI int    `json:"RSSI"`
	Name string `json:"name"`
}

// TemperatureResult struct
type TemperatureResult struct {
	MAC         string  `json:"MAC"`
	Temperature float64 `json:"temperature"`
}

// LearnDataResult struct
type LearnDataResult struct {
	MAC     string `json:"MAC"`
	Success bool   `json:"success"`
	Data    string `json:"data"`
}

func responseError(w http.ResponseWriter, code int, message string) {
	errorJSON, _ := json.Marshal(ErrorResult{Code: code, Message: message})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(errorJSON)
}

func scan(w http.ResponseWriter, r *http.Request) {
	log.Println("Request Scan")
	duration, err := strconv.Atoi(r.URL.Query().Get("duration"))
	if err != nil {
		duration = 5000
	} else if duration <= 0 {
		responseError(
			w,
			http.StatusBadRequest,
			"Duration should be between 0 and 60000",
		)
		return
	}

	advc, _ := motedem.ScanDevice(duration)

	scanResults := []ScanResult{}
	for _, adv := range advc {
		scanResults = append(scanResults, ScanResult{
			MAC:  adv.MAC,
			Name: adv.Name,
			RSSI: adv.RSSI,
		})
	}
	json.NewEncoder(w).Encode(scanResults)
}

func allStatus(w http.ResponseWriter, r *http.Request) {
	var statusResults []StatusResult
	for _, device := range devices {
		statusResults = append(statusResults, getStatus(device))
	}

	json.NewEncoder(w).Encode(statusResults)
}

func getStatus(device *motedem.Device) StatusResult {
	var statusResult StatusResult
	statusResult.MAC = device.MAC
	statusResult.Connected = device.Connected == true
	if statusResult.Connected {
		// statusResult.RSSI = device.client.ReadRSSI()
	}
	return statusResult
}

var devices map[string]*motedem.Device

func getDevice(r *http.Request, defaultDuration int) (*motedem.Device, error) {
	vars := mux.Vars(r)
	mac := vars["mac"]
	// mac = strings.Replace(mac, ":", "", -1)
	_, exists := devices[mac]
	if !exists {
		devices[mac] = motedem.NewDevice(mac)
	}

	duration, err := strconv.Atoi(r.URL.Query().Get("duration"))
	if err != nil {
		duration = defaultDuration
	} else if duration <= 0 {
		return nil, errors.New("Duration should be between 0 and 60000")
	}
	devices[mac].Timeout = time.Duration(duration) * time.Millisecond

	return devices[mac], nil
}

func status(w http.ResponseWriter, r *http.Request) {
	device, err := getDevice(r, 5000)
	if err != nil {
		responseError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.NewEncoder(w).Encode(getStatus(device))
}

func emit(w http.ResponseWriter, r *http.Request) {
	log.Println("Request emit")
	device, err := getDevice(r, 5000)
	if err != nil {
		responseError(w, http.StatusBadRequest, err.Error())
		return
	}

	vars := mux.Vars(r)
	irData := vars["irData"]
	if err := device.EmitData(irData); err != nil {
		responseError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	responseError(
		w,
		http.StatusOK,
		"Emit",
	)
}

func connect(w http.ResponseWriter, r *http.Request) {
	log.Println("Request connect")
	device, err := getDevice(r, 5000)
	if err != nil {
		responseError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := device.Connect(); err != nil {
		responseError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	statusResult := getStatus(device)
	json.NewEncoder(w).Encode(statusResult)
}

func disconnect(w http.ResponseWriter, r *http.Request) {
	log.Println("Request disconnect")
	device, err := getDevice(r, 5000)
	if err != nil {
		responseError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := device.DisconnectSync(); err != nil {
		responseError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	statusResult := getStatus(device)
	json.NewEncoder(w).Encode(statusResult)
}

func temperature(w http.ResponseWriter, r *http.Request) {
	device, err := getDevice(r, 5000)
	if err != nil {
		responseError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Println("Request temperature")
	c, err := device.GetTemperature()
	if err != nil {
		responseError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	select {
	case <-time.After(device.Timeout):
		responseError(w, http.StatusRequestTimeout, "Request timeout")
		return
	case sensorData := <-c:
		json.NewEncoder(w).Encode(TemperatureResult{
			MAC:         device.MAC,
			Temperature: sensorData.Temperature,
		})
	}
}

func learnAV(w http.ResponseWriter, r *http.Request) {
	device, err := getDevice(r, 20000)
	if err != nil {
		responseError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Println("Request LearnAV")
	c, err := device.LearnAV()
	if err != nil {
		responseError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	select {
	case <-time.After(device.Timeout):
		responseError(w, http.StatusRequestTimeout, "Request timeout")
		return
	case learnData := <-c:
		if learnData.Success {
			if learnData.HaveData {
				fmt.Printf("Learn Data: %x\n", learnData.Data)
			} else {
			}
		} else {
			fmt.Print("Learn Data fail\n")
		}
		json.NewEncoder(w).Encode(LearnDataResult{
			MAC:     device.MAC,
			Success: learnData.Success,
			Data:    learnData.Data,
		})
	}
}

func learnAC(w http.ResponseWriter, r *http.Request) {
	device, err := getDevice(r, 20000)
	if err != nil {
		responseError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Println("Request LearnAC")
	c, err := device.LearnAC()
	if err != nil {
		responseError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	select {
	case <-time.After(device.Timeout):
		responseError(w, http.StatusRequestTimeout, "Request timeout")
		return
	case learnData := <-c:
		if learnData.Success {
			if learnData.HaveData {
				fmt.Printf("Learn Data: %x\n", learnData.Data)
			} else {
			}
		} else {
			fmt.Print("Learn Data fail\n")
		}
		json.NewEncoder(w).Encode(LearnDataResult{
			MAC:     device.MAC,
			Success: learnData.Success,
			Data:    learnData.Data,
		})
	}
}
