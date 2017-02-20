package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alittlebrighter/thermostat/controller"
	"github.com/alittlebrighter/thermostat/thermometer"
	"github.com/alittlebrighter/thermostat/util"
)

func main() {
	log.Println("Starting thermostat.")

	config, err := readState("/etc/thermostat.conf")
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
	thermometer, err := thermometer.NewJSONWebService(config.Thermometer.Endpoint)
	if err != nil {
		log.Fatalln("Error getting thermometer instance: " + err.Error())
	}
	defer thermometer.Shutdown()

	log.Println("Initializing thermostat.")
	thermostat := config.Thermostat
	if _, ok := thermostat.Modes[thermostat.DefaultMode]; !ok {
		log.Fatalln("Invalid default mode.")
	}

	thermostat.Events = util.NewRingBuffer(60)
	thermostat.control = control
	thermostat.thermometer = thermometer

	cancel := make(chan bool)
	defer close(cancel)
	go thermostat.Run(cancel)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			newThermostat := new(Thermostat)
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

			thermostat.DefaultMode = newThermostat.DefaultMode
			thermostat.MaxErrors = newThermostat.MaxErrors
			thermostat.Modes = newThermostat.Modes
			thermostat.Overshoot = newThermostat.Overshoot
			thermostat.PollInterval = newThermostat.PollInterval
			thermostat.MinFan = newThermostat.MinFan
			thermostat.Schedule = newThermostat.Schedule
			thermostat.UnitPreference = newThermostat.UnitPreference

			cancel <- true
			go thermostat.Run(cancel)
		}

		err := json.NewEncoder(w).Encode(thermostat)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "ERROR: could not marshal thermostat struct.")
			return
		}

		//log.Printf("Last Event: %f, %s", thermostat.Events.GetLast().AmbientTemperature, thermostat.Events.GetLast().Direction.String())
	})

	log.Println("Starting web server.")
	log.Fatal(http.ListenAndServe(config.ServeAt, nil))
}

type Config struct {
	Thermostat  *Thermostat
	Controller  struct{ Pins struct{ Fan, Cool, Heat int } }
	Thermometer struct{ Type, Endpoint string }
	ServeAt     string `json:"serveAt"`
}

// Validate checks that a thermostat has a valid configuration and returns a string explaining any issues.  An empty string denotes a valid configuration.
func (stat *Thermostat) Validate() string {
	if _, ok := stat.Modes[stat.DefaultMode]; !ok {
		return "DefaultMode definition not found!"
	}

	for key, window := range stat.Modes {
		if window.LowTemp >= window.HighTemp {
			return fmt.Sprintf("%s mode is not valid.", key)
		}
	}

	for i, spec := range stat.Schedule {
		if time.Time(spec.Start).Unix() >= time.Time(spec.End).Unix() {
			return fmt.Sprintf("Schedule entry #%d not valid.", i+1)
		} else if _, ok := stat.Modes[spec.ModeName]; !ok {
			return fmt.Sprintf("Schedule entry #%d not valid.", i+1)
		}
	}

	return ""
}
