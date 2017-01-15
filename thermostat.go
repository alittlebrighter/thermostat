package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	lowTemp  float64 = 70
	highTemp         = 80

	tempCheckInterval = 1 * time.Minute
)

func TempCToF(tempC float64) float64 {
	return tempC*9/5 + 32
}

func TempFToC(tempF float64) float64 {
	return (tempF - 32) * 5 / 9
}

type Thermostat struct {
	LowTemp, HighTemp, overshoot, TargetTemp float64
	pollInterval                             time.Duration
	control                                  *Controller
	thermometer                              *Thermometer
}

func NewThermostat(controls *Controller, meter *Thermometer, low, high, overshoot float64) *Thermostat {
	return &Thermostat{control: controls, thermometer: meter, LowTemp: low, HighTemp: high, overshoot: overshoot}
}

func (stat *Thermostat) ReadTemperature() (float64, error) {
	temp, err := stat.thermometer.ReadTemperature()
	if err != nil {
		return 0, err
	}

	return TempCToF(temp.CelsiusDeg), nil
}

func (stat *Thermostat) ProcessTemperatureReading(tempF float64) {
	log.Printf("Current Temperature (F): %f\n", tempF)
	switch {
	case (stat.control.Direction == Heating && tempF > stat.LowTemp+stat.overshoot) || (stat.control.Direction == Cooling && tempF < stat.HighTemp-stat.overshoot):
		log.Println("turning OFF")
		stat.control.Off()
	case tempF < stat.LowTemp:
		log.Println("turning on HEAT")
		stat.control.Heat()
	case tempF > stat.HighTemp:
		log.Println("turning on COOL")
		stat.control.Cool()
	default:
		log.Println("doing NOTHING")
	}
}

func (stat *Thermostat) Run() {
	ticker := time.NewTicker(tempCheckInterval)
	for {
		<-ticker.C

		tempF, err := stat.ReadTemperature()
		if err != nil {
			log.Println("Error reading Temperature: " + err.Error())
		} else {
			stat.ProcessTemperatureReading(tempF)
		}
	}
}

func main() {
	log.Println("Starting thermostat.")

	log.Println("Setting up controller.")
	control, err := NewController(16, 20, 21)
	if err != nil {
		log.Fatalln("Error starting controller: " + err.Error())
	}
	defer control.Shutdown()

	log.Println("Starting thermometer.")
	thermometer, err := NewThermometer()
	if err != nil {
		log.Fatalln("Error starting thermometer: " + err.Error())
	}
	defer thermometer.Shutdown()

	thermostat := NewThermostat(control, thermometer, lowTemp, highTemp, 2)

	tempF, err := thermostat.ReadTemperature()
	if err != nil {
		log.Println("Error reading Temperature: " + err.Error())
	} else {
		thermostat.ProcessTemperatureReading(tempF)
	}

	go thermostat.Run()

	tmpl, _ := template.New("windowForm").Parse(`<!DOCTYPE html>
<html>
<head>
    <title>Thermostat</title>
</head>
<body>
    <form action="/thermostat/window" method="POST">
        <input name="low" type="text" value="{{.LowTemp}}" />
        <input name="high" type="text" value="{{.HighTemp}}" />
        <button>Submit</button>
    </form>
</body>
</html>`)

	http.HandleFunc("/thermostat/window", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			low, lowErr := strconv.ParseFloat(r.PostFormValue("low"), 64)
			if lowErr != nil {
				low = thermostat.LowTemp
			}

			high, highErr := strconv.ParseFloat(r.PostFormValue("high"), 64)
			if highErr != nil {
				high = thermostat.HighTemp
			}

			log.Printf("Setting new temperature window. Low: %f, High: %f\n", low, high)
			thermostat.LowTemp = low
			thermostat.HighTemp = high
		}

		tmpl.Execute(w, thermostat)
	})

	log.Println("Starting web server.")
	log.Fatal(http.ListenAndServe(":80", nil))
}
