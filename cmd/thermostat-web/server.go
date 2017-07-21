package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alittlebrighter/thermostat"
	"github.com/alittlebrighter/thermostat/controller"
	"github.com/alittlebrighter/thermostat/thermometer"
	"github.com/alittlebrighter/thermostat/util"
)

const DEFAULT_CONFIG = "/etc/thermostat.conf"

func main() {
	log.Println("Starting thermostat.")

	config, err := readState(DEFAULT_CONFIG)
	if err != nil {
		panic(err)
	}

	log.Println("Setting up controller.")
	control, err := controller.NewCentralController(config.Controller.Pins.Heat, config.Controller.Pins.Cool, config.Controller.Pins.Fan)
	if err != nil {
		log.Fatalln("Error starting controller: " + err.Error())
	}
	control.Off()
	defer control.Shutdown()
	defer control.Off()

	log.Println("Getting thermometer.")
	thermometer, err := thermometer.NewRemote(config.Thermometer.Endpoint)
	if err != nil {
		log.Fatalln("Error getting thermometer instance: " + err.Error())
	}
	defer thermometer.Shutdown()

	log.Println("Initializing thermostat.")
	thermostatMain := config.Thermostat
	if _, ok := thermostatMain.Modes[thermostatMain.DefaultMode]; !ok {
		log.Fatalln("Invalid default mode.")
	}

	thermostatMain.Events = util.NewRingBuffer(60)
	thermostatMain.LastFan = time.Now()
	thermostatMain.SetController(control)
	thermostatMain.SetThermometer(thermometer)

	cancel := make(chan bool)
	defer close(cancel)
	go thermostatMain.Run(cancel)

	http.HandleFunc("/", CORSFilterFactory(ConfigHandlerFactory(thermostatMain, config, cancel)))

	log.Println("Starting web server.")
	log.Fatal(http.ListenAndServe(config.ServeAt, nil))
}

// Config defines the configuration needed to run the thermostat.
type Config struct {
	thermostat.Config
	ServeAt string `json:"serveAt"`
}

func ConfigHandlerFactory(thermostatMain *thermostat.Thermostat, config *Config, cancel chan bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			newThermostat := new(thermostat.Thermostat)
			err := json.NewDecoder(r.Body).Decode(newThermostat)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprintf(w, "ERROR: "+err.Error())
				return
			}

			valid := newThermostat.Validate()
			if valid != "" {
				w.WriteHeader(422)
				fmt.Fprintf(w, "ERROR: invalid thermostat configuration. "+valid)
				return
			}

			thermostatMain.DefaultMode = newThermostat.DefaultMode
			thermostatMain.MaxErrors = newThermostat.MaxErrors
			thermostatMain.Modes = newThermostat.Modes
			thermostatMain.Overshoot = newThermostat.Overshoot
			thermostatMain.PollInterval = newThermostat.PollInterval
			thermostatMain.MinFan = newThermostat.MinFan
			thermostatMain.Schedule = newThermostat.Schedule
			thermostatMain.UnitPreference = newThermostat.UnitPreference

			cancel <- true
			go thermostatMain.Run(cancel)
			go saveState(DEFAULT_CONFIG, config)
		}

		err := json.NewEncoder(w).Encode(thermostatMain)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "ERROR: could not marshal thermostat struct.")
			return
		}

		w.WriteHeader(200)
	}
}

func CORSFilterFactory(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET,POST")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTION" {
			w.WriteHeader(200)
			return
		}

		handler(w, r)
	}
}
