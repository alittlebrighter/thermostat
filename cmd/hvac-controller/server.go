package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/stianeikeland/go-rpio"

	"github.com/alittlebrighter/thermostat/controller"
)

const DEFAULT_CONFIG = "/etc/thermostat.conf"

func main() {
	configFile := flag.String("config", DEFAULT_CONFIG, "The configuration file for the controller.")
	flag.Parse()

	if err := rpio.Open(); err != nil {
		log.Fatalln("ERROR: Can't open GPIOs.\n" + err.Error())
	}

	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalln("ERROR: Could not read configuration file!\n" + err.Error())
	}

	config := new(controllerConfig)
	if err = yaml.Unmarshal(data, config); err != nil {
		log.Fatalln("ERROR: Could not parse configuration!\n" + err.Error())
	}

	var fanCooldown time.Duration
	if fanCooldown, err = time.ParseDuration(config.FanCooldown); err != nil {
		fanCooldown = 1 * time.Minute
	}

	log.Println("Setting up controller.")
	control, err := controller.NewCentralController(config.Pins.Heat, config.Pins.Cool, config.Pins.Fan, fanCooldown)
	if err != nil {
		log.Fatalln("ERROR: Cannot start controller!\n" + err.Error())
	}
	control.Off()
	defer control.Shutdown()
	defer control.Off()

	appCtx := &appContext{control}

	http.HandleFunc("/heat", appCtx.controlElement("HEAT", appCtx.hvacControl.Heat))
	http.HandleFunc("/cool", appCtx.controlElement("AC", appCtx.hvacControl.Cool))
	http.HandleFunc("/fan", appCtx.controlElement("FAN", appCtx.hvacControl.Fan))

	log.Println("Starting web server at " + config.ServeAt)
	log.Fatal(http.ListenAndServe(config.ServeAt, nil))
}

type appContext struct {
	hvacControl controller.Controller
}

func (appCtx *appContext) controlElement(elementName string, turnOn func()) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := new(response)

		if strings.ToUpper(r.Method) == http.MethodPost {
			command := &response{Errors: []string{}}
			err := json.NewDecoder(r.Body).Decode(command)
			switch {
			case err != nil:
				log.Println("ERROR: " + err.Error())
				resp.Errors = append(resp.Errors, err.Error())
			case command.ElementOn:
				log.Println("Turning " + elementName + " ON.")
				turnOn()
			case !command.ElementOn:
				log.Println("Turning " + elementName + " OFF.")
				appCtx.hvacControl.Off()
			}
		}

		resp.ElementOn = appCtx.hvacControl.Direction() == controller.Heating

		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			log.Println("ERROR: " + err.Error())
		}
	}
}

func (appCtx *appContext) Shutdown(w http.ResponseWriter, r *http.Request) {
	if strings.ToUpper(r.Method) == http.MethodPost {
		appCtx.hvacControl.Shutdown()
		os.Exit(0)
	}
}

type response struct {
	ElementOn bool
	Errors    []string
}

type controllerConfig struct {
	ServeAt     string
	Pins        struct{ Fan, Cool, Heat int }
	FanCooldown string
}
