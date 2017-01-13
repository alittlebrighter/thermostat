package main

import (
	"log"
	"time"

	"github.com/alittlebrighter/embd"
	_ "github.com/alittlebrighter/embd/host/rpi"
	"github.com/alittlebrighter/embd/sensor/mcp9808"
)

func StartThermometer(interval time.Duration, cancel chan bool) (<-chan *mcp9808.Temperature, error) {
	bus := embd.NewI2CBus(1)
	thermometer, err := mcp9808.New(bus)
	if err != nil {
		return nil, err
	}

	thermometer.SetShutdownMode(false)
	thermometer.SetCriticalTempLock(false)
	thermometer.SetWindowTempLock(false)
	thermometer.SetAlertMode(false) // comparator output
	thermometer.SetInterruptClear(true)
	thermometer.SetAlertStatus(false)
	thermometer.SetAlertControl(false)
	thermometer.SetAlertSelect(false)
	thermometer.SetAlertPolarity(false)
	thermometer.SetTempResolution(mcp9808.EighthC)
	thermometer.SetTempHysteresis(mcp9808.Zero)

	tempOutput := make(chan *mcp9808.Temperature)

	go func() {
		defer embd.CloseI2C()
		defer thermometer.SetShutdownMode(true)

		ticker := time.NewTicker(interval)

		for {
			select {
			case <-cancel:
				ticker.Stop()
				break
			case <-ticker.C:
				go func() {
					tempOutput <- readTemperature(thermometer)
				}()
			}
		}
		close(tempOutput)
		return
	}()

	return tempOutput, nil
}

func readTemperature(thermometer *mcp9808.MCP9808) *mcp9808.Temperature {
	temperature, err := thermometer.AmbientTemp()
	if err != nil {
		log.Println(err)
	}
	return temperature
}
