package main

import (
	"log"
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
	lowTemp, highTemp, overshoot, targetTemp float64
	pollInterval                             time.Duration
	control                                  *Controller
	thermometer                              *Thermometer
}

func NewThermostat(controls *Controller, meter *Thermometer, low, high, overshoot float64) *Thermostat {
	return &Thermostat{control: controls, thermometer: meter, lowTemp: low, highTemp: high, overshoot: overshoot}
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
	case (stat.control.Direction == Heating && tempF > stat.targetTemp) || (stat.control.Direction == Cooling && tempF < stat.targetTemp):
		log.Println("turning OFF")
		stat.control.Off()
	case tempF < stat.lowTemp:
		log.Println("turning on HEAT")
		stat.control.Heat()
		stat.targetTemp = stat.lowTemp + stat.overshoot
	case tempF > stat.highTemp:
		log.Println("turning on COOL")
		stat.control.Cool()
		stat.targetTemp = stat.highTemp - stat.overshoot
	default:
		log.Println("doing NOTHING")
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

	ticker := time.NewTicker(tempCheckInterval)
	for {
		<-ticker.C

		tempF, err := thermostat.ReadTemperature()
		if err != nil {
			log.Println("Error reading Temperature: " + err.Error())
		} else {
			thermostat.ProcessTemperatureReading(tempF)
		}
	}
}
