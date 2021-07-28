package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alittlebrighter/thermostat"
	"github.com/alittlebrighter/thermostat/controller"
	"github.com/alittlebrighter/thermostat/models"
	"github.com/alittlebrighter/thermostat/thermometer"
	"github.com/alittlebrighter/thermostat/util"
	nats "github.com/nats-io/nats.go"
)

const (
	natsSub = "otto.sensor.temperature.current"
)

var (
	ConfigPath         = "/etc/thermostat.conf"
	ThermometerSvcName = "thermometer"
	natsURL            = nats.DefaultURL
)

func main() {
	flag.StringVar(&ConfigPath, "config", ConfigPath, "Path to the configuration file to use.")
	flag.StringVar(&ThermometerSvcName, "thermometerSvc", ThermometerSvcName, "mDNS service name for the thermometer.")
	flag.StringVar(&natsURL, "natsUrl", natsURL, "Url for NATS instance to connect to.")
	flag.Parse()

	log.Println("Starting thermostat.")

	/*
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Println("Could not connect to message bus.")
	}
	defer nc.Close()
	log.Println("Connected to NATS.")
	*/
	config, err := readState(ConfigPath)
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Println("Setting up controller.")
	control, err := controller.NewCentralController(config.Controller.Pins.Heat, config.Controller.Pins.Cool, config.Controller.Pins.Fan, 1*time.Minute)
	if err != nil {
		log.Fatalln("Error starting controller: " + err.Error())
	}
	control.Off()
	defer control.Shutdown()
	defer control.Off()

	thermometer, err := thermometer.NewRemote(config.Thermometer.Endpoint)
	if err != nil {
		log.Fatalln("Error getting thermometer instance: " + err.Error())
	}
	defer thermometer.Shutdown()
	log.Println("Got thermometer at " + config.Thermometer.Endpoint)

	log.Println("Initializing thermostat.")
	thermostatMain := config.Thermostat
	if _, ok := thermostatMain.Modes[thermostatMain.DefaultMode]; !ok {
		log.Fatalln("Invalid default mode.")
	}

	thermostatMain.Events = util.NewRingBuffer(60)
	thermostatMain.LastFan = time.Now()
	thermostatMain.SetController(control)
	thermostatMain.SetThermometer(thermometer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thermostatMain.Run(ctx) // fires a goroutine
	/*
	nc.Subscribe(natsSub, func(m *nats.Msg) {
		update := new(models.SensorUpdate)
		if err := json.Unmarshal(m.Data, update); err != nil {
			log.Println("could not parse update from NATS")
		}

		log.Println("got update from NATS: value:", update.Value)
		thermostatMain.ProcessTemperatureReading(update.Value.Degrees, update.Value.Unit)
	})
	*/
	
	http.HandleFunc("/", CORSFilterFactory(ConfigHandlerFactory(thermostatMain, config, cancel)))
	http.HandleFunc()

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
			thermostatMain.Run(cancel)
			go saveState(ConfigPath, config)
		}

		err := json.NewEncoder(w).Encode(thermostatMain)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "ERROR: could not marshal thermostat struct.")
			return
		}
	}
}

func CORSFilterFactory(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET,POST")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(200)
			return
		}

		handler(w, r)
	}
}
