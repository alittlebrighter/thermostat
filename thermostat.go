package main

import (
//	"bufio"
	"log"
//	"os"
	"time"
)

const (
	lowTemp  float64 = 68
	highTemp         = 80

	tempCheckInterval = 1 * time.Minute
)

func TempCToF(tempC float64) float64 {
	return tempC*9/5 + 32
}

func main() {
	log.Println("Starting thermostat.")

	cancel := make(chan bool)
	/*
	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		log.Println("Received cancel request.")
		cancel <- true
		return
	}()
	*/
	log.Println("Setting up controller.")
	control, err := StartController(16, 20, 21)
	if err != nil {
		close(cancel)
		log.Fatalln("Error starting controller: " + err.Error())
		return
	}

	log.Println("Starting thermometer.")
	temperatures, err := StartThermometer(tempCheckInterval, cancel)
	if err != nil {
		close(control)
		close(cancel)
		log.Fatalln("Error starting thermometer: " + err.Error())
		return
	}

	tempTarget := 0.0
	for temp := range temperatures {
		tempF := TempCToF(temp.CelsiusDeg)
		log.Printf("Current temperature: %fF\n", tempF)
		
		switch {
		case tempTarget != 0 && tempF > tempTarget-.5 && tempF < tempTarget +.5:
			log.Println("turning OFF")
			control <- Off
			tempTarget = 0
		case tempF < lowTemp:
			log.Println("turning on HEAT")
			control <- Heat
			tempTarget = lowTemp + 2
		case tempF > highTemp:
			log.Println("turning on COOL")
			control <- Cool
			tempTarget = highTemp - 2
		default:
			log.Println("doing NOTHING")
		}
	}
	close(control)
}
